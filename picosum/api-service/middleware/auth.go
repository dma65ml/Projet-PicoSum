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

// defaultDevToken est la valeur de repli utilisée uniquement en développement local.
// En production, API_TOKEN doit toujours être défini dans les variables d'environnement.
const defaultDevToken = "poc-token-123"

// hmacKey est la clé interne utilisée pour normaliser les tokens avant comparaison.
// Elle n'a pas besoin d'être secrète : son rôle est purement technique (voir tokenMAC).
const hmacKey = "picosum-token-comparator-v1"

// AuthMiddleware vérifie que chaque requête porte un token Bearer valide.
//
// Flux OAuth2 simplifié (POC) :
//  1. Le client (web-app) obtient un token auprès d'oauth-server (POST /token).
//  2. Il l'envoie dans l'en-tête HTTP : Authorization: Bearer <token>.
//  3. Ce middleware valide le token avant de laisser passer la requête.
//
// Pattern "middleware factory" avec pré-calcul : le token attendu et son HMAC
// sont calculés une seule fois au démarrage, puis réutilisés à chaque requête
// sans relire la variable d'environnement — plus efficace et thread-safe.
func AuthMiddleware() fiber.Handler {
	expected := os.Getenv("API_TOKEN")
	switch {
	case expected == "":
		// Avertissement visible dans les logs au démarrage du conteneur.
		slog.Warn("[SECURITE] API_TOKEN non défini — token de développement utilisé. NE PAS déployer en production.")
		expected = defaultDevToken
	case expected == defaultDevToken:
		slog.Warn("[SECURITE] API_TOKEN utilise la valeur par défaut poc-token-123. Changer avant la mise en production.")
	}

	// Pré-calcul du HMAC du token attendu (voir tokenMAC pour l'explication).
	expectedMAC := tokenMAC(expected)

	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")

		// Le schéma "Bearer" est défini par la RFC 6750 (OAuth2 Bearer Token).
		// strings.HasPrefix est sûr même si auth est vide.
		if !strings.HasPrefix(auth, "Bearer ") {
			slog.Warn("authentification échouée : en-tête absent ou malformé",
				"request_id", c.Locals("requestID"), "path", c.Path())
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token invalide ou absent"})
		}
		provided := strings.TrimPrefix(auth, "Bearer ")

		// subtle.ConstantTimeCompare compare deux slices en temps constant :
		// il ne court-circuite pas à la première différence, éliminant l'oracle de timing
		// (un attaquant ne peut pas déduire le token en mesurant les temps de réponse).
		//
		// Problème résiduel : ConstantTimeCompare retourne 0 immédiatement si les
		// longueurs diffèrent. Solution : normaliser via HMAC-SHA256 → toujours 32 octets,
		// quelle que soit la longueur du token fourni. (voir tokenMAC ci-dessous)
		if subtle.ConstantTimeCompare(tokenMAC(provided), expectedMAC) != 1 {
			slog.Warn("authentification échouée : token incorrect",
				"request_id", c.Locals("requestID"), "path", c.Path())
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token invalide ou absent"})
		}
		return c.Next()
	}
}

// tokenMAC produit un HMAC-SHA256 du token fourni avec une clé interne fixe.
//
// HMAC (Hash-based Message Authentication Code) applique SHA-256 au token
// pour produire toujours 32 octets en sortie, quelle que soit la longueur
// de l'entrée. Cela permet à subtle.ConstantTimeCompare de comparer des
// tranches de longueur identique dans tous les cas, éliminant l'oracle de timing
// lié aux longueurs différentes.
func tokenMAC(token string) []byte {
	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(token))
	return mac.Sum(nil)
}
