# Plan de prompts pour l’implémentation itérative

## Groupe 1 – Base commune et configuration (priorité CRITICAL)

### Fichiers :
- `web-app/go.mod`
- `api-service/go.mod`
- `oauth-server/go.mod`
- `docker-compose.yml`

**Prompt :**
> Nous construisons PicoSum, une POC Go avec trois services (web-app, api-service, oauth-server). Crée les fichiers `go.mod` pour chaque service avec les dépendances minimales : Fiber v2, log/slog, testify (pour tests). Pour api-service, ajoute swaggo/swag et swaggo/fiber-swagger. Ensuite, crée un `docker-compose.yml` qui définit les trois services, un réseau `poc-net`, expose les ports (web:8080, api:8081, oauth:8082), et utilise des variables d’environnement pour la configuration (API_URL, API_TOKEN). Assure-toi que les Dockerfiles seront créés plus tard ; pour l’instant, mets `build: ./web-app` etc. Inclus les tests d’intégration de base (vérification que les services répondent).

## Groupe 2 – Service API (calcul + Swagger)

### Fichiers :
- `api-service/main.go`
- `api-service/handlers/sum.go`
- `api-service/internal/calculator/sum.go`
- `api-service/middleware/request_id.go`
- `api-service/middleware/auth.go` (version simpliste token fixe)
- `api-service/docs/docs.go` (généré automatiquement)

**Prompt :**
> Implémente le service API complet avec les caractéristiques suivantes :
> - `main.go` : initialise Fiber, applique les middlewares (request_id, auth), monte la route GET `/sum` vers `SumHandler`, et sert Swagger sur `/swagger/*`.
> - `middleware/request_id.go` : génère un X-Request-ID si absent, l’ajoute dans le contexte et le log.
> - `middleware/auth.go` : vérifie l’en-tête `Authorization: Bearer <token>`. Le token attendu est lu depuis la variable d’environnement `API_TOKEN` (par défaut `poc-token-123`). Retourne 401 si invalide.
> - `handlers/sum.go` : extrait les paramètres `a` et `b` (GET query), valide qu’ils sont des entiers entre 0 et 10. En cas d’erreur, répond avec `{"error":"message"}` et code 400. En cas de succès, appelle `calculator.Add` et répond `{"sum": result}`. Ajoute les annotations Swagger nécessaires.
> - `calculator/sum.go` : simple fonction `Add(x, y int) int`.
> - Tests unitaires : pour `sum_test.go` (mock du contexte, test validation), `auth_test.go`, `request_id_test.go`, `calculator/sum_test.go`.
> - Génère la documentation Swagger avec `swag init` (indique les commandes dans le README).

## Groupe 3 – Client web-app (front + appel API)

### Fichiers :
- `web-app/main.go`
- `web-app/middleware/request_id.go`
- `web-app/handlers/calc.go`
- `web-app/internal/client/api_client.go`
- `web-app/static/index.html`
- `web-app/static/pico.min.css`
- `web-app/static/alpine.min.js` (optionnel)

**Prompt :**
> Développe l’application web qui sert une page HTML interactive.
> - `main.go` : configure Fiber, utilise `//go:embed` pour embarquer le dossier `static/`. Applique le middleware `request_id`. Définit la route GET `/` qui sert `index.html`. Définit la route POST `/sum` (car HTMX envoie POST) qui appelle `handlers.HandleSum`.
> - `handlers/calc.go` : reçoit les paramètres `a` et `b` du formulaire (via `c.FormValue`). Valide rapidement (0-10), puis utilise `api_client.CallSum` pour interroger le service API. En cas d’erreur, retourne un texte d’erreur (car HTMX attend du HTML ou du texte). En cas de succès, retourne la somme sous forme de chaîne.
> - `api_client.go` : construit l’URL à partir de `API_URL` (variable d’env), ajoute le token (`API_TOKEN`) dans l’en-tête, propage le X-Request-ID. Retourne le résultat ou une erreur.
> - `index.html` : page complète avec Alpine.js (état local, validation `isValid`), HTMX (envoi POST vers `/sum` avec `hx-vals`), Pico.css. Affiche le résultat dans `<div id="result">`. Gère l’affichage d’erreur locale si les champs sont hors bornes.
> - Tests : `calc_test.go` avec mock du client HTTP, `api_client_test.go`.

## Groupe 4 – Serveur OAuth simpliste

### Fichiers :
- `oauth-server/main.go`
- `oauth-server/Dockerfile`

**Prompt :**
> Implémente un serveur OAuth minimaliste pour la POC. Objectif : fournir un endpoint `/token` qui retourne un token fixe (ex: `{"access_token":"poc-token-123","token_type":"Bearer"}`) lors d’une requête POST avec des paramètres factices (client_id, client_secret). Utilise Fiber, écoute sur le port 8082. Pour gagner du temps, ne pas implémenter la validation des credentials ; tout appel POST à `/token` renvoie le même token. Ajoute des logs. Ce serveur est optionnel car l’api-service utilise déjà un token fixe, mais il permet de démontrer le flux OAuth2 complet. Le web-app devra échanger un client_id/client_secret (en dur) pour obtenir le token avant chaque appel (ou au démarrage). Cependant, pour simplifier, on peut faire en sorte que le web-app utilise directement le token fixe sans appeler le serveur OAuth – documente les deux approches. Fournis un Dockerfile.

## Groupe 5 – Intégration et tests finaux

### Fichiers :
- Tous les Dockerfiles
- `tests/integration/integration_test.go` (optionnel)
- `README.md`

**Prompt :**
> Finalise l’intégration :
> - Crée les Dockerfiles pour chaque service (multi-stage : `golang:1.21` -> binaire scratch ou alpine). Copie les fichiers nécessaires, expose les ports.
> - Ajuste `docker-compose.yml` pour utiliser les Dockerfiles et passer les variables d’environnement (`API_URL=http://api:8081`, `API_TOKEN=poc-token-123`, `OAUTH_URL=http://oauth:8082` optionnel).
> - Ajoute un `README.md` avec les instructions : `docker-compose up --build`, accès à http://localhost:8080, test du calcul, consultation des logs, accès Swagger sur http://localhost:8081/swagger/index.html.
> - (Optionnel) Écris un test d’intégration simple en Go qui lance les trois containers et vérifie le flux complet.
> - Vérifie que tous les tests unitaires passent (`go test ./...` dans chaque service).

