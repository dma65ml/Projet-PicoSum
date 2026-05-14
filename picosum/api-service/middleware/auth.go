package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const defaultDevToken = "poc-token-123"

// hmacKey normalise les longueurs avant la comparaison ; pas un secret partagé.
const hmacKey = "picosum-token-comparator-v1"

// AuthMiddleware lit le token au démarrage (une seule fois) et valide
// chaque requête par comparaison HMAC en temps constant.
func AuthMiddleware() fiber.Handler {
	expected := os.Getenv("API_TOKEN")
	switch {
	case expected == "":
		slog.Warn("[SECURITE] API_TOKEN non défini — token de développement utilisé. NE PAS déployer en production.")
		expected = defaultDevToken
	case expected == defaultDevToken:
		slog.Warn("[SECURITE] API_TOKEN utilise la valeur par défaut poc-token-123. Changer avant la mise en production.")
	}
	expectedMAC := tokenMAC(expected)

	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			slog.Warn("authentification échouée : en-tête absent ou malformé",
				"request_id", c.Locals("requestID"), "path", c.Path())
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token invalide ou absent"})
		}
		// Normalisation HMAC-SHA256 : ConstantTimeCompare retourne 0 immédiatement si
		// les slices ont des longueurs différentes — la normalisation élimine ce timing oracle (N2).
		provided := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare(tokenMAC(provided), expectedMAC) != 1 {
			slog.Warn("authentification échouée : token incorrect",
				"request_id", c.Locals("requestID"), "path", c.Path())
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token invalide ou absent"})
		}
		return c.Next()
	}
}

// tokenMAC retourne le HMAC-SHA256 d'un token avec une clé interne fixe.
func tokenMAC(token string) []byte {
	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(token))
	return mac.Sum(nil)
}
