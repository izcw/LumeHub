package api

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lumehub/internal/auth"
	"lumehub/internal/model"
	"lumehub/internal/store"
)

const (
	testPasskeyPassword = "passkey-test-123"
	testPasskeyEmail    = "passkey@test.local"
)

func setupPasskeyAPITest(t *testing.T) *httptest.Server {
	t.Helper()

	dir := t.TempDir()
	doc := model.AccountsDoc{
		Version: 1,
		Accounts: []model.Account{
			{
				ID:           "u-passkey-test",
				Username:     "passkeyuser",
				Email:        testPasskeyEmail,
				PasswordHash: model.HashPasswordUTF8(testPasskeyPassword),
				DisplayName:  "Passkey Tester",
			},
		},
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal accounts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "accounts.json"), raw, 0o644); err != nil {
		t.Fatalf("write accounts: %v", err)
	}

	st := store.New(dir)
	authMgr := auth.New(st, "")
	h := NewHandler(st, authMgr, log.Default())
	mux := http.NewServeMux()
	h.Register(mux)
	return httptest.NewServer(mux)
}

func loginSessionCookie(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	body := bytes.NewBufferString(`{"email":"` + testPasskeyEmail + `","password":"` + testPasskeyPassword + `"}`)
	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json", body)
	if err != nil {
		t.Fatalf("login post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("login status=%d body=%s", resp.StatusCode, string(b))
	}
	for _, c := range resp.Cookies() {
		if c.Name == "lumehub_session" && c.Value != "" {
			return c.Value
		}
	}
	t.Fatal("login cookie missing")
	return ""
}

func authReq(t *testing.T, method, url, cookie string, body io.Reader, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: "lumehub_session", Value: cookie})
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, raw
}

func b64url(raw []byte) string {
	return base64.RawURLEncoding.EncodeToString(raw)
}

func makeAuthData(rpID string, signCount uint32, flags byte) []byte {
	hash := sha256.Sum256([]byte(rpID))
	out := make([]byte, 37)
	copy(out[:32], hash[:])
	out[32] = flags
	binary.BigEndian.PutUint32(out[33:37], signCount)
	return out
}

func makeClientDataJSON(t *testing.T, typ, challenge, origin string) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]string{
		"type":      typ,
		"challenge": challenge,
		"origin":    origin,
	})
	if err != nil {
		t.Fatalf("marshal clientData: %v", err)
	}
	return raw
}

func generateES256Key(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal pubkey: %v", err)
	}
	return key, b64url(pubDER)
}

func signAssertion(t *testing.T, key *ecdsa.PrivateKey, authData, clientDataRaw []byte) string {
	t.Helper()
	cdHash := sha256.Sum256(clientDataRaw)
	msg := append(append([]byte(nil), authData...), cdHash[:]...)
	msgHash := sha256.Sum256(msg)
	sig, err := ecdsa.SignASN1(rand.Reader, key, msgHash[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return b64url(sig)
}

func TestInferOriginAndRPID_PrefersBrowserOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:5353/api/auth/passkey/register/options", nil)
	req.Host = "127.0.0.1:5353"
	req.Header.Set("Origin", "http://localhost:5173")

	origin, rpID := inferOriginAndRPID(req)
	if origin != "http://localhost:5173" {
		t.Fatalf("origin = %q, want http://localhost:5173", origin)
	}
	if rpID != "localhost" {
		t.Fatalf("rpID = %q, want localhost", rpID)
	}
}

func TestInferOriginAndRPID_FallbackToHostWhenNoOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://192.168.0.87:5353/api/auth/passkey/register/options", nil)
	req.Host = "192.168.0.87:5353"

	origin, rpID := inferOriginAndRPID(req)
	if origin != "http://192.168.0.87:5353" {
		t.Fatalf("origin = %q, want http://192.168.0.87:5353", origin)
	}
	if rpID != "192.168.0.87" {
		t.Fatalf("rpID = %q, want 192.168.0.87", rpID)
	}
}

