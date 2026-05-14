package handlers_test

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/picosum/web-app/handlers"
)

// mockCaller est un mock du SumCaller pour les tests.
type mockCaller struct {
	sum int
	err error
}

func (m *mockCaller) CallSum(_ context.Context, _, _ int, _ string) (int, error) {
	return m.sum, m.err
}

func TestHandleSum(t *testing.T) {
	tests := []struct {
		name       string
		formBody   string
		caller     *mockCaller
		wantStatus int
		wantBody   string
	}{
		{
			name:       "succès 5+3",
			formBody:   "a=5&b=3",
			caller:     &mockCaller{sum: 8},
			wantStatus: 200,
			wantBody:   "5 + 3 = 8",
		},
		{
			name:       "succès 0+0",
			formBody:   "a=0&b=0",
			caller:     &mockCaller{sum: 0},
			wantStatus: 200,
			wantBody:   "0 + 0 = 0",
		},
		{
			name:       "a invalide (11)",
			formBody:   "a=11&b=3",
			caller:     &mockCaller{},
			wantStatus: 400,
			wantBody:   "Valeur A invalide",
		},
		{
			name:       "b invalide (négatif)",
			formBody:   "a=5&b=-1",
			caller:     &mockCaller{},
			wantStatus: 400,
			wantBody:   "Valeur B invalide",
		},
		{
			name:       "erreur API",
			formBody:   "a=3&b=4",
			caller:     &mockCaller{err: fmt.Errorf("service indisponible")},
			wantStatus: 502,
			wantBody:   "Erreur service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/sum", handlers.HandleSumWith(tt.caller))

			req := httptest.NewRequest("POST", "/sum",
				strings.NewReader(tt.formBody))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), tt.wantBody)
		})
	}
}
