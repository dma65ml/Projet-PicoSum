# Architecture technique de PicoSum

## Structure des dossiers
picosum/  
├── .devcontainer/  
│ ├── devcontainer.json  
│ └── docker-compose.yml  
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
│ │ └── alpine.min.js (optionnel, si embarqué)  
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
│ │ ├── auth.go (validation token fixe)  
│ │ └── request_id.go  
│ ├── docs/ (généré par swag init)  
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
| Technologie | Utilisation | Justification (POC 1 jour) |
|-------------|-------------|-----------------------------|
| Go 1.21+ | Langage principal | Simplicité, écosystème riche, binaires statiques |
| Fiber v2 | Framework HTTP (web-app + api-service) | Performance, facile à prendre en main, middleware natif |
| HTMX | Interactions AJAX | Réduit le JS à écrire, s’intègre bien avec Alpine |
| Alpine.js | État local et validation UI | Léger (<15ko), réactif sans framework lourd |
| Pico.css | Style CSS | Moderne, responsive, sans classes, facilite l’embarquement |
| `//go:embed` | Embarquement des assets | Binaire unique, déploiement simple |
| Swaggo | Documentation OpenAPI | Génération automatique, standard Go |
| `log/slog` | Logs structurés | Natif en Go 1.21, format JSON, niveaux de log |
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

+-------------+ HTTP +---------------+ HTTP +----------------+  
| web-app | ------------> | api-service | ------------> | oauth-server |  
| :8080 | (avec token) | :8081 | (vérif token) | (optionnel) |  
+-------------+ +---------------+ +----------------+  
| |  
| (logs) | (logs)  
v v  
stdout JSON stdout JSON

text

## Stratégie de déploiement (Docker Compose)
- Le fichier `docker-compose.yml` définit trois services : `web`, `api`, `oauth`.
- Utilisation d’un réseau bridge `poc-net` pour la communication.
- Les dépendances sont gérées par `depends_on` (ordre de démarrage).
- Les binaires Go sont compilés dans des conteneurs multi-stage pour réduire la taille.
- Les ports exposés : `8080:8080` (web-app) accessible depuis l’hôte.
## Gestion des logs centralisée
Bien que non obligatoire pour la POC, chaque conteneur écrit ses logs sur stdout. Docker Compose les collecte et les affiche avec `docker-compose logs`. Pour corréler les requêtes, le `X-Request-ID` est présent dans les logs des trois services.
