// Package main est le point d'entrée de l'oauth-server.
//
// Ce service implémente un serveur OAuth2 minimal pour la POC.
// Son unique rôle est de délivrer un token fixe via le flux "client_credentials"
// (RFC 6749, section 4.4), utilisé pour les communications machine-à-machine
// sans intervention d'un utilisateur humain.
//
// En production, remplacer ce service par un serveur OAuth2 complet
// (Keycloak, Auth0, Dex, Ory Hydra…) qui gère :
//   - La validation du client_id / client_secret
//   - L'émission de tokens JWT signés (RS256 ou ES256)
//   - La rotation et la révocation des tokens
//   - L'introspection de token (RFC 7662)
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"github.com/picosum/oauth-server/handlers"
)

func main() {
	initLogger()

	app := fiber.New(fiber.Config{
		// BodyLimit : protège contre les corps de requête volumineux.
		// L'endpoint /token n'attend que quelques paramètres de formulaire.
		BodyLimit: 4 * 1024,
		// ProxyHeader : lit l'IP réelle du client quand derrière un proxy (Caddy).
		// Laisser vide en accès direct pour éviter le spoofing IP.
		ProxyHeader: os.Getenv("TRUSTED_PROXY_HEADER"),
		// ErrorHandler centralisé : évite d'exposer des détails internes dans les erreurs.
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			slog.Error("erreur interne", "status", code, "error", err.Error())
			return c.Status(code).JSON(fiber.Map{"error": http.StatusText(code)})
		},
	})

	// Rate limiting sur l'endpoint /token : protège contre le brute-force
	// des credentials (client_id/client_secret) et les abus en rafale.
	// 30 req/min est plus restrictif que les autres services car /token est
	// une cible privilégiée pour les attaques d'authentification.
	// Stockage en mémoire → compteurs remis à zéro au redémarrage du processus.
	app.Use(limiter.New(limiter.Config{
		Max:        30,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "trop de requêtes, réessayez dans un moment",
			})
		},
	}))

	// POST /token : point d'entrée OAuth2 standard (RFC 6749, section 3.2).
	// La méthode POST est imposée par la RFC (pas GET) pour éviter que le
	// client_secret apparaisse dans l'URL et donc dans les logs de proxy.
	app.Post("/token", handlers.TokenHandler)

	// GET /health : endpoint de vérification de santé pour Docker / orchestrateurs.
	// Retourne 200 OK si le service est démarré et capable de répondre.
	// Docker Compose, Kubernetes et les load balancers sondent cet endpoint
	// pour décider si le conteneur est prêt à recevoir du trafic.
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	slog.Info("oauth-server démarré", "port", port,
		"note", "POC : token fixe, pas de validation des credentials",
	)
	if err := app.Listen(":" + port); err != nil {
		slog.Error("erreur de démarrage", "error", err)
		os.Exit(1)
	}
}

// initLogger configure log/slog comme logger global JSON.
// Voir api-service/main.go pour l'explication complète.
func initLogger() {
	level := slog.LevelInfo
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}
