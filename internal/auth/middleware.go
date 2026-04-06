package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthedSite struct {
	SiteID string
}

// Ingest Middleware -
// Takes the public api key and checks its hash with the DB to give the siteID.
// If the siteId is present in the API key table and the key is not revoked -- the Ingest Request is Authenticated..
func IngestAuthMiddleware(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := bearerToken(c)

		if raw == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ApiErr("unauthorized", "invalid or revoked site key"))
		}

		hash := sha256Hex(raw)

		var siteID string
		err := db.QueryRow(context.Background(),
			`SELECT site_id FROM api_keys
			 WHERE key_hash = $1
				AND key_type = 'public'
				AND revoked_at IS NULL
			`,
			hash,
		).Scan(&siteID)

		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ApiErr("unauthorized", "invalid or revoked site key"))
		}

		c.Locals("authed_site", AuthedSite{SiteID: siteID})
		return c.Next()
	}
}

// QueryAuthMiddleware validates a JWT for dashboard read endpoints.
// It checks:
//  1. The Authorization header contains a valid, non-expired Bearer JWT.
//  2. The JWT was signed with jwtSecret (HMAC-SHA256).
//  3. The requested :site_id path param is listed in the token's site_ids claim.
func QueryAuthMiddleware(jwtSecret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := bearerToken(c)
		if raw == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ApiErr("unauthorized", "missing or malformed authorization header"))
		}

		claims, err := parseToken(jwtSecret, raw)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ApiErr("unauthorized", "invalid or expired token"))
		}

		// Enforce site-level access — the token must list this site.
		siteID := c.Params("site_id")
		if siteID != "" && !containsSiteID(claims, siteID) {
			return c.Status(fiber.StatusForbidden).JSON(ApiErr("forbidden", "token does not grant access to this site"))
		}

		// Stash claims so downstream handlers can read user_id if needed.
		c.Locals("jwt_claims", claims)
		return c.Next()
	}
}

func bearerToken(c *fiber.Ctx) string {
	h := c.Get("Authorization")
	if len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}

	return ""
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func ApiErr(code, msg string) fiber.Map {
	return fiber.Map{"error": fiber.Map{"code": code, "message": msg}}
}
