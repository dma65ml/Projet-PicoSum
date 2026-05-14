// Package handlers contient les handlers HTTP de l'oauth-server.
package handlers

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
)

// TokenResponse est la structure de réponse standard OAuth2 (RFC 6749, section 5.1).
// Les tags `json:"..."` contrôlent les noms des champs dans la sérialisation JSON.
// La RFC impose ces noms exacts (access_token, token_type, expires_in).
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// TokenHandler implémente le flux "Client Credentials" d'OAuth2 (RFC 6749, section 4.4).
//
// Flux Client Credentials — utilisé pour les communications machine-à-machine :
//  1. Le client (web-app) envoie POST /token avec grant_type=client_credentials.
//  2. Le serveur valide le type de flux et retourne un access token.
//  3. Le client utilise ce token dans Authorization: Bearer <token> vers l'api-service.
//
// POC simplifiée : pas de validation du client_id/client_secret ni de JWT.
// En production, utiliser un serveur OAuth2 complet (Keycloak, Auth0, Dex…)
// qui émet des tokens JWT signés avec une durée de vie courte et révocables.
func TokenHandler(c *fiber.Ctx) error {
	// c.FormValue lit un champ d'un corps application/x-www-form-urlencoded.
	// La RFC 6749 impose ce format (pas JSON) pour les requêtes de token.
	grantType := c.FormValue("grant_type")

	// Validation du grant_type : seul "client_credentials" est supporté ici.
	// Un serveur OAuth2 complet gère aussi "authorization_code", "refresh_token"…
	if grantType != "client_credentials" {
		slog.Warn("grant_type non supporté", "grant_type", grantType)
		// "unsupported_grant_type" est le code d'erreur OAuth2 normalisé (RFC 6749, section 5.2).
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             "unsupported_grant_type",
			"error_description": "seul client_credentials est supporté",
		})
	}

	token := os.Getenv("API_TOKEN")
	if token == "" {
		// Avertissement visible dans les logs — en production API_TOKEN doit être défini.
		slog.Warn("[SECURITE] API_TOKEN non défini — token de développement utilisé. NE PAS déployer en production.")
		token = "poc-token-123"
	}

	slog.Info("token délivré", "client_id", c.FormValue("client_id"))

	// ExpiresIn est en secondes (3600 = 1 heure). En production, préférer des
	// durées courtes (ex. 5 min) avec un refresh token pour limiter la fenêtre
	// d'exploitation en cas de compromission du token.
	return c.JSON(TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	})
}
