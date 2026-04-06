package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the payload embedded inside every issued JWT.
// It carries the owner's identity and the set of sites they can access.
type Claims struct {
	UserID  string   `json:"user_id"`
	SiteIDs []string `json:"site_ids"` // sites this user owns or is a member of
	jwt.RegisteredClaims
}

// IssueToken creates and signs a JWT for a given user and their accessible sites.
// Call this from your login / session-create handler.
//
//	tokenStr, err := auth.IssueToken(secret, userID, []string{"site_abc"}, 24*time.Hour)
func IssueToken(secret []byte, userID string, siteIDs []string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()

	claims := Claims{
		UserID:  userID,
		SiteIDs: siteIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// parseToken verifies the signature and expiry, then returns the decoded claims.
func parseToken(secret []byte, raw string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (any, error) {
		// Guard against algorithm substitution attacks (e.g. none / RS256 confusion)
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// containsSiteID returns true if the given site is in the claims' allowed list.
func containsSiteID(claims *Claims, siteID string) bool {
	for _, s := range claims.SiteIDs {
		if s == siteID {
			return true
		}
	}
	return false
}
