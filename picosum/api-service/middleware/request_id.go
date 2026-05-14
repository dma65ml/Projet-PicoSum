package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"regexp"

	"github.com/gofiber/fiber/v2"
)

const requestIDHeader = "X-Request-ID"

// validRequestID accepte uniquement alphanumérique, tirets et underscores, 1-64 caractères.
// Cela empêche l'injection de caractères de contrôle ou de séquences JSON dans les logs (H2).
var validRequestID = regexp.MustCompile(`^[a-zA-Z0-9\-_]{1,64}$`)

// RequestIDMiddleware génère ou propage un identifiant de requête valide.
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := c.Get(requestIDHeader)
		if !validRequestID.MatchString(reqID) {
			// Header absent, trop long ou format invalide → on génère
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			reqID = hex.EncodeToString(b)
		}
		c.Set(requestIDHeader, reqID)
		c.Locals("requestID", reqID)
		slog.Info("requête reçue",
			"request_id", reqID,
			"method", c.Method(),
			"path", c.Path(),
		)
		return c.Next()
	}
}