func TestPasskeyRegisterVerifyUsesBrowserOriginBehindDevProxy(t *testing.T) {
	srv := setupPasskeyAPITest(t)
	defer srv.Close()

	cookie := loginSessionCookie(t, srv)
	browserOrigin := "http://localhost:5173"
	proxyHeaders := map[string]string{
		"Origin": browserOrigin,
		"Host":   "127.0.0.1:5353",
	}

	resp, raw := authReq(t, http.MethodPost, srv.URL+"/api/auth/passkey/register/options", cookie,
		bytes.NewBufferString(`{"label":"测试通行证"}`), proxyHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("options status=%d body=%s", resp.StatusCode, string(raw))
	}

	var options map[string]any
	if err := json.Unmarshal(raw, &options); err != nil {
		t.Fatalf("decode options: %v", err)
	}
	challenge, _ := options["challenge"].(string)
	if challenge == "" {
		t.Fatalf("missing challenge in options: %s", string(raw))
	}
	rp, _ := options["rp"].(map[string]any)
	rpID, _ := rp["id"].(string)
	if rpID != "localhost" {
		t.Fatalf("rp.id = %q, want localhost", rpID)
	}

	credID := b64url([]byte("test-credential-id"))
	clientDataRaw := makeClientDataJSON(t, "webauthn.create", challenge, browserOrigin)
	authData := makeAuthData(rpID, 0, 0x05) // UP + UV
	_, pubKey := generateES256Key(t)

	verifyBody, _ := json.Marshal(map[string]any{
		"credentialId":      credID,
		"publicKey":         pubKey,
		"algorithm":         -7,
		"clientDataJSON":    b64url(clientDataRaw),
		"authenticatorData": b64url(authData),
		"transports":        []string{"internal"},
		"label":             "测试通行证",
	})
	resp, raw = authReq(t, http.MethodPost, srv.URL+"/api/auth/passkey/register/verify", cookie,
		bytes.NewReader(verifyBody), proxyHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", resp.StatusCode, string(raw))
	}

	resp, raw = authReq(t, http.MethodGet, srv.URL+"/api/auth/passkey/list", cookie, nil, proxyHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d body=%s", resp.StatusCode, string(raw))
	}
	var list struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("passkey count = %d, want 1", len(list.Items))
	}
}

