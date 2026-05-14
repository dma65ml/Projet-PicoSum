package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/picosum/api-service/middleware"
)

// newAuthApp crée une app Fiber avec AuthMiddleware chargé APRÈS le Setenv.
// Le token est lu au moment de l'appel à AuthMiddleware(), pas par requête.
func newAuthApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(middleware.AuthMiddleware())
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

func TestAuthMiddleware(t *testing.T) {
	t.Run("token valide depuis env", func(t *testing.T) {
		t.Setenv("API_TOKEN", "secret-test-token")
		app := newAuthApp(t)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer secret-test-token")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("token invalide", func(t *testing.T) {
		t.Setenv("API_TOKEN", "secret-test-token")
		app := newAuthApp(t)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer mauvais-token")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("en-tête Authorization absent", func(t *testing.T) {
		t.Setenv("API_TOKEN", "secret-test-token")
		app := newAuthApp(t)
		req := httptest.NewRequest("GET", "/protected", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("en-tête sans préfixe Bearer", func(t *testing.T) {
		t.Setenv("API_TOKEN", "secret-test-token")
		app := newAuthApp(t)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "secret-test-token") // manque "Bearer "
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("token par défaut quand env absent", func(t *testing.T) {
		t.Setenv("API_TOKEN", "") // force le défaut
		app := newAuthApp(t)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer poc-token-123")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}
