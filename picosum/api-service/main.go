package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	_ "github.com/picosum/api-service/docs"
	"github.com/picosum/api-service/handlers"
	"github.com/picosum/api-service/middleware"
)

// @title PicoSum API
// @version 1.0
// @description API de calcul de somme pour la POC PicoSum
// @host localhost:8081
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	initLogger()

	app := fiber.New(fiber.Config{
		BodyLimit: 4 * 1024,
		// N5 : ProxyHeader configurable pour les déploiements derrière un reverse proxy de confiance.
		ProxyHeader: os.Getenv("TRUSTED_PROXY_HEADER"),
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			slog.Error("erreur interne", "status", code, "error", err.Error(),
				"request_id", c.Locals("requestID"))
			return c.Status(code).JSON(fiber.Map{"error": http.StatusText(code)})
		},
	})

	app.Use(middleware.RequestIDMiddleware())
	app.Use(securityHeaders())

	app.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "trop de requêtes, réessayez dans un moment",
			})
		},
	}))

	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	api := app.Group("/", middleware.AuthMiddleware())
	api.Get("/sum", handlers.SumHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	slog.Info("api-service démarré", "port", port)
	if err := app.Listen(":" + port); err != nil {
		slog.Error("erreur de démarrage", "error", err)
		os.Exit(1)
	}
}

func securityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// N6 : HSTS — inerte sur HTTP, obligatoire dès que TLS est activé
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		return c.Next()
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
