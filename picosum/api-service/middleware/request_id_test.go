package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/picosum/api-service/middleware"
)

func TestRequestIDMiddleware(t *testing.T) {
	app := fiber.New()
	app.Use(middleware.RequestIDMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString(c.Get("X-Request-ID"))
	})

	t.Run("génère un request ID si absent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
	})

	t.Run("propage le request ID existant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "test-id-123")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, "test-id-123", resp.Header.Get("X-Request-ID"))
	})
}
