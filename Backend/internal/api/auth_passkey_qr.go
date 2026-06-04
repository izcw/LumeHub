package api

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"lumehub/internal/model"
	"lumehub/internal/store"
)

const (
	qrSessionTTL          = 90 * time.Second
	qrPasskeyChallengeTTL = 70 * time.Second
	defaultPollIntervalMs = 1800
	registerChallengeTTL  = 5 * time.Minute
)

type qrPollStatus string

const (
	qrPollPending  qrPollStatus = "pending"
	qrPollApproved qrPollStatus = "approved"
	qrPollExpired  qrPollStatus = "expired"
	qrPollRejected qrPollStatus = "rejected"
)

type qrSession struct {
	ID               string
	Ticket           string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	Status           qrPollStatus
	LastError        string
	Challenge        string
	ChallengeExpires time.Time
	Origin           string
	RPID             string
	ApprovedToken    string
	ApprovedExpires  time.Time
	ApprovedUser     model.AccountPublic
	Delivered        bool
}

type registerChallenge struct {
	UserID    string
	Challenge string
	Origin    string
	RPID      string
	ExpiresAt time.Time
	Label     string
}

type qrLoginState struct {
	mu              sync.Mutex
	byID            map[string]*qrSession
	byTicket        map[string]*qrSession
	registerPending map[string]registerChallenge
}

func newQRLoginState() *qrLoginState {
	return &qrLoginState{
		byID:            make(map[string]*qrSession),
		byTicket:        make(map[string]*qrSession),
		registerPending: make(map[string]registerChallenge),
	}
}

func (s *qrLoginState) gcLocked(now time.Time) {
	for id, ses := range s.byID {
		if now.After(ses.ExpiresAt.Add(2 * time.Minute)) {
			delete(s.byID, id)
			delete(s.byTicket, ses.Ticket)
		}
	}
	for uid, rec := range s.registerPending {
		if now.After(rec.ExpiresAt) {
			delete(s.registerPending, uid)
		}
	}
}

func (s *qrLoginState) createSession(origin, rpID string) (*qrSession, error) {
	id, err := randTokenHex(12)
	if err != nil {
		return nil, err
	}
	ticket, err := randTokenHex(18)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	ses := &qrSession{
		ID:        "qrs_" + id,
		Ticket:    "qrt_" + ticket,
		CreatedAt: now,
		ExpiresAt: now.Add(qrSessionTTL),
		Status:    qrPollPending,
		Origin:    origin,
		RPID:      rpID,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked(now)
	s.byID[ses.ID] = ses
	s.byTicket[ses.Ticket] = ses
	return ses, nil
}

func (s *qrLoginState) getByID(id string) *qrSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked(time.Now().UTC())
	ses, ok := s.byID[strings.TrimSpace(id)]
	if !ok {
		return nil
	}
	return ses
}

func (s *qrLoginState) pollState(sessionID string) (qrSession, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked(time.Now().UTC())
	ses, ok := s.byID[strings.TrimSpace(sessionID)]
	if !ok {
		return qrSession{}, false, false
	}
	deliveredNow := false
	if ses.Status == qrPollApproved && !ses.Delivered && ses.ApprovedToken != "" {
		ses.Delivered = true
		deliveredNow = true
	}
	cp := *ses
	return cp, true, deliveredNow
}

func (s *qrLoginState) getByTicket(ticket string) *qrSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked(time.Now().UTC())
	ses, ok := s.byTicket[strings.TrimSpace(ticket)]
	if !ok {
		return nil
	}
	return ses
}

func (s *qrLoginState) setChallenge(sessionID, challenge string, exp time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ses, ok := s.byID[sessionID]
	if !ok {
		return false
	}
	ses.Challenge = challenge
	ses.ChallengeExpires = exp
	ses.Status = qrPollPending
	ses.LastError = ""
	return true
}

func (s *qrLoginState) reject(sessionID, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ses, ok := s.byID[sessionID]; ok {
		ses.Status = qrPollRejected
		ses.LastError = strings.TrimSpace(msg)
	}
}

