package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/picosum/web-app/internal/client"
)

func TestCallSum(t *testing.T) {
	t.Run("réponse correcte", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			assert.Equal(t, "req-id-1", r.Header.Get("X-Request-ID"))
			assert.Equal(t, "5", r.URL.Query().Get("a"))
			assert.Equal(t, "3", r.URL.Query().Get("b"))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]int{"sum": 8})
		}))
		defer srv.Close()

		t.Setenv("API_URL", srv.URL)
		t.Setenv("API_TOKEN", "test-token")
		c := client.NewClient()

		sum, err := c.CallSum(context.Background(), 5, 3, "req-id-1")
		require.NoError(t, err)
		assert.Equal(t, 8, sum)
	})

	t.Run("erreur HTTP 401", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "token invalide"})
		}))
		defer srv.Close()

		t.Setenv("API_URL", srv.URL)
		t.Setenv("API_TOKEN", "test-token")
		c := client.NewClient()

		_, err := c.CallSum(context.Background(), 5, 3, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token invalide")
	})
}
