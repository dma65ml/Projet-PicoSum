// Package client fournit le client HTTP pour appeler l'api-service.
//
// En Go, on définit une interface côté consommateur (ici SumCaller dans ce package)
// plutôt que côté fournisseur. C'est le principe "accept interfaces, return structs" :
// le handler (calc.go) dépend de l'interface légère, pas de la struct concrète.
// Cela rend les tests unitaires possibles sans démarrer un vrai serveur HTTP.
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

// SumCaller est l'interface minimale attendue par le handler calc.go.
//
// En Go, les interfaces sont satisfaites implicitement : APIClient implémente
// SumCaller sans le déclarer explicitement. Une interface à une seule méthode
// est idiomatique en Go (io.Reader, io.Writer, fmt.Stringer…).
type SumCaller interface {
	CallSum(ctx context.Context, a, b int, requestID string) (int, error)
}

// APIClient est le client HTTP concret vers l'api-service.
// Les champs sont en minuscules (non exportés) : seul ce package peut les modifier,
// ce qui garantit que l'état interne n'est accessible qu'via les méthodes publiques.
type APIClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient crée un APIClient configuré depuis les variables d'environnement.
//
// Configuration par variables d'environnement (principe 12-Factor App) :
// le comportement change selon l'environnement (dev/prod) sans recompiler.
// API_URL et API_TOKEN sont injectés par docker-compose.yml.
func NewClient() *APIClient {
	baseURL := os.Getenv("API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}

	token := os.Getenv("API_TOKEN")
	if token == "" {
		// Avertissement explicite : le token de dev ne doit jamais arriver en production.
		slog.Warn("[SECURITE] API_TOKEN non défini — token de développement utilisé. NE PAS déployer en production.")
		token = "poc-token-123"
	}

	return &APIClient{
		baseURL: baseURL,
		token:   token,
		// http.Client avec Timeout est indispensable : sans timeout, une connexion
		// lente ou un service qui ne répond pas bloque la goroutine indéfiniment,
		// épuisant le pool et provoquant un déni de service en cascade.
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

// CallSum envoie GET /sum?a=...&b=... à l'api-service et retourne la somme.
//
// Le paramètre ctx (context.Context) est fondamental en Go pour la propagation
// des annulations. Si le navigateur ferme la connexion avant la réponse, Fiber
// annule le contexte → http.NewRequestWithContext l'annule → l'appel TCP se coupe.
// Sans contexte, on gaspillerait des ressources serveur pour un client déjà parti.
func (c *APIClient) CallSum(ctx context.Context, a, b int, requestID string) (int, error) {
	url := fmt.Sprintf("%s/sum?a=%d&b=%d", c.baseURL, a, b)

	// http.NewRequestWithContext lie la requête au contexte parent.
	// Si ctx est annulé (timeout ou déconnexion client), la requête HTTP est abandonnée.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("création requête : %w", err)
	}

	// En-tête d'authentification OAuth2 Bearer (RFC 6750).
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Propagation du request ID pour corréler les logs entre web-app et api-service.
	// Sans cela, impossible de retrouver dans les logs api-service quelle requête
	// correspond à quel appel web-app.
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("appel api-service : %w", err)
	}
	// defer garantit la fermeture du body même en cas de retour anticipé (erreur).
	// Sans defer resp.Body.Close(), la connexion TCP reste ouverte et le pool
	// de connexions se sature progressivement (fuite de ressource).
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// On essaie de lire le message d'erreur JSON retourné par l'api-service.
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody.Error != "" {
			return 0, fmt.Errorf("api-service erreur %d : %s", resp.StatusCode, errBody.Error)
		}
		return 0, fmt.Errorf("api-service HTTP %d", resp.StatusCode)
	}

	// Décodage de la réponse JSON { "sum": N } dans une struct anonyme.
	// Les structs anonymes sont idiomatiques en Go pour un usage ponctuel
	// sans polluer le package avec un type nommé.
	var result struct {
		Sum int `json:"sum"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("décodage réponse : %w", err)
	}
	return result.Sum, nil
}