func (s *qrLoginState) approve(sessionID, token string, exp time.Time, user model.AccountPublic) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ses, ok := s.byID[sessionID]; ok {
		ses.Status = qrPollApproved
		ses.LastError = ""
		ses.Challenge = ""
		ses.ChallengeExpires = time.Time{}
		ses.ApprovedToken = token
		ses.ApprovedExpires = exp
		ses.ApprovedUser = user
		ses.Delivered = false
	}
}

func (s *qrLoginState) takeRegisterChallenge(uid string) (registerChallenge, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.registerPending[uid]
	if !ok {
		return registerChallenge{}, false
	}
	if time.Now().UTC().After(rec.ExpiresAt) {
		delete(s.registerPending, uid)
		return registerChallenge{}, false
	}
	delete(s.registerPending, uid)
	return rec, true
}

func (s *qrLoginState) putRegisterChallenge(rec registerChallenge) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked(time.Now().UTC())
	s.registerPending[rec.UserID] = rec
}

func randTokenHex(bytesN int) (string, error) {
	b := make([]byte, bytesN)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func inferOriginAndRPID(r *http.Request) (string, string) {
	// WebAuthn clientDataJSON.origin 来自浏览器页面地址；开发代理下 r.Host 往往是后端地址，必须优先读 Origin。
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		if u, err := url.Parse(origin); err == nil && u.Scheme != "" && u.Host != "" {
			return u.Scheme + "://" + u.Host, rpIDFromHost(u.Host)
		}
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		if u, err := url.Parse(referer); err == nil && u.Scheme != "" && u.Host != "" {
			return u.Scheme + "://" + u.Host, rpIDFromHost(u.Host)
		}
	}

	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if i := strings.Index(host, ","); i >= 0 {
		host = strings.TrimSpace(host[:i])
	}
	return proto + "://" + host, rpIDFromHost(host)
}

func rpIDFromHost(host string) string {
	if i := strings.Index(host, ","); i >= 0 {
		host = strings.TrimSpace(host[:i])
	}
	rpID := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		rpID = h
	}
	rpID = strings.ToLower(strings.TrimSpace(rpID))
	if rpID == "" {
		rpID = "localhost"
	}
	return rpID
}

type qrSessionCreateResponse struct {
	SessionID      string `json:"sessionId"`
	QRLoginURL     string `json:"qrLoginUrl"`
	ExpiresAt      string `json:"expiresAt"`
	PollIntervalMS int    `json:"pollIntervalMs"`
}

func (h *Handler) handleAuthQRSessionCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !h.auth.Configured() {
		http.Error(w, "auth not configured", http.StatusBadRequest)
		return
	}
	origin, rpID := inferOriginAndRPID(r)
	ses, err := h.qr.createSession(origin, rpID)
	if err != nil {
		http.Error(w, "failed to create qr session", http.StatusInternalServerError)
		return
	}
	out := qrSessionCreateResponse{
		SessionID:      ses.ID,
		QRLoginURL:     origin + "/auth/qr/approve?t=" + ses.Ticket,
		ExpiresAt:      ses.ExpiresAt.UTC().Format(time.RFC3339),
		PollIntervalMS: defaultPollIntervalMs,
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) handleAuthQRSessionPoll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	id := strings.TrimSpace(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/auth/qr/session/"), "/"))
	if id == "" {
		http.NotFound(w, r)
		return
	}
	ses, ok, deliveredNow := h.qr.pollState(id)
	if !ok {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": qrPollExpired})
		return
	}
	now := time.Now().UTC()
	if now.After(ses.ExpiresAt) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": qrPollExpired})
		return
	}
	switch ses.Status {
	case qrPollPending:
		stage := "waiting_scan"
		if ses.Challenge != "" && now.Before(ses.ChallengeExpires) {
			stage = "waiting_confirm"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": qrPollPending,
			"stage":  stage,
		})
	case qrPollRejected:
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": qrPollRejected,
			"error":  ses.LastError,
		})
	case qrPollApproved:
		if ses.ApprovedToken == "" || !deliveredNow {
			_ = json.NewEncoder(w).Encode(map[string]any{"status": qrPollExpired})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    qrPollApproved,
			"token":     ses.ApprovedToken,
			"expiresAt": ses.ApprovedExpires.UTC().Format(time.RFC3339),
			"user":      ses.ApprovedUser,
		})
	default:
		_ = json.NewEncoder(w).Encode(map[string]any{"status": qrPollExpired})
	}
}

