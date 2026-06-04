package store

import (
	"errors"
	"os"
	"strings"
	"time"

	"lumehub/internal/model"
)

var (
	ErrPasskeyCredentialIDEmpty    = errors.New("passkey credential id empty")
	ErrPasskeyPublicKeyEmpty       = errors.New("passkey public key empty")
	ErrPasskeyCredentialIDConflict = errors.New("passkey credential id conflict")
)

func (s *Store) FindAccountByPasskeyCredentialID(credentialID string) (*model.Account, *model.PasskeyCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readAccountsUnlocked()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, os.ErrNotExist
		}
		return nil, nil, err
	}
	needle := strings.TrimSpace(credentialID)
	for i := range doc.Accounts {
		for j := range doc.Accounts[i].Passkeys {
			if doc.Accounts[i].Passkeys[j].ID == needle {
				acc := doc.Accounts[i]
				cred := doc.Accounts[i].Passkeys[j]
				return &acc, &cred, nil
			}
		}
	}
	return nil, nil, os.ErrNotExist
}

func (s *Store) AddAccountPasskey(accountID string, passkey model.PasskeyCredential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readAccountsUnlocked()
	if err != nil {
		return err
	}
	credID := strings.TrimSpace(passkey.ID)
	pubKey := strings.TrimSpace(passkey.PublicKey)
	if credID == "" {
		return ErrPasskeyCredentialIDEmpty
	}
	if pubKey == "" {
		return ErrPasskeyPublicKeyEmpty
	}

	for i := range doc.Accounts {
		for j := range doc.Accounts[i].Passkeys {
			if doc.Accounts[i].Passkeys[j].ID == credID && doc.Accounts[i].ID != accountID {
				return ErrPasskeyCredentialIDConflict
			}
		}
	}

	found := -1
	for i := range doc.Accounts {
		if doc.Accounts[i].ID == accountID {
			found = i
			break
		}
	}
	if found < 0 {
		return os.ErrNotExist
	}

	if passkey.CreatedAt == "" {
		passkey.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	replaced := false
	for i := range doc.Accounts[found].Passkeys {
		if doc.Accounts[found].Passkeys[i].ID == credID {
			doc.Accounts[found].Passkeys[i] = passkey
			replaced = true
			break
		}
	}
	if !replaced {
		doc.Accounts[found].Passkeys = append(doc.Accounts[found].Passkeys, passkey)
	}

	return s.writeAccountsUnlocked(doc)
}

func (s *Store) UpdatePasskeySignCount(accountID, credentialID string, signCount uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, err := s.readAccountsUnlocked()
	if err != nil {
		return err
	}
	found := -1
	for i := range doc.Accounts {
		if doc.Accounts[i].ID == accountID {
			found = i
			break
		}
	}
	if found < 0 {
		return os.ErrNotExist
	}

	for i := range doc.Accounts[found].Passkeys {
		if doc.Accounts[found].Passkeys[i].ID == strings.TrimSpace(credentialID) {
			doc.Accounts[found].Passkeys[i].SignCount = signCount
			doc.Accounts[found].Passkeys[i].LastUsedAt = time.Now().UTC().Format(time.RFC3339)
			return s.writeAccountsUnlocked(doc)
		}
	}
	return os.ErrNotExist
}
