package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/golang-jwt/jwt/v5"
)

// ALBVerifierConfig configures RS256 JWT verification using a public JWKS document.
type ALBVerifierConfig struct {
	Issuer string
	JWKS   []byte
}

// ALBVerifier validates tokens using public keys only.
// Create one per key configuration and reuse it for requests.
type ALBVerifier struct {
	issuer string
	keys   map[string]*rsa.PublicKey
}

// NewALBVerifier validates the public key configuration and creates a reusable verifier.
func NewALBVerifier(cfg ALBVerifierConfig) (*ALBVerifier, error) {
	if cfg.Issuer == "" {
		return nil, errors.New("issuer is required")
	}
	var jwks struct {
		Keys []struct {
			KeyID     string `json:"kid"`
			KeyType   string `json:"kty"`
			Algorithm string `json:"alg"`
			Use       string `json:"use"`
			N         string `json:"n"`
			E         string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(cfg.JWKS, &jwks); err != nil {
		return nil, errors.New("invalid JWKS")
	}
	if len(jwks.Keys) == 0 {
		return nil, errors.New("JWKS must contain at least one key")
	}
	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, key := range jwks.Keys {
		if key.KeyID == "" {
			return nil, errors.New("JWKS key ID is required")
		}
		if _, exists := keys[key.KeyID]; exists {
			return nil, errors.New("duplicate JWKS key ID")
		}
		// Decode the modulus to check its size, but report encoding errors last.
		n, nErr := base64.RawURLEncoding.DecodeString(key.N)
		modulus := new(big.Int).SetBytes(n)
		if key.KeyType != "RSA" || (nErr == nil && len(n) > 0 && modulus.BitLen() < 2048) {
			return nil, errors.New("JWKS key must be RSA with at least 2048 bits")
		}
		if (key.Algorithm != "" && key.Algorithm != "RS256") || (key.Use != "" && key.Use != "sig") {
			return nil, errors.New("JWKS key must be an RS256 signing key")
		}
		e, eErr := base64.RawURLEncoding.DecodeString(key.E)
		exponent := new(big.Int).SetBytes(e)
		if nErr != nil || len(n) == 0 || eErr != nil || exponent.BitLen() > 31 || exponent.Int64() < 2 || exponent.Bit(0) == 0 || modulus.Bit(0) == 0 {
			return nil, errors.New("invalid JWKS key encoding")
		}
		keys[key.KeyID] = &rsa.PublicKey{N: modulus, E: int(exponent.Int64())}
	}
	return &ALBVerifier{issuer: cfg.Issuer, keys: keys}, nil
}

// Validate verifies the signature, key ID, issuer, and registered time claims.
func (v *ALBVerifier) Validate(encoded string) (ALBClaims, error) {
	return validateALB(encoded, v.issuer, v.keys)
}

func validateALB(encoded, issuer string, keys map[string]*rsa.PublicKey) (ALBClaims, error) {
	claims := ALBClaims{}
	parsed, err := jwt.ParseWithClaims(encoded, &claims, func(token *jwt.Token) (any, error) {
		keyID, ok := token.Header["kid"].(string)
		key := keys[keyID]
		if token.Method != jwt.SigningMethodRS256 || !ok || key == nil {
			return nil, errors.New("unexpected signing key")
		}
		return key, nil
	}, jwt.WithIssuer(issuer), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !parsed.Valid || claims.ID == "" || claims.Subject == "" {
		return ALBClaims{}, errors.New("invalid or expired token")
	}
	return claims, nil
}