func (h *Handler) handleAuthQRApprovePage(w http.ResponseWriter, r *http.Request) {
	ticket := strings.TrimSpace(r.URL.Query().Get("t"))
	if ticket == "" {
		http.Error(w, "invalid ticket", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1" />
  <title>LumeHub 通行证确认</title>
  <style>
    *,*::before,*::after{box-sizing:border-box}
    body{
      font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'PingFang SC','Microsoft YaHei',sans-serif;
      margin:0;min-height:100dvh;background:#f5f5f3;color:#111;
      display:flex;align-items:center;justify-content:center;padding:24px;
      -webkit-font-smoothing:antialiased;
    }
    .wrap{width:100%;max-width:380px}
    .card{
      background:#fff;border:1px solid #e4e4e2;border-radius:16px;
      padding:32px 28px 24px;text-align:center;
    }
    .brand{
      margin:0 0 24px;font-size:12px;font-weight:500;letter-spacing:.14em;
      text-transform:uppercase;color:#9a9a9a;
    }
    .icon-ring{
      width:56px;height:56px;margin:0 auto 20px;border:1px solid #e4e4e2;
      border-radius:50%;display:flex;align-items:center;justify-content:center;
      background:#fafafa;color:#111;overflow:hidden;
    }
    .icon-ring svg{width:26px;height:26px;display:block}
    .icon-ring img.face-icon{width:34px;height:34px;display:block;object-fit:contain}
    .icon-ring.is-loading .face-icon{display:none}
    .icon-ring.is-loading .spinner{display:block}
    .icon-ring .spinner{display:none}
    .icon-ring.ok{border-color:#bbf7d0;background:#f0fdf4;color:#067647}
    .icon-ring.err{border-color:#fecaca;background:#fef2f2;color:#b42318}
    .spinner{
      width:22px;height:22px;border:2px solid #e4e4e2;border-top-color:#111;
      border-radius:50%;animation:spin .75s linear infinite;
    }
    @keyframes spin{to{transform:rotate(360deg)}}
    h1{font-size:18px;font-weight:600;margin:0 0 10px;letter-spacing:-.02em}
    .status{font-size:15px;line-height:1.5;color:#111;margin:0 0 6px;font-weight:500}
    .status.ok{color:#067647}
    .status.err{color:#b42318}
    .detail{font-size:13px;line-height:1.65;color:#6b6b6b;margin:0}
    .hint{
      margin:22px 0 0;padding-top:18px;border-top:1px solid #efefee;
      font-size:12px;line-height:1.55;color:#9a9a9a;
    }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <p class="brand">LumeHub</p>
      <div class="icon-ring is-loading" id="iconRing">
        <img src="/resource/system/faceID.svg" alt="" class="face-icon" />
        <div class="spinner" aria-hidden="true"></div>
      </div>
      <h1>确认登录</h1>
      <p class="status" id="status">正在调用通行证…</p>
      <p class="detail" id="detail">请按系统提示使用 Face ID / Touch ID 完成验证</p>
      <p class="hint" id="hint">验证成功后即可关闭本页</p>
    </div>
  </div>
  <script>
    const ticket = new URLSearchParams(location.search).get('t') || '';
    const statusEl = document.getElementById('status');
    const detailEl = document.getElementById('detail');
    const hintEl = document.getElementById('hint');
    const iconRing = document.getElementById('iconRing');
    const iconOk = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>';
    const iconErr = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/></svg>';
    function setOk(msg, detail) {
      statusEl.className = 'status ok';
      statusEl.textContent = msg;
      detailEl.textContent = detail;
      hintEl.textContent = '此页面可以关闭';
      iconRing.className = 'icon-ring ok';
      iconRing.innerHTML = iconOk;
    }
    function setErr(msg, detail) {
      statusEl.className = 'status err';
      statusEl.textContent = msg;
      detailEl.textContent = detail;
      hintEl.textContent = '请返回电脑刷新二维码后重试';
      iconRing.className = 'icon-ring err';
      iconRing.innerHTML = iconErr;
    }
    const toBytes = (b64url) => {
      const pad = '='.repeat((4 - b64url.length % 4) % 4);
      const b64 = (b64url + pad).replace(/-/g, '+').replace(/_/g, '/');
      const raw = atob(b64);
      const arr = new Uint8Array(raw.length);
      for (let i = 0; i < raw.length; i++) arr[i] = raw.charCodeAt(i);
      return arr;
    };
    const fromBytes = (buf) => {
      const arr = new Uint8Array(buf);
      let raw = '';
      for (let i = 0; i < arr.length; i++) raw += String.fromCharCode(arr[i]);
      return btoa(raw).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/,'');
    };
    async function run() {
      if (!ticket) throw new Error('二维码已失效');
      if (!window.PublicKeyCredential || !navigator.credentials?.get) {
        throw new Error('当前浏览器不支持通行证');
      }
      iconRing.classList.add('is-loading');
      const optResp = await fetch('/api/auth/qr/passkey/options', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ticket })
      });
      if (!optResp.ok) throw new Error('登录会话已失效，请返回电脑刷新二维码');
      const optData = await optResp.json();
      const publicKey = {
        challenge: toBytes(optData.challenge),
        rpId: optData.rpId,
        timeout: optData.timeoutMs || 60000,
        userVerification: optData.userVerification || 'required'
      };
      statusEl.textContent = '等待系统通行证确认...';
      iconRing.classList.remove('is-loading');
      const credential = await navigator.credentials.get({ publicKey });
      if (!credential) throw new Error('通行证确认已取消');
      const resp = credential.response;
      const payload = {
        ticket,
        credentialId: credential.id,
        clientDataJSON: fromBytes(resp.clientDataJSON),
        authenticatorData: fromBytes(resp.authenticatorData),
        signature: fromBytes(resp.signature),
        userHandle: resp.userHandle ? fromBytes(resp.userHandle) : ''
      };
      const verifyResp = await fetch('/api/auth/qr/passkey/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (!verifyResp.ok) {
        const text = await verifyResp.text();
        throw new Error(text || '通行证验证失败');
      }
      setOk('确认成功，可返回电脑继续', '电脑端将自动完成登录');
    }
    run().catch((err) => {
      setErr('确认失败', (err && err.message) ? err.message : '请返回电脑刷新二维码后重试');
    });
  </script>
</body>
</html>`))
}

type passkeyOptionsReq struct {
	Ticket string `json:"ticket"`
}

func (h *Handler) handleAuthQRPasskeyOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<15))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req passkeyOptionsReq
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ses := h.qr.getByTicket(req.Ticket)
	if ses == nil || time.Now().UTC().After(ses.ExpiresAt) || ses.Status != qrPollPending {
		http.Error(w, "session expired", http.StatusBadRequest)
		return
	}
	challengeRaw, err := randTokenHex(20)
	if err != nil {
		http.Error(w, "failed to issue challenge", http.StatusInternalServerError)
		return
	}
	challenge := base64.RawURLEncoding.EncodeToString([]byte(challengeRaw))
	exp := time.Now().UTC().Add(qrPasskeyChallengeTTL)
	if ok := h.qr.setChallenge(ses.ID, challenge, exp); !ok {
		http.Error(w, "session unavailable", http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"challenge":        challenge,
		"rpId":             ses.RPID,
		"timeoutMs":        int(qrPasskeyChallengeTTL / time.Millisecond),
		"userVerification": "required",
	})
}

type passkeyVerifyReq struct {
	Ticket            string `json:"ticket"`
	CredentialID      string `json:"credentialId"`
	ClientDataJSON    string `json:"clientDataJSON"`
	AuthenticatorData string `json:"authenticatorData"`
	Signature         string `json:"signature"`
	UserHandle        string `json:"userHandle"`
}

type webauthnClientData struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	Origin    string `json:"origin"`
}

func (h *Handler) handleAuthQRPasskeyVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req passkeyVerifyReq
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ses := h.qr.getByTicket(req.Ticket)
	if ses == nil || ses.Status != qrPollPending || time.Now().UTC().After(ses.ExpiresAt) {
		http.Error(w, "session expired", http.StatusBadRequest)
		return
	}
	if ses.Challenge == "" || time.Now().UTC().After(ses.ChallengeExpires) {
		http.Error(w, "challenge expired", http.StatusBadRequest)
		return
	}

	clientDataRaw, err := decodeAnyBase64(req.ClientDataJSON)
	if err != nil {
		h.qr.reject(ses.ID, "客户端数据解码失败")
		http.Error(w, "invalid clientDataJSON", http.StatusBadRequest)
		return
	}
	var clientData webauthnClientData
	if err := json.Unmarshal(clientDataRaw, &clientData); err != nil {
		h.qr.reject(ses.ID, "客户端数据格式错误")
		http.Error(w, "invalid clientDataJSON", http.StatusBadRequest)
		return
	}
	if clientData.Type != "webauthn.get" {
		h.qr.reject(ses.ID, "通行证类型不匹配")
		http.Error(w, "invalid credential type", http.StatusBadRequest)
		return
	}
	if clientData.Challenge != ses.Challenge {
		h.qr.reject(ses.ID, "challenge 不匹配")
		http.Error(w, "challenge mismatch", http.StatusBadRequest)
		return
	}
	if normalizeOrigin(clientData.Origin) != normalizeOrigin(ses.Origin) {
		h.qr.reject(ses.ID, "origin 不匹配")
		http.Error(w, "origin mismatch", http.StatusBadRequest)
		return
	}

	authData, err := decodeAnyBase64(req.AuthenticatorData)
	if err != nil {
		h.qr.reject(ses.ID, "认证器数据解码失败")
		http.Error(w, "invalid authenticatorData", http.StatusBadRequest)
		return
	}
	signature, err := decodeAnyBase64(req.Signature)
	if err != nil {
		h.qr.reject(ses.ID, "签名解码失败")
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	newSignCount, err := verifyAuthenticatorData(authData, ses.RPID, true)
	if err != nil {
		h.qr.reject(ses.ID, err.Error())
		http.Error(w, "invalid authenticator data", http.StatusBadRequest)
		return
	}

	acc, cred, err := h.store.FindAccountByPasskeyCredentialID(strings.TrimSpace(req.CredentialID))
	if err != nil {
		h.qr.reject(ses.ID, "通行证未绑定账号")
		http.Error(w, "passkey not found", http.StatusUnauthorized)
		return
	}

	if cred.SignCount > 0 && newSignCount <= cred.SignCount {
		h.qr.reject(ses.ID, "检测到计数器异常，请重新绑定通行证")
		http.Error(w, "sign counter invalid", http.StatusUnauthorized)
		return
	}

	if err := verifyPasskeySignature(cred.PublicKey, authData, clientDataRaw, signature); err != nil {
		h.qr.reject(ses.ID, "签名校验失败")
		http.Error(w, "signature verify failed", http.StatusUnauthorized)
		return
	}

	token, exp := h.auth.CreateSession(acc.ID)
	if err := h.store.UpdatePasskeySignCount(acc.ID, cred.ID, newSignCount); err != nil && !errors.Is(err, os.ErrNotExist) {
		h.log.Printf("passkey signCount update failed: %v", err)
	}

	pub := model.ToAccountPublic(*acc)
	h.fillDefaultAvatar(&pub, acc)
	h.qr.approve(ses.ID, token, exp, pub)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": qrPollApproved})
}

type passkeyRegisterOptionsReq struct {
	Label string `json:"label"`
}

func (h *Handler) handlePasskeyRegisterOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !h.auth.Configured() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	uid, ok := h.auth.SessionUserID(r)
	if !ok || uid == "legacy" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	acc, err := h.store.GetAccountByID(uid)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<14))
	var req passkeyRegisterOptionsReq
	_ = json.Unmarshal(body, &req)

	origin, rpID := inferOriginAndRPID(r)
	raw, err := randTokenHex(20)
	if err != nil {
		http.Error(w, "challenge issue failed", http.StatusInternalServerError)
		return
	}
	challenge := base64.RawURLEncoding.EncodeToString([]byte(raw))
	h.qr.putRegisterChallenge(registerChallenge{
		UserID:    uid,
		Challenge: challenge,
		Origin:    origin,
		RPID:      rpID,
		ExpiresAt: time.Now().UTC().Add(registerChallengeTTL),
		Label:     strings.TrimSpace(req.Label),
	})

	userHandle := base64.RawURLEncoding.EncodeToString([]byte(uid))
	name := strings.TrimSpace(acc.Email)
	if name == "" {
		name = strings.TrimSpace(acc.Username)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"challenge": challenge,
		"rp": map[string]any{
			"id":   rpID,
			"name": "LumeHub",
		},
		"user": map[string]any{
			"id":          userHandle,
			"name":        name,
			"displayName": acc.DisplayName,
		},
		"pubKeyCredParams": []map[string]any{
			{"type": "public-key", "alg": -7},
			{"type": "public-key", "alg": -257},
		},
		"timeoutMs":   int(registerChallengeTTL / time.Millisecond),
		"attestation": "none",
		"authenticatorSelection": map[string]any{
			"userVerification": "required",
			"residentKey":      "required",
		},
	})
}

type passkeyRegisterVerifyReq struct {
	CredentialID      string   `json:"credentialId"`
	PublicKey         string   `json:"publicKey"`
	Algorithm         int      `json:"algorithm"`
	ClientDataJSON    string   `json:"clientDataJSON"`
	AuthenticatorData string   `json:"authenticatorData"`
	Transports        []string `json:"transports"`
	Label             string   `json:"label"`
}

func (h *Handler) handlePasskeyList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !h.auth.Configured() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	uid, ok := h.auth.SessionUserID(r)
	if !ok || uid == "legacy" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	acc, err := h.store.GetAccountByID(uid)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	out := make([]map[string]any, 0, len(acc.Passkeys))
	for _, p := range acc.Passkeys {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		mask := id
		if len(id) > 12 {
			mask = id[:6] + "..." + id[len(id)-4:]
		}
		out = append(out, map[string]any{
			"id":         id,
			"label":      strings.TrimSpace(p.Label),
			"displayId":  mask,
			"algorithm":  p.Algorithm,
			"signCount":  p.SignCount,
			"transports": append([]string(nil), p.Transports...),
			"createdAt":  p.CreatedAt,
			"lastUsedAt": p.LastUsedAt,
		})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"items": out})
}

func (h *Handler) handlePasskeyRegisterVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !h.auth.Configured() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	uid, ok := h.auth.SessionUserID(r)
	if !ok || uid == "legacy" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req passkeyRegisterVerifyReq
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	rec, ok := h.qr.takeRegisterChallenge(uid)
	if !ok || time.Now().UTC().After(rec.ExpiresAt) {
		http.Error(w, "challenge expired", http.StatusBadRequest)
		return
	}

	clientDataRaw, err := decodeAnyBase64(req.ClientDataJSON)
	if err != nil {
		http.Error(w, "invalid clientDataJSON", http.StatusBadRequest)
		return
	}
	var clientData webauthnClientData
	if err := json.Unmarshal(clientDataRaw, &clientData); err != nil {
		http.Error(w, "invalid clientDataJSON", http.StatusBadRequest)
		return
	}
	if clientData.Type != "webauthn.create" {
		http.Error(w, "invalid credential type", http.StatusBadRequest)
		return
	}
	if clientData.Challenge != rec.Challenge {
		http.Error(w, "challenge mismatch", http.StatusBadRequest)
		return
	}
	if normalizeOrigin(clientData.Origin) != normalizeOrigin(rec.Origin) {
		http.Error(w, "origin mismatch", http.StatusBadRequest)
		return
	}

	authData, err := decodeAnyBase64(req.AuthenticatorData)
	if err != nil {
		http.Error(w, "invalid authenticatorData", http.StatusBadRequest)
		return
	}
	signCount, err := verifyAuthenticatorData(authData, rec.RPID, true)
	if err != nil {
		http.Error(w, "invalid authenticator data", http.StatusBadRequest)
		return
	}

	passkey := model.PasskeyCredential{
		ID:         strings.TrimSpace(req.CredentialID),
		Label:      strings.TrimSpace(req.Label),
		PublicKey:  strings.TrimSpace(req.PublicKey),
		Algorithm:  req.Algorithm,
		SignCount:  signCount,
		Transports: append([]string(nil), req.Transports...),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if passkey.ID == "" || passkey.PublicKey == "" {
		http.Error(w, "passkey data missing", http.StatusBadRequest)
		return
	}
	if err := h.store.AddAccountPasskey(uid, passkey); err != nil {
		switch {
		case errors.Is(err, store.ErrPasskeyCredentialIDConflict):
			http.Error(w, "该通行证已绑定其他账号", http.StatusConflict)
		case errors.Is(err, os.ErrNotExist):
			http.Error(w, "账号不存在", http.StatusBadRequest)
		default:
			http.Error(w, "保存通行证失败", http.StatusInternalServerError)
		}
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":           true,
		"credentialId": passkey.ID,
	})
}

func verifyAuthenticatorData(authData []byte, rpID string, requireUV bool) (uint32, error) {
	if len(authData) < 37 {
		return 0, errors.New("authenticatorData too short")
	}
	rpHashExpected := sha256.Sum256([]byte(rpID))
	if !equalFixed(authData[:32], rpHashExpected[:]) {
		return 0, errors.New("rpIdHash mismatch")
	}
	flags := authData[32]
	if (flags & 0x01) == 0 {
		return 0, errors.New("user presence required")
	}
	if requireUV && (flags&0x04) == 0 {
		return 0, errors.New("user verification required")
	}
	signCount := binary.BigEndian.Uint32(authData[33:37])
	return signCount, nil
}

func verifyPasskeySignature(pubKeyEncoded string, authenticatorData, clientDataJSON, signature []byte) error {
	pubRaw, err := decodeAnyBase64(pubKeyEncoded)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}
	pub, err := x509.ParsePKIXPublicKey(pubRaw)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}
	cdHash := sha256.Sum256(clientDataJSON)
	msg := append(append([]byte(nil), authenticatorData...), cdHash[:]...)
	msgHash := sha256.Sum256(msg)

	switch k := pub.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(k, msgHash[:], signature) {
			return errors.New("ecdsa signature mismatch")
		}
		return nil
	case *rsa.PublicKey:
		return rsa.VerifyPKCS1v15(k, crypto.SHA256, msgHash[:], signature)
	case ed25519.PublicKey:
		if !ed25519.Verify(k, msg, signature) {
			return errors.New("ed25519 signature mismatch")
		}
		return nil
	default:
		return fmt.Errorf("unsupported public key type: %T", pub)
	}
}

func decodeAnyBase64(s string) ([]byte, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return nil, errors.New("empty base64")
	}
	if b, err := base64.RawURLEncoding.DecodeString(raw); err == nil {
		return b, nil
	}
	if b, err := base64.StdEncoding.DecodeString(raw); err == nil {
		return b, nil
	}
	pad := raw + strings.Repeat("=", (4-len(raw)%4)%4)
	if b, err := base64.URLEncoding.DecodeString(pad); err == nil {
		return b, nil
	}
	return nil, errors.New("base64 decode failed")
}

func equalFixed(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func normalizeOrigin(origin string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(origin)), "/")
}
