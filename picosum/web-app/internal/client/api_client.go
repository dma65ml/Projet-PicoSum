package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// SumCaller est l'interface du client pour appeler le service de somme.
type SumCaller interface {
	CallSum(ctx context.Context, a, b int, requestID string) (int, error)
}

// APIClient appelle l'api-service via HTTP.
type APIClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient crée un APIClient depuis les variables d'environnement API_URL et API_TOKEN.
func NewClient() *APIClient {
	baseURL := os.Getenv("API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}
	token := os.Getenv("API_TOKEN")
	if token == "" {
		slog.Warn("[SECURITE] API_TOKEN non défini — token de développement utilisé. NE PAS déployer en production.")
		token = "poc-token-123"
	}
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// CallSum envoie GET /sum?a=...&b=... à l'api-service et retourne la somme.
func (c *APIClient) CallSum(ctx context.Context, a, b int, requestID string) (int, error) {
	url := fmt.Sprintf("%s/sum?a=%d&b=%d", c.baseURL, a, b)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("création requête : %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("appel api-service : %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody.Error != "" {
			return 0, fmt.Errorf("api-service erreur %d : %s", resp.StatusCode, errBody.Error)
		}
		return 0, fmt.Errorf("api-service HTTP %d", resp.StatusCode)
	}

	var result struct {
		Sum int `json:"sum"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("décodage réponse : %w", err)
	}
	return result.Sum, nil
}
