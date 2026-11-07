package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/api/idtoken"
)

func TestRandomID(t *testing.T) {
	id, err := randomID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 43 {
		t.Fatalf("encoded ID length = %d, want 43", len(id))
	}

	original := readRandom
	readRandom = func([]byte) (int, error) { return 0, errors.New("random source failed") }
	t.Cleanup(func() { readRandom = original })
	if id, err := randomID(); err == nil || id != "" {
		t.Fatalf("randomID() = (%q, %v), want empty ID and error", id, err)
	}
}

func TestNewGoogle(t *testing.T) {
	if _, err := NewGoogle(GoogleConfig{}); err == nil {
		t.Fatal("expected missing client ID error")
	}
	if _, err := NewGoogle(GoogleConfig{ClientID: "client"}); err == nil {
		t.Fatal("expected missing allowed domain error")
	}
	g, err := NewGoogle(GoogleConfig{ClientID: "client", AllowedDomains: []string{"", " Example.COM "}})
	if err != nil {
		t.Fatal(err)
	}
	if g.audience != "client" || len(g.domains) != 1 {
		t.Fatalf("unexpected Google config: %#v", g)
	}
}

func TestGoogleVerify(t *testing.T) {
	original := validateGoogleToken
	t.Cleanup(func() { validateGoogleToken = original })
	g, err := NewGoogle(GoogleConfig{ClientID: "client", AllowedDomains: []string{"example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Verify(context.Background(), ""); err == nil {
		t.Fatal("expected empty credential error")
	}
	validateGoogleToken = func(context.Context, string, string) (*idtoken.Payload, error) {
		return nil, errors.New("bad token")
	}
	if _, err := g.Verify(context.Background(), "credential"); err == nil || err.Error() != "invalid Google credential" {
		t.Fatalf("unexpected validation error: %v", err)
	}

	now := time.Now().Unix()
	payload := &idtoken.Payload{Issuer: "wrong", Expires: now + 60, Subject: "subject", Claims: map[string]interface{}{}}
	validateGoogleToken = func(context.Context, string, string) (*idtoken.Payload, error) { return payload, nil }
	if _, err := g.Verify(context.Background(), "credential"); err == nil || err.Error() != "invalid Google issuer" {
		t.Fatalf("unexpected issuer error: %v", err)
	}

	payload.Issuer = "accounts.google.com"
	payload.Claims = map[string]interface{}{"email": "User@Example.com", "email_verified": true, "hd": "EXAMPLE.COM", "name": "User"}
	user, err := g.Verify(context.Background(), "credential")
	if err != nil || user != (GoogleUser{Subject: "subject", Email: "user@example.com", Name: "User"}) {
		t.Fatalf("Verify() = (%+v, %v)", user, err)
	}
	payload.Issuer = "https://accounts.google.com"
	if _, err := g.Verify(context.Background(), "credential"); err != nil {
		t.Fatalf("https issuer rejected: %v", err)
	}

	for _, tc := range []struct {
		name    string
		claims  map[string]interface{}
		expires int64
		want    string
	}{
		{"missing email", map[string]interface{}{"email_verified": true, "hd": "example.com"}, now + 60, "Google account email is not verified or credential is expired"},
		{"unverified email", map[string]interface{}{"email": "a@example.com", "email_verified": false, "hd": "example.com"}, now + 60, "Google account email is not verified or credential is expired"},
		{"expired", map[string]interface{}{"email": "a@example.com", "email_verified": true, "hd": "example.com"}, now, "Google account email is not verified or credential is expired"},
		{"wrong domain", map[string]interface{}{"email": "a@example.com", "email_verified": true, "hd": "other.com"}, now + 60, "Google hosted domain is not allowed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload.Claims, payload.Expires = tc.claims, tc.expires
			if _, err := g.Verify(context.Background(), "credential"); err == nil || err.Error() != tc.want {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func privateKeyPEM(t *testing.T, key *rsa.PrivateKey, pkcs8 bool) []byte {
	t.Helper()
	if pkcs8 {
		b, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			t.Fatal(err)
		}
		return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b})
	}
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
}

func TestNewALB(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	good := ALBConfig{Issuer: "issuer", KeyID: "key", PrivateKeyPEM: privateKeyPEM(t, key, true)}
	for _, cfg := range []ALBConfig{{}, {Issuer: "issuer"}, {KeyID: "key"}} {
		if _, err := NewALB(cfg); err == nil {
			t.Errorf("NewALB(%+v) expected required fields error", cfg)
		}
	}
	if _, err := NewALB(ALBConfig{Issuer: "i", KeyID: "k"}); err == nil {
		t.Fatal("expected PEM error")
	}
	if _, err := NewALB(ALBConfig{Issuer: "i", KeyID: "k", PrivateKeyPEM: []byte("-----BEGIN PRIVATE KEY-----\nYQ==\n-----END PRIVATE KEY-----")}); err == nil {
		t.Fatal("expected invalid key error")
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewALB(ALBConfig{Issuer: "i", KeyID: "k", PrivateKeyPEM: privateKeyPEM(t, otherKey, true)}); err == nil {
		t.Fatal("expected undersized key error")
	}
	if _, err := NewALB(ALBConfig{Issuer: "i", KeyID: "k", PrivateKeyPEM: privateKeyPEM(t, key, false), TTL: time.Second}); err == nil {
		t.Fatal("expected low TTL error")
	}
	if _, err := NewALB(ALBConfig{Issuer: "i", KeyID: "k", PrivateKeyPEM: privateKeyPEM(t, key, true), TTL: 25 * time.Hour}); err == nil {
		t.Fatal("expected high TTL error")
	}
	if alb, err := NewALB(good); err != nil || alb.ttl != 24*time.Hour {
		t.Fatalf("default TTL: alb=%v err=%v", alb, err)
	}
	if alb, err := NewALB(ALBConfig{Issuer: "i", KeyID: "k", PrivateKeyPEM: privateKeyPEM(t, key, false), TTL: time.Minute}); err != nil || alb.ttl != time.Minute {
		t.Fatalf("PKCS1 key: alb=%v err=%v", alb, err)
	}
	if _, err := NewALB(ALBConfig{Issuer: "i", KeyID: "k", PrivateKeyPEM: privateKeyPEM(t, key, true), TTL: 24 * time.Hour}); err != nil {
		t.Fatal(err)
	}
}

func TestALBGenerateValidateJWKS(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewALB(ALBConfig{Issuer: "issuer", KeyID: "key", PrivateKeyPEM: privateKeyPEM(t, key, true), TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.Generate("", nil); err == nil {
		t.Fatal("expected empty subject error")
	}
	encoded, generated, err := a.Generate("user", []string{"read", "write"})
	if err != nil {
		t.Fatal(err)
	}
	if generated.Scope != "read write" || generated.Subject != "user" || generated.ID == "" {
		t.Fatalf("unexpected claims: %+v", generated)
	}
	originalSigner := signJWT
	t.Cleanup(func() { signJWT = originalSigner })
	signJWT = func(*jwt.Token, *rsa.PrivateKey) (string, error) { return "", errors.New("sign failed") }
	if _, _, err := a.Generate("user", nil); err == nil {
		t.Fatal("expected signing error")
	}
	signJWT = originalSigner
	originalReader := readRandom
	readRandom = func([]byte) (int, error) { return 0, errors.New("random source failed") }
	if _, _, err := a.Generate("user", nil); err == nil {
		t.Fatal("expected random ID error")
	}
	readRandom = originalReader
	validated, err := a.Validate(encoded)
	if err != nil || validated.Subject != "user" || validated.Scope != "read write" {
		t.Fatalf("Validate() = (%+v, %v)", validated, err)
	}

	for name, bad := range map[string]string{
		"malformed": "not a jwt",
		"signature": encoded[:len(encoded)-2] + "xx",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := a.Validate(bad); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	wrongKeyID := jwt.NewWithClaims(jwt.SigningMethodRS256, validated)
	wrongKeyID.Header["kid"] = "other"
	wrongKidToken, err := wrongKeyID.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Validate(wrongKidToken); err == nil {
		t.Fatal("expected key ID error")
	}

	wrongMethod := jwt.NewWithClaims(jwt.SigningMethodRS512, validated)
	wrongMethod.Header["kid"] = "key"
	wrongMethodToken, err := wrongMethod.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Validate(wrongMethodToken); err == nil {
		t.Fatal("expected signing method error")
	}

	other, err := NewALB(ALBConfig{Issuer: "different", KeyID: "key", PrivateKeyPEM: privateKeyPEM(t, key, true)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Validate(encoded); err == nil {
		t.Fatal("expected issuer error")
	}

	noSubject := ALBClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: "issuer", ID: "id", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now())}}
	noSubjectToken := jwt.NewWithClaims(jwt.SigningMethodRS256, noSubject)
	noSubjectToken.Header["kid"] = "key"
	noSubjectEncoded, err := noSubjectToken.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Validate(noSubjectEncoded); err == nil {
		t.Fatal("expected empty subject error")
	}

	noID := ALBClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: "issuer", Subject: "user", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now())}}
	noIDToken := jwt.NewWithClaims(jwt.SigningMethodRS256, noID)
	noIDToken.Header["kid"] = "key"
	noIDEncoded, err := noIDToken.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Validate(noIDEncoded); err == nil {
		t.Fatal("expected empty token ID error")
	}

	var jwks struct {
		Keys []map[string]string `json:"keys"`
	}
	b, err := a.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &jwks); err != nil {
		t.Fatal(err)
	}
	if len(jwks.Keys) != 1 || jwks.Keys[0]["kid"] != "key" || jwks.Keys[0]["kty"] != "RSA" || jwks.Keys[0]["alg"] != "RS256" || jwks.Keys[0]["n"] == "" || jwks.Keys[0]["e"] == "" {
		t.Fatalf("unexpected JWKS: %s", b)
	}
	if strings.Contains(string(b), "PRIVATE") {
		t.Fatalf("JWKS contains private key data: %s", b)
	}
}

func TestALBGenerateWithKeyID(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewALB(ALBConfig{Issuer: "issuer", KeyID: "signing-key", PrivateKeyPEM: privateKeyPEM(t, key, true)})
	if err != nil {
		t.Fatal(err)
	}
	decodePart := func(part string) map[string]any {
		t.Helper()
		b, err := base64.RawURLEncoding.DecodeString(part)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(b, &decoded); err != nil {
			t.Fatal(err)
		}
		return decoded
	}

	encoded, generated, err := a.GenerateWithKeyID("user", []string{"read", "write"}, "api-key")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(encoded, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}
	header, payload := decodePart(parts[0]), decodePart(parts[1])
	if payload["key_id"] != "api-key" || payload["iss"] != "issuer" || payload["sub"] != "user" || payload["scope"] != "read write" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	expiresAt, hasExpiresAt := payload["exp"].(float64)
	issuedAt, hasIssuedAt := payload["iat"].(float64)
	if !hasExpiresAt || !hasIssuedAt || expiresAt-issuedAt != 86400 {
		t.Fatalf("unexpected token lifetime: exp=%v iat=%v, want 86400 seconds", payload["exp"], payload["iat"])
	}
	for _, name := range []string{"aud", "tenant_id", "customer_id"} {
		if _, ok := payload[name]; ok {
			t.Errorf("unexpected %s claim: %v", name, payload[name])
		}
	}
	if header["kid"] != "signing-key" {
		t.Fatalf("header kid = %v, want signing-key", header["kid"])
	}
	if generated.APIKeyID != "api-key" {
		t.Fatalf("generated APIKeyID = %q, want api-key", generated.APIKeyID)
	}
	validated, err := a.Validate(encoded)
	if err != nil || validated.APIKeyID != generated.APIKeyID {
		t.Fatalf("Validate() = (%+v, %v), want APIKeyID %q", validated, err, generated.APIKeyID)
	}

	plain, _, err := a.Generate("user", []string{"read", "write"})
	if err != nil {
		t.Fatal(err)
	}
	parts = strings.Split(plain, ".")
	if len(parts) != 3 {
		t.Fatalf("plain token has %d parts, want 3", len(parts))
	}
	payload = decodePart(parts[1])
	if _, ok := payload["key_id"]; ok {
		t.Fatalf("plain Generate payload contains key_id: %+v", payload)
	}
	validated, err = a.Validate(plain)
	if err != nil || validated.APIKeyID != "" {
		t.Fatalf("plain Validate() = (%+v, %v), want empty APIKeyID", validated, err)
	}

	for _, tc := range []struct {
		name    string
		subject string
		keyID   string
		want    string
	}{
		{"empty subject", "", "api-key", "subject is required"},
		{"empty key ID", "user", "", "API key ID is required"},
		{"both empty", "", "", "subject is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if encoded, _, err := a.GenerateWithKeyID(tc.subject, nil, tc.keyID); err == nil || err.Error() != tc.want || encoded != "" {
				t.Fatalf("GenerateWithKeyID() = (%q, %v), want empty token and %q", encoded, err, tc.want)
			}
		})
	}
	t.Run("random ID failure", func(t *testing.T) {
		original := readRandom
		t.Cleanup(func() { readRandom = original })
		want := errors.New("random source failed")
		readRandom = func([]byte) (int, error) { return 0, want }
		if encoded, _, err := a.GenerateWithKeyID("user", nil, "api-key"); !errors.Is(err, want) || encoded != "" {
			t.Fatalf("GenerateWithKeyID() = (%q, %v), want empty token and %v", encoded, err, want)
		}
	})
	t.Run("signing failure", func(t *testing.T) {
		original := signJWT
		t.Cleanup(func() { signJWT = original })
		want := errors.New("sign failed")
		signJWT = func(*jwt.Token, *rsa.PrivateKey) (string, error) { return "", want }
		if encoded, _, err := a.GenerateWithKeyID("user", nil, "api-key"); !errors.Is(err, want) || encoded != "" {
			t.Fatalf("GenerateWithKeyID() = (%q, %v), want empty token and %v", encoded, err, want)
		}
	})
}

