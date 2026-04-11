package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// AssertSiteScope returns 403 if the event's site_id doesn't match the
// authenticated site from IngestAuthMiddleware.
func AssertSiteScope(c *fiber.Ctx, eventSiteID string) error {
	authed, ok := c.Locals("authed_site").(AuthedSite)
	if !ok || authed.SiteID != eventSiteID {
		return errors.New("event site_id does not match authenticated site")
	}
	return nil
}
