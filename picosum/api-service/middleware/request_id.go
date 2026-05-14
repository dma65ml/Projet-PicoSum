// Package middleware regroupe les middlewares transverses de l'api-service.
//
// En HTTP, un middleware est une fonction qui s'intercale dans la chaîne de
// traitement d'une requête. Fiber applique les middlewares dans l'ordre où
// ils sont enregistrés avec app.Use(). Chaque middleware appelle c.Next()
// pour passer la main au suivant, ou retourne une erreur pour court-circuiter.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"regexp"

	"github.com/gofiber/fiber/v2"
)

// requestIDHeader est le nom de l'en-tête HTTP standard de corrélation de requête.
// Propagé de service en service, il permet de reconstituer la trace complète
// d'une transaction dans les logs : web-app → api-service → oauth-server.
const requestIDHeader = "X-Request-ID"

// validRequestID filtre les identifiants entrants par une expression régulière.
// Sans cette validation, un client malveillant pourrait injecter des séquences
// JSON ou des caractères de contrôle dans les logs (Log Injection, CWE-117).
// regexp.MustCompile panique au démarrage si le pattern est invalide — c'est
// voulu : une regex mal formée est un bug de programmation, pas une erreur runtime.
var validRequestID = regexp.MustCompile(`^[a-zA-Z0-9\-_]{1,64}$`)

// RequestIDMiddleware génère ou propage un identifiant de requête unique.
//
// Pattern "middleware factory" : la fonction retourne un fiber.Handler (closure).
// L'avantage est de pouvoir pré-calculer des valeurs (regex, config) une seule
// fois à l'enregistrement plutôt qu'à chaque requête.
//
// Comportement :
//   - Si l'en-tête X-Request-ID entrant est valide, on le réutilise (traçabilité
//     end-to-end depuis le client ou un autre service).
//   - Sinon, on génère 8 octets aléatoires cryptographiquement sûrs (crypto/rand)
//     encodés en hexadécimal → 16 caractères, collision quasi impossible.
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := c.Get(requestIDHeader)
		if !validRequestID.MatchString(reqID) {
			// crypto/rand (et non math/rand) garantit l'imprévisibilité,
			// empêchant un attaquant de deviner les IDs et de forger des logs.
			b := make([]byte, 8)
			_, _ = rand.Read(b) // rand.Read ne retourne jamais d'erreur en pratique
			reqID = hex.EncodeToString(b)
		}

		// c.Set pose l'en-tête sur la réponse (pour que les clients puissent
		// corréler leur requête avec les logs serveur).
		c.Set(requestIDHeader, reqID)

		// c.Locals stocke une valeur dans le contexte de la requête courante.
		// Les handlers et middlewares suivants y accèdent via c.Locals("requestID").
		// C'est l'équivalent fiber de context.WithValue en net/http standard.
		c.Locals("requestID", reqID)

		slog.Info("requête reçue",
			"request_id", reqID,
			"method", c.Method(),
			"path", c.Path(),
		)
		return c.Next()
	}
}