func albSigner(t *testing.T, keyID string) *ALB {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewALB(ALBConfig{Issuer: "issuer", KeyID: keyID, PrivateKeyPEM: privateKeyPEM(t, key, true)})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func albClaimsJSON(t *testing.T, claims ALBClaims) string {
	t.Helper()
	b, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestNewALBVerifier(t *testing.T) {
	a := albSigner(t, "key")
	public, err := a.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(public, &document); err != nil {
		t.Fatal(err)
	}
	jwks := func(changes ...map[string]any) []byte {
		t.Helper()
		keys := make([]map[string]any, len(changes))
		for i, fields := range changes {
			keys[i] = make(map[string]any)
			for name, value := range document.Keys[0] {
				keys[i][name] = value
			}
			for name, value := range fields {
				keys[i][name] = value
				if value == nil {
					delete(keys[i], name)
				}
			}
		}
		b, err := json.Marshal(map[string]any{"keys": keys})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	evenModulus := make([]byte, 256)
	evenModulus[0] = 0x80
	evenModulus[len(evenModulus)-1] = 2
	evenModulusFields := map[string]any{"kid": "even", "n": base64.RawURLEncoding.EncodeToString(evenModulus), "e": "Aw"}
	for _, tc := range []struct {
		name   string
		issuer string
		jwks   []byte
		want   string
	}{
		{"missing issuer", "", []byte("{"), "issuer is required"},
		{"nil JWKS", "issuer", nil, "invalid JWKS"},
		{"invalid JSON", "issuer", []byte("{"), "invalid JWKS"},
		{"invalid field type", "issuer", jwks(map[string]any{"kid": 42}), "invalid JWKS"},
		{"no keys", "issuer", jwks(), "JWKS must contain at least one key"},
		{"null document", "issuer", []byte("null"), "JWKS must contain at least one key"},
		{"null keys", "issuer", []byte(`{"keys":null}`), "JWKS must contain at least one key"},
		{"null key entry", "issuer", []byte(`{"keys":[null]}`), "JWKS key ID is required"},
		{"missing key ID", "issuer", jwks(map[string]any{"kid": nil, "kty": "EC"}), "JWKS key ID is required"},
		{"duplicate key ID", "issuer", jwks(nil, map[string]any{"kty": "EC"}), "duplicate JWKS key ID"},
		{"non RSA key", "issuer", jwks(nil, map[string]any{"kid": "bad", "kty": "EC", "alg": "ES256", "n": "!"}), "JWKS key must be RSA with at least 2048 bits"},
		{"undersized key", "issuer", jwks(map[string]any{"n": base64.RawURLEncoding.EncodeToString(a.key.N.Bytes()[1:]), "alg": "RS512"}), "JWKS key must be RSA with at least 2048 bits"},
		{"wrong algorithm", "issuer", jwks(map[string]any{"alg": "RS512", "e": "!"}), "JWKS key must be an RS256 signing key"},
		{"wrong use", "issuer", jwks(map[string]any{"use": "enc", "n": "!"}), "JWKS key must be an RS256 signing key"},
		{"bad modulus", "issuer", jwks(map[string]any{"n": "!"}), "invalid JWKS key encoding"},
		{"missing modulus", "issuer", jwks(map[string]any{"n": nil}), "invalid JWKS key encoding"},
		{"even modulus", "issuer", jwks(evenModulusFields), "invalid JWKS key encoding"},
		{"even modulus after valid key", "issuer", jwks(nil, evenModulusFields), "invalid JWKS key encoding"},
		{"bad exponent", "issuer", jwks(map[string]any{"e": "!"}), "invalid JWKS key encoding"},
		{"missing exponent", "issuer", jwks(map[string]any{"e": nil}), "invalid JWKS key encoding"},
		{"zero exponent", "issuer", jwks(map[string]any{"e": "AA"}), "invalid JWKS key encoding"},
		{"one exponent", "issuer", jwks(map[string]any{"e": "AQ"}), "invalid JWKS key encoding"},
		{"even exponent", "issuer", jwks(map[string]any{"e": "Ag"}), "invalid JWKS key encoding"},
		{"even exponent after valid key", "issuer", jwks(nil, map[string]any{"kid": "even", "e": "Ag"}), "invalid JWKS key encoding"},
		{"exponent above maximum", "issuer", jwks(map[string]any{"e": "gAAAAA"}), "invalid JWKS key encoding"},
		{"exponent overflow", "issuer", jwks(map[string]any{"e": "AQAAAAAAAAAB"}), "invalid JWKS key encoding"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if v, err := NewALBVerifier(ALBVerifierConfig{Issuer: tc.issuer, JWKS: tc.jwks}); err == nil || err.Error() != tc.want || v != nil {
				t.Fatalf("NewALBVerifier() = (%v, %v), want nil and %q", v, err, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		name     string
		jwks     []byte
		count    int
		exponent int
	}{
		{"round trip", public, 1, a.key.E},
		{"optional fields omitted", jwks(map[string]any{"alg": nil, "use": nil}), 1, a.key.E},
		{"several keys", jwks(nil, map[string]any{"kid": "second"}), 2, a.key.E},
		{"private fields ignored", jwks(map[string]any{"d": map[string]any{"ignored": true}, "p": 42, "q": false, "dp": []int{1}, "dq": "!", "qi": nil, "oth": "!"}), 1, a.key.E},
		{"minimum exponent", jwks(map[string]any{"e": "Aw"}), 1, 3},
		{"maximum exponent", jwks(map[string]any{"e": "f____w"}), 1, 1<<31 - 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := NewALBVerifier(ALBVerifierConfig{Issuer: "issuer", JWKS: tc.jwks})
			if err != nil {
				t.Fatal(err)
			}
			key := v.keys["key"]
			if v.issuer != "issuer" || len(v.keys) != tc.count || key == nil || key.N.Cmp(a.key.N) != 0 || key.E != tc.exponent {
				t.Fatalf("unexpected verifier: %+v, key: %+v", v, key)
			}
		})
	}
}

func TestALBVerifierValidate(t *testing.T) {
	a, other := albSigner(t, "key"), albSigner(t, "second")
	var keys []json.RawMessage
	for _, signer := range []*ALB{a, other} {
		b, err := signer.JWKS()
		if err != nil {
			t.Fatal(err)
		}
		var document struct {
			Keys []json.RawMessage `json:"keys"`
		}
		if err := json.Unmarshal(b, &document); err != nil {
			t.Fatal(err)
		}
		keys = append(keys, document.Keys...)
	}
	b, err := json.Marshal(map[string]any{"keys": keys})
	if err != nil {
		t.Fatal(err)
	}
	v, err := NewALBVerifier(ALBVerifierConfig{Issuer: "issuer", JWKS: b})
	if err != nil {
		t.Fatal(err)
	}
	for _, signer := range []*ALB{a, other} {
		for _, keyID := range []string{"", "api-key"} {
			var encoded string
			var generated ALBClaims
			if keyID == "" {
				encoded, generated, err = signer.Generate("user", []string{"read", "write"})
			} else {
				encoded, generated, err = signer.GenerateWithKeyID("user", []string{"read", "write"}, keyID)
			}
			if err != nil {
				t.Fatal(err)
			}
			if claims, err := v.Validate(encoded); err != nil || albClaimsJSON(t, claims) != albClaimsJSON(t, generated) {
				t.Fatalf("Validate() = (%+v, %v), want %+v", claims, err, generated)
			}
		}
	}
	now := time.Now()
	good := ALBClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: "issuer", Subject: "user", ID: "id", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}}
	for _, tc := range []struct {
		name   string
		change func(*jwt.Token, *ALBClaims)
		key    any
	}{
		{"malformed", nil, a.key},
		{"unknown key ID", func(token *jwt.Token, _ *ALBClaims) { token.Header["kid"] = "unknown" }, a.key},
		{"missing key ID", func(token *jwt.Token, _ *ALBClaims) { delete(token.Header, "kid") }, a.key},
		{"non string key ID", func(token *jwt.Token, _ *ALBClaims) { token.Header["kid"] = 42 }, a.key},
		{"wrong key", func(*jwt.Token, *ALBClaims) {}, other.key},
		{"wrong issuer", func(_ *jwt.Token, claims *ALBClaims) { claims.Issuer = "Issuer" }, a.key},
		{"missing issuer", func(_ *jwt.Token, claims *ALBClaims) { claims.Issuer = "" }, a.key},
		{"expired", func(_ *jwt.Token, claims *ALBClaims) { claims.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Second)) }, a.key},
		{"missing expiration", func(_ *jwt.Token, claims *ALBClaims) { claims.ExpiresAt = nil }, a.key},
		{"future issued at", func(_ *jwt.Token, claims *ALBClaims) { claims.IssuedAt = jwt.NewNumericDate(now.Add(time.Hour)) }, a.key},
		{"future not before", func(_ *jwt.Token, claims *ALBClaims) { claims.NotBefore = jwt.NewNumericDate(now.Add(time.Hour)) }, a.key},
		{"missing subject", func(_ *jwt.Token, claims *ALBClaims) { claims.Subject = "" }, a.key},
		{"missing token ID", func(_ *jwt.Token, claims *ALBClaims) { claims.ID = "" }, a.key},
		{"none", func(token *jwt.Token, _ *ALBClaims) { token.Method = jwt.SigningMethodNone }, jwt.UnsafeAllowNoneSignatureType},
		{"HS256 key confusion", func(token *jwt.Token, _ *ALBClaims) { token.Method = jwt.SigningMethodHS256 }, x509.MarshalPKCS1PublicKey(&a.key.PublicKey)},
		{"RS512", func(token *jwt.Token, _ *ALBClaims) { token.Method = jwt.SigningMethodRS512 }, a.key},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := good
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
			token.Header["kid"] = "key"
			encoded := "not a jwt"
			if tc.change != nil {
				tc.change(token, &claims)
				token.Header["alg"] = token.Method.Alg()
				var err error
				encoded, err = token.SignedString(tc.key)
				if err != nil {
					t.Fatal(err)
				}
			}
			if claims, err := v.Validate(encoded); err == nil || err.Error() != "invalid or expired token" || !reflect.DeepEqual(claims, ALBClaims{}) {
				t.Fatalf("Validate() = (%+v, %v), want empty claims and invalid or expired token", claims, err)
			}
		})
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, good)
	token.Header["kid"] = "key"
	encoded, err := token.SignedString(a.key)
	if err != nil {
		t.Fatal(err)
	}
	if claims, err := v.Validate(encoded); err != nil || albClaimsJSON(t, claims) != albClaimsJSON(t, good) {
		t.Fatalf("Validate() without optional claims = (%+v, %v), want %+v", claims, err, good)
	}
}

