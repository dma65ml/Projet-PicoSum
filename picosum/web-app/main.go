package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"github.com/picosum/web-app/handlers"
	"github.com/picosum/web-app/internal/client"
	"github.com/picosum/web-app/middleware"
)

//go:embed static
var staticFiles embed.FS

func main() {
	initLogger()

	app := fiber.New(fiber.Config{
		BodyLimit: 4 * 1024,
		// N5 : ProxyHeader configurable via env pour lire l'IP réelle derrière un proxy de confiance.
		// Laisser vide (défaut) si l'appli est exposée directement — sinon risque de spoofing IP.
		ProxyHeader: os.Getenv("TRUSTED_PROXY_HEADER"),
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			slog.Error("erreur interne", "status", code, "error", err.Error(),
				"request_id", c.Locals("requestID"))
			return c.Status(code).SendString(http.StatusText(code))
		},
	})

	app.Use(middleware.RequestIDMiddleware())
	app.Use(securityHeaders())

	// N9 : rate limiting global (couvre /, /static, /sum)
	app.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Trop de requêtes. Réessayez dans un moment.")
		},
	}))

	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		slog.Error("erreur accès static embed", "error", err)
		os.Exit(1)
	}
	app.Use("/static", filesystem.New(filesystem.Config{Root: http.FS(sub)}))

	app.Get("/", func(c *fiber.Ctx) error {
		data, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			return fiber.ErrInternalServerError
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Send(data)
	})

	apiClient := client.NewClient()
	app.Post("/sum", handlers.HandleSumWith(apiClient))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("web-app démarrée", "port", port)
	if err := app.Listen(":" + port); err != nil {
		slog.Error("erreur de démarrage", "error", err)
		os.Exit(1)
	}
}

// securityHeaders pose les en-têtes HTTP de sécurité sur toutes les réponses.
// N4 : CSP sans aucun unsafe-* — rendu possible par la suppression d'Alpine.js
//      et l'externalisation des styles dans app.css.
// N6 : HSTS activé (inerte sur HTTP, obligatoire dès que TLS est en place).
func securityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"style-src 'self'; "+
				"script-src 'self'; "+
				"img-src 'self' data:; "+
				"font-src 'self'; "+
				"form-action 'self'",
		)
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
