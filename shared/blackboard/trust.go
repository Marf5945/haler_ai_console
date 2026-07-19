package blackboard

// trust.go — HMAC 簽章與信任根（spec §6.2）。
// 信任根 trusted_keys.json 由主持人保管在本機；能改共享文件的人不能改
// 信任根；金鑰交換走帶外。無簽章或驗證失敗 → untrusted（投影器降級）。

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// SigAlgHMACSHA256 is the only supported signature algorithm (v1).
const SigAlgHMACSHA256 = "hmac-sha256"

// TrustedKeyConfig is one entry of trusted_keys.json.
type TrustedKeyConfig struct {
	KeyID     string `json:"key_id"`
	Alg       string `json:"alg"`
	SecretHex string `json:"secret_hex"`
}

// TrustConfig is the on-disk shape of trusted_keys.json:
//
//	{
//	  "allow_unsigned": ["human:local-user"],
//	  "keys": {
//	    "agent:backend": {"key_id":"backend_2026_07","alg":"hmac-sha256","secret_hex":"..."}
//	  }
//	}
type TrustConfig struct {
	AllowUnsigned []string                    `json:"allow_unsigned"`
	Keys          map[string]TrustedKeyConfig `json:"keys"`
}

type trustKey struct {
	keyID  string
	secret []byte
}

// TrustStore is the loaded, host-managed trust root.
type TrustStore struct {
	allowUnsigned map[string]bool
	keys          map[string]trustKey // actor "type:id" -> key
}

// NewTrustStore builds a TrustStore from config.
func NewTrustStore(cfg TrustConfig) (*TrustStore, error) {
	ts := &TrustStore{
		allowUnsigned: map[string]bool{},
		keys:          map[string]trustKey{},
	}
	for _, a := range cfg.AllowUnsigned {
		ts.allowUnsigned[a] = true
	}
	for actor, kc := range cfg.Keys {
		if kc.Alg != "" && kc.Alg != SigAlgHMACSHA256 {
			return nil, fmt.Errorf("actor %s: unsupported alg %q", actor, kc.Alg)
		}
		secret, err := hex.DecodeString(kc.SecretHex)
		if err != nil || len(secret) == 0 {
			return nil, fmt.Errorf("actor %s: invalid secret_hex", actor)
		}
		ts.keys[actor] = trustKey{keyID: kc.KeyID, secret: secret}
	}
	return ts, nil
}

// LoadTrustStore reads trusted_keys.json from a host-private path.
func LoadTrustStore(path string) (*TrustStore, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg TrustConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("trusted_keys: %w", err)
	}
	return NewTrustStore(cfg)
}

// sigPayload produces the canonical signing payload: the event JSON as a
// key-sorted object with the "sig" field removed. Go marshals maps with
// sorted keys, so signer and verifier agree regardless of field order.
func sigPayload(raw []byte) ([]byte, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	delete(m, "sig")
	return json.Marshal(m)
}

// SignEvent attaches an HMAC-SHA256 signature to the event.
func SignEvent(ev *Event, keyID string, secret []byte) error {
	if len(secret) == 0 {
		return fmt.Errorf("empty signing secret")
	}
	ev.Sig = nil
	raw, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	payload, err := sigPayload(raw)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	ev.Sig = &Sig{
		Alg:   SigAlgHMACSHA256,
		KeyID: keyID,
		Value: hex.EncodeToString(mac.Sum(nil)),
	}
	return nil
}

// VerifyEventSig checks the event's HMAC against the given secret. It uses
// the exact Raw payload as parsed from the document when available, so
// foreign field ordering cannot break verification.
func VerifyEventSig(ev Event, secret []byte) bool {
	if ev.Sig == nil || ev.Sig.Alg != SigAlgHMACSHA256 {
		return false
	}
	want, err := hex.DecodeString(ev.Sig.Value)
	if err != nil {
		return false
	}
	raw := ev.Raw
	if len(raw) == 0 {
		ev.Sig = nil
		b, err := json.Marshal(&ev)
		if err != nil {
			return false
		}
		raw = b
	}
	payload, err := sigPayload(raw)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return hmac.Equal(mac.Sum(nil), want)
}

// TrustFunc adjudicates claimed identities for the projector:
// allow_unsigned actors pass as-is; actors with a registered key must carry
// a valid signature under the matching key_id; everyone else is untrusted.
func (ts *TrustStore) TrustFunc() TrustFunc {
	return func(ev Event) bool {
		actor := ev.Actor.String()
		if ts.allowUnsigned[actor] {
			return true
		}
		k, ok := ts.keys[actor]
		if !ok {
			return false
		}
		if ev.Sig == nil || ev.Sig.KeyID != k.keyID {
			return false
		}
		return VerifyEventSig(ev, k.secret)
	}
}
