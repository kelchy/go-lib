package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ALBConfig configures the RS256 JWTs validated by an AWS ALB listener.
type ALBConfig struct {
	Issuer        string
	KeyID         string
	PrivateKeyPEM []byte
	TTL           time.Duration
}

// ALBClaims contains the JWT identity, timing, and scope claims.
type ALBClaims struct {
	Scope string `json:"scope,omitempty"`
	jwt.RegisteredClaims
}

// ALB signs and validates tokens and exposes the corresponding JWKS document.
// Create one per key configuration and reuse it for requests.
type ALB struct {
	issuer string
	keyID  string
	ttl    time.Duration
	key    *rsa.PrivateKey
}

var signJWT = func(token *jwt.Token, key *rsa.PrivateKey) (string, error) {
	return token.SignedString(key)
}

// NewALB validates the signing configuration and creates a reusable ALB token handler.
func NewALB(cfg ALBConfig) (*ALB, error) {
	if cfg.Issuer == "" || cfg.KeyID == "" {
		return nil, errors.New("issuer and key ID are required")
	}
	block, _ := pem.Decode(cfg.PrivateKeyPEM)
	if block == nil {
		return nil, errors.New("private key must be PEM encoded")
	}
	var key *rsa.PrivateKey
	if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		key, _ = parsed.(*rsa.PrivateKey)
	} else {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, errors.New("invalid RSA private key")
		}
	}
	if key == nil || key.N.BitLen() < 2048 {
		return nil, errors.New("private key must be RSA with at least 2048 bits")
	}
	if cfg.TTL == 0 {
		cfg.TTL = 24 * time.Hour
	}
	if cfg.TTL < time.Minute || cfg.TTL > 24*time.Hour {
		return nil, errors.New("token TTL must be between one minute and 24 hours")
	}
	return &ALB{issuer: cfg.Issuer, keyID: cfg.KeyID, ttl: cfg.TTL, key: key}, nil
}

// Generate signs a token for subject with the supplied space-separated scopes.
func (a *ALB) Generate(subject string, scopes []string) (string, ALBClaims, error) {
	if subject == "" {
		return "", ALBClaims{}, errors.New("subject is required")
	}
	id, err := randomID()
	if err != nil {
		return "", ALBClaims{}, err
	}
	now := time.Now().UTC()
	claims := ALBClaims{Scope: strings.Join(scopes, " "), RegisteredClaims: jwt.RegisteredClaims{
		Issuer: a.issuer, Subject: subject, ID: id, IssuedAt: jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(a.ttl)),
	}}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = a.keyID
	encoded, err := signJWT(token, a.key)
	return encoded, claims, err
}

// Validate verifies the signature, key ID, issuer, and registered time claims.
func (a *ALB) Validate(encoded string) (ALBClaims, error) {
	claims := ALBClaims{}
	parsed, err := jwt.ParseWithClaims(encoded, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 || token.Header["kid"] != a.keyID {
			return nil, errors.New("unexpected signing key")
		}
		return &a.key.PublicKey, nil
	}, jwt.WithIssuer(a.issuer), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !parsed.Valid || claims.ID == "" || claims.Subject == "" {
		return ALBClaims{}, errors.New("invalid or expired token")
	}
	return claims, nil
}

// JWKS returns the JSON Web Key Set containing this signer's public key.
func (a *ALB) JWKS() ([]byte, error) {
	pub := &a.key.PublicKey
	e := big.NewInt(int64(pub.E)).Bytes()
	return json.Marshal(map[string]any{"keys": []any{map[string]string{
		"kty": "RSA", "use": "sig", "alg": "RS256", "kid": a.keyID,
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(e),
	}}})
}
