// Package main est le point d'entrée de l'api-service.
//
// Ce service expose une API REST GET /sum?a=N&b=M protégée par un token Bearer.
// Il illustre les patterns Go courants pour un microservice HTTP :
//   - Fiber comme framework HTTP performant (FastHTTP sous le capot)
//   - Middlewares chaînés pour la traçabilité, la sécurité et le rate limiting
//   - Configuration par variables d'environnement (12-Factor App)
//   - Documentation OpenAPI via annotations Swaggo
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	// Import blank "_" : déclenche l'init() de docs.go qui enregistre la spec Swagger.
	_ "github.com/picosum/api-service/docs"
	"github.com/picosum/api-service/handlers"
	"github.com/picosum/api-service/middleware"
)

// Les annotations @title, @version… sont lues par swaggo pour générer docs/docs.go.
// Elles apparaissent dans l'interface Swagger UI accessible sur /swagger/index.html.
//
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

	// fiber.Config permet de personnaliser le comportement du serveur.
	app := fiber.New(fiber.Config{
		// BodyLimit protège contre les attaques par corps de requête volumineux
		// (ex. envoi d'un fichier de 1 Go pour saturer la mémoire).
		// 4 Ko est largement suffisant pour des paramètres de formulaire simples.
		BodyLimit: 4 * 1024,

		// ProxyHeader indique quel en-tête HTTP contient l'IP réelle du client
		// quand le service est derrière un reverse proxy (ici Caddy avec X-Forwarded-For).
		// Laisser vide en accès direct pour éviter le spoofing IP.
		ProxyHeader: os.Getenv("TRUSTED_PROXY_HEADER"),

		// ErrorHandler centralise la gestion des erreurs : un seul endroit pour
		// formater les réponses d'erreur de manière cohérente et sécurisée.
		// Sans cela, Fiber expose des détails internes dans la réponse.
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			// *fiber.Error (ex. fiber.ErrBadRequest) transporte un code HTTP.
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			slog.Error("erreur interne", "status", code, "error", err.Error(),
				"request_id", c.Locals("requestID"))
			// http.StatusText retourne le texte standard (ex. "Bad Request") —
			// jamais de stack trace ni de message interne dans la réponse client.
			return c.Status(code).JSON(fiber.Map{"error": http.StatusText(code)})
		},
	})

	// Ordre des middlewares : chaque middleware s'applique aux routes déclarées après lui.
	// 1. RequestID : doit être en premier pour que tous les logs suivants l'incluent.
	app.Use(middleware.RequestIDMiddleware())

	// 2. En-têtes de sécurité : appliqués à toutes les réponses, y compris les erreurs.
	app.Use(securityHeaders())

	// 3. Rate limiting : limite à 60 requêtes/minute/IP pour freiner les abus.
	//    Stockage en mémoire → compteurs remis à zéro au redémarrage du processus.
	//    Pour un déploiement multi-instances, utiliser un store Redis à la place.
	app.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		// KeyGenerator définit la granularité du compteur. Ici par adresse IP.
		// c.IP() lit TRUSTED_PROXY_HEADER si configuré, sinon l'IP de connexion TCP.
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "trop de requêtes, réessayez dans un moment",
			})
		},
	}))

	// Route Swagger UI : accessible sans authentification pour faciliter les tests.
	// fiberSwagger.WrapHandler sert l'interface HTML + les assets JS/CSS de Swagger.
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Groupe de routes protégées par le middleware d'authentification.
	// app.Group permet d'appliquer un middleware uniquement à un sous-ensemble de routes.
	api := app.Group("/", middleware.AuthMiddleware())
	api.Get("/sum", handlers.SumHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	slog.Info("api-service démarré", "port", port)

	// app.Listen bloque jusqu'à l'arrêt du serveur (signal OS ou erreur réseau).
	if err := app.Listen(":" + port); err != nil {
		slog.Error("erreur de démarrage", "error", err)
		os.Exit(1)
	}
}

// securityHeaders pose les en-têtes de sécurité HTTP sur toutes les réponses.
//
// Ces en-têtes sont une ligne de défense supplémentaire côté navigateur :
//   - X-Content-Type-Options : empêche le navigateur de deviner le type MIME
//     (MIME sniffing), évitant qu'un fichier texte soit exécuté comme du JS.
//   - X-Frame-Options : interdit l'intégration dans un <iframe>, protège contre
//     le clickjacking (l'utilisateur pense cliquer sur A mais clique sur B).
//   - Referrer-Policy : limite les informations envoyées dans l'en-tête Referer
//     lors de navigations vers des sites externes.
//   - Permissions-Policy : désactive les API navigateur non utilisées (caméra, micro…)
//     pour réduire la surface d'attaque en cas de XSS.
//   - Strict-Transport-Security (HSTS) : force le navigateur à n'utiliser que HTTPS
//     pour ce domaine pendant 1 an. "preload" permet l'inscription dans la liste
//     HSTS préconstruite des navigateurs (protection dès la première visite).
//     Inerte sur HTTP pur — ne prend effet qu'avec TLS actif (ex. derrière Caddy).
func securityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		return c.Next()
	}
}

// initLogger configure log/slog comme logger global de l'application.
//
// slog (Go 1.21) unifie les logs structurés dans la bibliothèque standard.
// Le format JSON (NewJSONHandler) est idéal pour les environnements conteneurisés
// où les logs sont collectés par un agrégateur (Loki, Datadog, CloudWatch…)
// qui sait parser le JSON pour indexer les champs.
//
// Le niveau est réglable via LOG_LEVEL sans recompiler : utile pour activer
// les logs debug en production temporairement sans redéployer une nouvelle image.
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