func TestALBValidateKeyIDHeader(t *testing.T) {
	a := albSigner(t, "key")
	claims := ALBClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: "issuer", Subject: "user", ID: "id", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	for _, tc := range []struct {
		name  string
		keyID any
		valid bool
	}{
		{"matching", "key", true},
		{"missing", nil, false},
		{"number", 42, false},
		{"boolean", true, false},
		{"array", []string{"key"}, false},
		{"object", map[string]string{"kid": "key"}, false},
		{"empty", "", false},
		{"unknown", "other", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			if tc.keyID != nil {
				token.Header["kid"] = tc.keyID
			}
			encoded, err := token.SignedString(a.key)
			if err != nil {
				t.Fatal(err)
			}
			validated, err := a.Validate(encoded)
			if tc.valid {
				if err != nil || albClaimsJSON(t, validated) != albClaimsJSON(t, claims) {
					t.Fatalf("Validate() with kid %v = (%+v, %v), want %+v", tc.keyID, validated, err, claims)
				}
			} else if err == nil || err.Error() != "invalid or expired token" || !reflect.DeepEqual(validated, ALBClaims{}) {
				t.Fatalf("Validate() with kid %v = (%+v, %v), want empty claims and invalid or expired token", tc.keyID, validated, err)
			}
		})
	}
}

func TestALBValidateZeroValue(t *testing.T) {
	signer := albSigner(t, "key")
	encoded, claims, err := signer.Generate("user", nil)
	if err != nil {
		t.Fatal(err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = ""
	emptyKeyID, err := token.SignedString(signer.key)
	if err != nil {
		t.Fatal(err)
	}
	a := &ALB{}
	for name, token := range map[string]string{
		"malformed":           "not a jwt",
		"signed":              encoded,
		"signed empty key ID": emptyKeyID,
	} {
		t.Run(name, func(t *testing.T) {
			if claims, err := a.Validate(token); err == nil || err.Error() != "invalid or expired token" || !reflect.DeepEqual(claims, ALBClaims{}) {
				t.Fatalf("Validate() = (%+v, %v), want empty claims and invalid or expired token", claims, err)
			}
		})
	}
}
