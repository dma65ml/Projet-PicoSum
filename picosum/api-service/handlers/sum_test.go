package handlers_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/picosum/api-service/handlers"
)

func TestSumHandler(t *testing.T) {
	app := fiber.New()
	app.Get("/sum", handlers.SumHandler)

	tests := []struct {
		name    string
		query   string
		status  int
		wantSum *int
	}{
		{"5+3=8", "?a=5&b=3", 200, intPtr(8)},
		{"0+0=0", "?a=0&b=0", 200, intPtr(0)},
		{"10+10=20", "?a=10&b=10", 200, intPtr(20)},
		{"1+9=10", "?a=1&b=9", 200, intPtr(10)},
		{"a=11 hors bornes", "?a=11&b=3", 400, nil},
		{"b=11 hors bornes", "?a=5&b=11", 400, nil},
		{"a négatif", "?a=-1&b=3", 400, nil},
		{"b négatif", "?a=5&b=-1", 400, nil},
		{"a non numérique", "?a=abc&b=3", 400, nil},
		{"b non numérique", "?a=5&b=xyz", 400, nil},
		{"paramètres manquants", "", 400, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/sum"+tt.query, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.status, resp.StatusCode)

			if tt.wantSum != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				var result map[string]int
				require.NoError(t, json.Unmarshal(body, &result))
				assert.Equal(t, *tt.wantSum, result["sum"])
			}
		})
	}
}

func intPtr(n int) *int { return &n }
