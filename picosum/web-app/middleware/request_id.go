// Package middleware regroupe les middlewares transverses de la web-app.
//
// Les middlewares s'intercalent dans la chaîne de traitement HTTP de Fiber.
// Ils sont enregistrés avec app.Use() et s'appliquent dans l'ordre de déclaration.
// Chaque middleware appelle c.Next() pour passer la main au suivant.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"regexp"

	"github.com/gofiber/fiber/v2"
)

// requestIDHeader est le nom de l'en-tête HTTP de corrélation de requête.
// Il permet de relier les logs de web-app à ceux d'api-service pour une même
// transaction utilisateur ("distributed tracing" simplifié).
const requestIDHeader = "X-Request-ID"

// validRequestID filtre les identifiants entrants pour prévenir l'injection de log
// (Log Injection, CWE-117) : sans validation, un client pourrait injecter des
// caractères spéciaux dans les logs via cet en-tête.
var validRequestID = regexp.MustCompile(`^[a-zA-Z0-9\-_]{1,64}$`)

// RequestIDMiddleware génère ou propage un identifiant de requête unique.
//
// Si le client envoie un X-Request-ID valide (ex. un navigateur ou un proxy),
// on le réutilise pour préserver la traçabilité de bout en bout.
// Sinon, on génère un identifiant aléatoire de 16 caractères hexadécimaux
// via crypto/rand (cryptographiquement sûr, contrairement à math/rand).
//
// L'ID est ensuite :
//   - Posé dans la réponse HTTP (header X-Request-ID) pour que le client puisse
//     le communiquer au support en cas de problème.
//   - Stocké dans c.Locals("requestID") pour être inclus dans tous les logs
//     des handlers suivants via c.Locals("requestID").
//   - Transmis à api-service dans l'en-tête de la requête sortante (voir api_client.go).
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := c.Get(requestIDHeader)
		if !validRequestID.MatchString(reqID) {
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
