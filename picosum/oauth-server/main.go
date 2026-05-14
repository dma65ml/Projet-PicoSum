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
		BodyLimit: 4 * 1024,
		// A3 : ProxyHeader configurable pour les déploiements derrière un reverse proxy de confiance.
		ProxyHeader: os.Getenv("TRUSTED_PROXY_HEADER"),
		// N1 : ne pas exposer les détails d'erreur internes
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			slog.Error("erreur interne", "status", code, "error", err.Error())
			return c.Status(code).JSON(fiber.Map{"error": http.StatusText(code)})
		},
	})

	// A3 : rate limiting — stockage en mémoire, réinitialisé au redémarrage (suffisant pour un POC).
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

	// Endpoint OAuth2 simplifié : retourne toujours le même token fixe
	app.Post("/token", handlers.TokenHandler)

	// Health check
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
