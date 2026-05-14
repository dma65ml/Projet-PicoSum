package handlers

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
)

// TokenResponse est la réponse standard OAuth2 pour un token.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// TokenHandler retourne un token d'accès fixe.
// POC : pas de validation des credentials client_id/client_secret.
// N8 : validation minimale du grant_type conformément à RFC 6749.
func TokenHandler(c *fiber.Ctx) error {
	grantType := c.FormValue("grant_type")
	if grantType != "client_credentials" {
		slog.Warn("grant_type non supporté", "grant_type", grantType)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             "unsupported_grant_type",
			"error_description": "seul client_credentials est supporté",
		})
	}

	token := os.Getenv("API_TOKEN")
	if token == "" {
		slog.Warn("[SECURITE] API_TOKEN non défini — token de développement utilisé. NE PAS déployer en production.")
		token = "poc-token-123"
	}

	slog.Info("token délivré", "client_id", c.FormValue("client_id"))
	return c.JSON(TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	})
}
