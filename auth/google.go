package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/api/idtoken"
)

// GoogleConfig configures verification of Google Identity Services ID tokens.
type GoogleConfig struct {
	ClientID       string
	AllowedDomains []string
}

// GoogleUser contains the verified identity claims returned by Google.
type GoogleUser struct {
	Subject string
	Email   string
	Name    string
}

// Google is a reusable Google ID-token verifier. Create one per application
// configuration and reuse it for requests.
type Google struct {
	audience string
	domains  map[string]struct{}
}

var validateGoogleToken = idtoken.Validate

// NewGoogle validates the client and allowed hosted domains for token verification.
func NewGoogle(cfg GoogleConfig) (*Google, error) {
	if cfg.ClientID == "" {
		return nil, errors.New("Google client ID is required")
	}
	domains := make(map[string]struct{}, len(cfg.AllowedDomains))
	for _, domain := range cfg.AllowedDomains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain != "" {
			domains[domain] = struct{}{}
		}
	}
	if len(domains) == 0 {
		return nil, errors.New("at least one allowed Google hosted domain is required")
	}
	return &Google{audience: cfg.ClientID, domains: domains}, nil
}

// Verify validates a Google ID token and returns its verified user identity.
func (g *Google) Verify(ctx context.Context, credential string) (GoogleUser, error) {
	if credential == "" {
		return GoogleUser{}, errors.New("Google credential is required")
	}
	p, err := validateGoogleToken(ctx, credential, g.audience)
	if err != nil {
		return GoogleUser{}, errors.New("invalid Google credential")
	}
	if p.Issuer != "accounts.google.com" && p.Issuer != "https://accounts.google.com" {
		return GoogleUser{}, errors.New("invalid Google issuer")
	}
	email, _ := p.Claims["email"].(string)
	verified, _ := p.Claims["email_verified"].(bool)
	domain, _ := p.Claims["hd"].(string)
	name, _ := p.Claims["name"].(string)
	if email == "" || !verified || p.Expires <= time.Now().Unix() {
		return GoogleUser{}, errors.New("Google account email is not verified or credential is expired")
	}
	if _, ok := g.domains[strings.ToLower(domain)]; !ok {
		return GoogleUser{}, errors.New("Google hosted domain is not allowed")
	}
	return GoogleUser{Subject: p.Subject, Email: strings.ToLower(email), Name: name}, nil
}
