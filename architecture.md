# Architecture technique de PicoSum

## Structure des dossiers
picosum/  
├── .devcontainer/  
│ ├── devcontainer.json  
│ └── docker-compose.yml  
├── Caddyfile (reverse proxy : routage, sécurité, HTTPS commenté)  
├── web-app/  
│ ├── Dockerfile  
│ ├── go.mod  
│ ├── main.go  
│ ├── handlers/  
│ │ └── calc.go  
│ ├── middleware/  
│ │ └── request_id.go  
│ ├── static/  
│ │ ├── index.html  
│ │ ├── pico.min.css  
│ │ ├── htmx.min.js  
│ │ ├── app.js (validation vanilla JS)  
│ │ └── app.css  
│ └── internal/  
│ └── client/  
│ └── api_client.go  
├── api-service/  
│ ├── Dockerfile  
│ ├── go.mod  
│ ├── main.go  
│ ├── handlers/  
│ │ └── sum.go  
│ ├── middleware/  
│ │ ├── auth.go (validation token fixe, HMAC-SHA256)  
│ │ └── request_id.go  
│ ├── docs/ (docs.go écrit manuellement)  
│ └── internal/  
│ └── calculator/  
│ └── sum.go  
├── oauth-server/  
│ ├── Dockerfile  
│ ├── go.mod  
│ ├── main.go  
│ └── handlers/  
│ └── token.go (endpoint /token, token fixe)  
└── tests/  
└── integration/ (optionnel)

text

## Choix technologiques avec justifications
| Technologie | Utilisation | Justification |
|-------------|-------------|---------------|
| Go 1.21+ | Langage principal | Simplicité, écosystème riche, binaires statiques |
| Fiber v2 | Framework HTTP (web-app + api-service + oauth-server) | Performance, middleware natif, `app.Test()` pour les tests |
| Caddy 2 | Reverse proxy | Configuration déclarative, HTTPS automatique (Let’s Encrypt/ZeroSSL) sans plugin |
| HTMX | Interactions AJAX | Réduit le JS, échange de fragments HTML |
| Vanilla JS | Validation UI | Remplace Alpine.js pour satisfaire la CSP `script-src ‘self’` sans `unsafe-eval` |
| Pico.css | Style CSS | Moderne, responsive, sans classes, embarquable |
| `//go:embed` | Embarquement des assets | Binaire unique, déploiement sans volume de fichiers |
| Swaggo | Documentation OpenAPI | Standard Go pour OpenAPI ; `docs.go` écrit manuellement (swag CLI absent) |
| `log/slog` | Logs structurés | Natif Go 1.21, format JSON, niveaux réglables par env |
| `testify` | Tests unitaires | Assertions claires, largement utilisé |
| Docker Compose | Orchestration locale | Idéal pour POC multi-containers |
## Patterns et conventions
- **Middleware** : chaque requête reçoit un `X-Request-ID` (généré ou propagé). Ce middleware s’applique à tous les services.
- **Communication** : web-app appelle api-service via HTTP avec en-tête `Authorization: Bearer <token>`. Le token est fixe pour la POC (ex: `poc-token-123`).
- **Configuration par variables d’environnement** :
  - `API_URL` (web-app) : adresse du service API
  - `OAUTH_URL` (web-app) : adresse du serveur OAuth (optionnel si token fixe)
  - `API_TOKEN` (api-service) : token attendu
- **Tests** : chaque package expose un test unitaire pour la fonction principale. Pour les handlers, utilisation de `app.Test()` de Fiber.
- **Logs** : tous les logs sortent sur stdout en JSON. Niveau réglable via `LOG_LEVEL` (debug, info, error).
- **Gestion d’erreur** : les erreurs sont retournées sous forme de JSON avec `{"error": "message"}` et code HTTP approprié.
## Diagrammes de flux de données
### Flux nominal (succès)

[Utilisateur] → POST /sum (via HTMX) → web-app (port 8080)  
web-app : génère X-Request-ID, appelle api-service:8081/sum? a=...& b=... avec token  
api-service : reçoit requête, vérifie token, calcule somme, retourne {"sum": N}  
web-app : transmet la réponse à l’utilisateur (injection dans #result)

text

### Flux avec erreur de validation (client)

Utilisateur → saisie hors bornes → Alpine.js désactive bouton, affiche message local (pas d’appel serveur)

text

### Flux avec erreur d’authentification

web-app → api-service (token invalide) → api-service retourne 401  
web-app → renvoie une erreur à l’utilisateur (via HTMX)

text

### Diagramme des containers

                          poc-net (bridge Docker)
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  ┌──────────┐   /swagger/*   ┌─────────────┐                   │
│  │          │ ─────────────► │ api-service │                   │
│  │  caddy   │                │    :8081    │                   │
│  │  :80 ◄───┤ (public)       └─────────────┘                   │
│  │          │   /*           ┌─────────────┐   ┌────────────┐  │
│  │          │ ─────────────► │   web-app   │──►│   oauth    │  │
│  └──────────┘                │    :8080    │   │   :8082    │  │
│                              └─────────────┘   └────────────┘  │
└─────────────────────────────────────────────────────────────────┘

Tous les services écrivent leurs logs sur stdout (JSON).
X-Forwarded-For propagé par Caddy → lu par le rate limiter des services Go.

text

## Stratégie de déploiement (Docker Compose)
- Le fichier `docker-compose.yml` définit quatre services : `caddy`, `web`, `api`, `oauth`.
- **Caddy** est le seul point d’entrée public (port 80). En production VPS, décommenter le port 443 et remplacer `:80` par le nom de domaine dans le `Caddyfile` pour activer le HTTPS automatique via Let’s Encrypt ou ZeroSSL.
- **web-app** et **api-service** n’exposent plus de port public ; ils communiquent uniquement sur le réseau bridge `poc-net`.
- `TRUSTED_PROXY_HEADER=X-Forwarded-For` configuré sur les services Go pour que le rate limiting lise l’IP réelle propagée par Caddy.
- Volumes `caddy_data` et `caddy_config` pour la persistance des certificats TLS (inactifs en mode HTTP local).
- Utilisation d’un réseau bridge `poc-net` pour l’isolation inter-services.
- Les dépendances sont gérées par `depends_on` (ordre de démarrage).
- Les binaires Go sont compilés dans des conteneurs multi-stage (`scratch`) pour minimiser la surface d’attaque.
## Gestion des logs centralisée
Bien que non obligatoire pour la POC, chaque conteneur écrit ses logs sur stdout. Docker Compose les collecte et les affiche avec `docker-compose logs`. Pour corréler les requêtes, le `X-Request-ID` est présent dans les logs des trois services.
