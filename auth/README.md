# Authentication

One module provides reusable Google ID-token verification and AWS ALB JWT
signing/validation. Applications choose which authenticator(s) they need by
creating process-lifetime instances at startup. The library has no HTTP server,
database access, or session storage; applications own authorization and state.

```go
googleAuth, err := auth.NewGoogle(auth.GoogleConfig{
    ClientID: clientID,
    AllowedDomains: []string{"example.com"},
})
albAuth, err := auth.NewALB(auth.ALBConfig{
    Issuer: issuer, KeyID: keyID, PrivateKeyPEM: privateKey,
})

user, err := googleAuth.Verify(ctx, credential)
token, claims, err := albAuth.Generate(user.Email, []string{"calls:read"})
validatedClaims, err := albAuth.Validate(token)
jwks, err := albAuth.JWKS()
```

`Generate` emits RS256 JWTs with standard issuer, subject, token ID, issued-at,
not-before, and expiry claims, plus a space-separated `scope` claim. `Validate`
checks signature, key ID, issuer, expiration, and issued-at. `JWKS` returns the
public key document for the configured signer. The default token lifetime is
24 hours and can be shortened through `ALBConfig.TTL`.
