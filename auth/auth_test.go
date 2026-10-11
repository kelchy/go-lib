package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
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
