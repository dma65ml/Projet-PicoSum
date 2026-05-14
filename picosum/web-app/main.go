// Package main est le point d'entrée de la web-app.
//
// Ce service sert l'interface utilisateur (HTML/HTMX/Pico.css) et relaie
// les calculs vers l'api-service. Il illustre plusieurs patterns Go :
//   - Fichiers statiques embarqués dans le binaire (//go:embed)
//   - Middlewares chaînés (traçabilité, sécurité, rate limiting)
//   - Dependency injection pour faciliter les tests (HandleSumWith)
//   - Content Security Policy stricte (CSP sans unsafe-*)
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

// //go:embed est une directive du compilateur Go (pas un commentaire ordinaire).
// Elle embarque le dossier "static" dans le binaire au moment de la compilation.
// Avantages : déploiement d'un seul fichier binaire, pas de dépendance au système
// de fichiers hôte, assets toujours synchronisés avec le code.
//
//go:embed static
var staticFiles embed.FS

func main() {
	initLogger()

	app := fiber.New(fiber.Config{
		// BodyLimit protège contre les requêtes volumineuses (ex. upload malveillant).
		BodyLimit: 4 * 1024,
		// ProxyHeader : à valoriser uniquement derrière un proxy de confiance (ici Caddy).
		// Laissé vide → Fiber utilise l'IP de connexion TCP directe.
		ProxyHeader: os.Getenv("TRUSTED_PROXY_HEADER"),
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			slog.Error("erreur interne", "status", code, "error", err.Error(),
				"request_id", c.Locals("requestID"))
			// SendString (pas JSON) : la web-app retourne du HTML, pas du JSON.
			return c.Status(code).SendString(http.StatusText(code))
		},
	})

	// Middleware 1 — Request ID : doit être en tête de chaîne pour que tous
	// les logs suivants incluent l'identifiant de corrélation.
	app.Use(middleware.RequestIDMiddleware())

	// Middleware 2 — En-têtes de sécurité : appliqués à toutes les réponses.
	app.Use(securityHeaders())

	// Middleware 3 — Rate limiting global : couvre toutes les routes (/static, /, /sum).
	// Stockage en mémoire → remis à zéro au redémarrage (suffisant pour un POC mono-instance).
	// En production multi-instances, utiliser un store partagé (Redis, Memcached).
	app.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Trop de requêtes. Réessayez dans un moment.")
		},
	}))

	// fs.Sub extrait le sous-dossier "static" de l'embed.FS.
	// Sans Sub, les chemins seraient "static/pico.min.css" au lieu de "pico.min.css".
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		slog.Error("erreur accès static embed", "error", err)
		os.Exit(1)
	}

	// filesystem.New sert les fichiers embarqués sous le préfixe /static/.
	// http.FS adapte un fs.FS (interface Go standard) en http.FileSystem (interface net/http).
	app.Use("/static", filesystem.New(filesystem.Config{Root: http.FS(sub)}))

	// Route racine : lit index.html depuis l'embed.FS et le sert avec le bon Content-Type.
	// On ne redirige pas vers /static/index.html pour garder l'URL propre (/).
	app.Get("/", func(c *fiber.Ctx) error {
		data, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			return fiber.ErrInternalServerError
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Send(data)
	})

	// client.NewClient() lit API_URL et API_TOKEN depuis l'environnement.
	// On passe le client au handler via HandleSumWith (dependency injection) :
	// les tests unitaires de calc.go peuvent injecter un mock à la place.
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

// securityHeaders pose les en-têtes de sécurité HTTP sur toutes les réponses.
//
// Content Security Policy (CSP) — directive la plus importante :
// Elle indique au navigateur quelles sources de contenu sont autorisées.
// "default-src 'self'" signifie : tout doit venir du même domaine, rien d'autre.
// Sous-directives précisant les règles par type de ressource :
//   - style-src 'self'   : seuls les CSS servis par ce serveur sont autorisés.
//     → Interdit les attributs style="..." inline (d'où la classe CSS error-response).
//   - script-src 'self'  : seuls les JS servis par ce serveur sont autorisés.
//     → Interdit les <script>...</script> inline et les attributs onclick="...".
//     → Incompatible avec Alpine.js v3 qui utilise new Function() (unsafe-eval).
//     → C'est pourquoi Alpine.js a été remplacé par du vanilla JS (app.js).
//   - img-src 'self' data: : autorise les images locales et les data URI (ex. icônes SVG).
//   - form-action 'self' : les formulaires ne peuvent soumettre que vers ce domaine.
//     → Protège contre les attaques open redirect via <form action="...externe">.
//
// HSTS (Strict-Transport-Security) :
// Force le navigateur à utiliser HTTPS exclusivement pendant 1 an.
// "preload" permet l'inscription dans la liste HSTS des navigateurs (protection dès
// la première visite, avant même la première réponse HTTPS). Inerte sur HTTP pur.
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