func TestPasskeyRegisterVerifyFailsWhenOriginMismatchWithoutBrowserHeader(t *testing.T) {
	srv := setupPasskeyAPITest(t)
	defer srv.Close()

	cookie := loginSessionCookie(t, srv)
	// 模拟旧行为：代理改写 Host，且请求未带 Origin。
	resp, raw := authReq(t, http.MethodPost, srv.URL+"/api/auth/passkey/register/options", cookie,
		bytes.NewBufferString(`{"label":"测试通行证"}`), map[string]string{"Host": "127.0.0.1:5353"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("options status=%d body=%s", resp.StatusCode, string(raw))
	}

	var options map[string]any
	_ = json.Unmarshal(raw, &options)
	challenge, _ := options["challenge"].(string)
	rp, _ := options["rp"].(map[string]any)
	rpID, _ := rp["id"].(string)

	browserOrigin := "http://localhost:5173"
	clientDataRaw := makeClientDataJSON(t, "webauthn.create", challenge, browserOrigin)
	authData := makeAuthData(rpID, 0, 0x05)
	_, pubKey := generateES256Key(t)

	verifyBody, _ := json.Marshal(map[string]any{
		"credentialId":      b64url([]byte("mismatch-cred")),
		"publicKey":         pubKey,
		"algorithm":         -7,
		"clientDataJSON":    b64url(clientDataRaw),
		"authenticatorData": b64url(authData),
		"label":             "测试通行证",
	})
	resp, raw = authReq(t, http.MethodPost, srv.URL+"/api/auth/passkey/register/verify", cookie,
		bytes.NewReader(verifyBody), map[string]string{"Host": "127.0.0.1:5353"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("verify status=%d, want 400 body=%s", resp.StatusCode, string(raw))
	}
	if !strings.Contains(string(raw), "origin mismatch") {
		t.Fatalf("expected origin mismatch, got: %s", string(raw))
	}
}

func TestPasskeyQRLoginAfterRegister(t *testing.T) {
	srv := setupPasskeyAPITest(t)
	defer srv.Close()

	cookie := loginSessionCookie(t, srv)
	browserOrigin := "http://localhost:5173"
	proxyHeaders := map[string]string{
		"Origin": browserOrigin,
		"Host":   "127.0.0.1:5353",
	}

	key, pubKey := generateES256Key(t)
	credID := b64url([]byte("qr-login-credential"))

	// 1) 绑定通行证
	resp, raw := authReq(t, http.MethodPost, srv.URL+"/api/auth/passkey/register/options", cookie,
		bytes.NewBufferString(`{"label":"扫码登录测试"}`), proxyHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register options status=%d body=%s", resp.StatusCode, string(raw))
	}
	var regOptions map[string]any
	_ = json.Unmarshal(raw, &regOptions)
	regChallenge, _ := regOptions["challenge"].(string)
	rp, _ := regOptions["rp"].(map[string]any)
	rpID, _ := rp["id"].(string)

	regClientData := makeClientDataJSON(t, "webauthn.create", regChallenge, browserOrigin)
	regAuthData := makeAuthData(rpID, 0, 0x05)
	regVerifyBody, _ := json.Marshal(map[string]any{
		"credentialId":      credID,
		"publicKey":         pubKey,
		"algorithm":         -7,
		"clientDataJSON":    b64url(regClientData),
		"authenticatorData": b64url(regAuthData),
		"label":             "扫码登录测试",
	})
	resp, raw = authReq(t, http.MethodPost, srv.URL+"/api/auth/passkey/register/verify", cookie,
		bytes.NewReader(regVerifyBody), proxyHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register verify status=%d body=%s", resp.StatusCode, string(raw))
	}

	// 2) 创建扫码登录会话
	resp, raw = authReq(t, http.MethodPost, srv.URL+"/api/auth/qr/session", "",
		bytes.NewBufferString("{}"), proxyHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("qr session status=%d body=%s", resp.StatusCode, string(raw))
	}
	var qrSession map[string]any
	_ = json.Unmarshal(raw, &qrSession)
	sessionID, _ := qrSession["sessionId"].(string)
	qrLoginURL, _ := qrSession["qrLoginUrl"].(string)
	if sessionID == "" || qrLoginURL == "" {
		t.Fatalf("invalid qr session: %s", string(raw))
	}
	ticket := strings.TrimPrefix(qrLoginURL[strings.LastIndex(qrLoginURL, "t="):], "t=")

	// 3) 获取扫码通行证的 assertion options
	optBody, _ := json.Marshal(map[string]string{"ticket": ticket})
	resp, raw = authReq(t, http.MethodPost, srv.URL+"/api/auth/qr/passkey/options", "",
		bytes.NewReader(optBody), proxyHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("qr passkey options status=%d body=%s", resp.StatusCode, string(raw))
	}
	var qrOptions map[string]any
	_ = json.Unmarshal(raw, &qrOptions)
	loginChallenge, _ := qrOptions["challenge"].(string)
	if loginChallenge == "" {
		t.Fatalf("missing qr challenge: %s", string(raw))
	}

	// 4) 模拟手机确认通行证
	loginClientData := makeClientDataJSON(t, "webauthn.get", loginChallenge, browserOrigin)
	loginAuthData := makeAuthData(rpID, 1, 0x05)
	signature := signAssertion(t, key, loginAuthData, loginClientData)
	loginVerifyBody, _ := json.Marshal(map[string]any{
		"ticket":            ticket,
		"credentialId":      credID,
		"clientDataJSON":    b64url(loginClientData),
		"authenticatorData": b64url(loginAuthData),
		"signature":         signature,
	})
	resp, raw = authReq(t, http.MethodPost, srv.URL+"/api/auth/qr/passkey/verify", "",
		bytes.NewReader(loginVerifyBody), proxyHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("qr passkey verify status=%d body=%s", resp.StatusCode, string(raw))
	}

	// 5) 轮询应返回 approved + token
	resp, raw = authReq(t, http.MethodGet, srv.URL+"/api/auth/qr/session/"+sessionID, "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("qr poll status=%d body=%s", resp.StatusCode, string(raw))
	}
	var poll map[string]any
	_ = json.Unmarshal(raw, &poll)
	if poll["status"] != string(qrPollApproved) {
		t.Fatalf("poll status=%v body=%s", poll["status"], string(raw))
	}
	if tok, _ := poll["token"].(string); tok == "" {
		t.Fatalf("approved poll missing token: %s", string(raw))
	}
}
