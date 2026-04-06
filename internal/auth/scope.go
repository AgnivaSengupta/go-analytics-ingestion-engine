package auth

import "github.com/gofiber/fiber/v2"

// AssertSiteScope returns 403 if the event's site_id doesn't match the
// authenticated site from IngestAuthMiddleware.
func AssertSiteScope(c *fiber.Ctx, eventSiteID string) error {
	authed, ok := c.Locals("authed_site").(AuthedSite)
	if !ok || authed.SiteID != eventSiteID {
		return c.Status(fiber.StatusForbidden).JSON(
			ApiErr("forbidden", "event site_id does not match authenticated site"),
		)
	}
	return nil
}
