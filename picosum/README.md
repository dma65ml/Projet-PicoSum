# PicoSum

POC microservices Go — calculatrice distribuée avec authentification OAuth2 simpliste.

## Architecture

```
[Navigateur] → [Caddy:80] ──┬──→ [web-app:8080]
                             └──→ [api-service:8081]  (/swagger/*)
                                         ↑
                                  [oauth-server:8082]
```

Quatre services : un reverse proxy Caddy et trois services Go compilés statiquement.  
Caddy est le seul point d'entrée public — web-app et api-service ne sont pas exposés directement.

## Démarrage rapide

### Prérequis

- Docker & Docker Compose
- Go 1.21+ (pour développement local)

### Lancer avec Docker Compose

```bash
docker-compose up --build
```

| URL                                   | Description              |
|---------------------------------------|--------------------------|
| http://localhost                      | Interface utilisateur     |
| http://localhost/swagger/index.html   | Documentation Swagger API |
| http://localhost:8082/token (POST)    | Token OAuth2 — dev only   |

### Développement local (sans Docker)

Lancer chaque service dans un terminal séparé :

```bash
# Terminal 1 – api-service
cd api-service
API_TOKEN=poc-token-123 go run .

# Terminal 2 – oauth-server
cd oauth-server
go run .

# Terminal 3 – web-app
cd web-app
API_URL=http://localhost:8081 API_TOKEN=poc-token-123 go run .
```

## Tests unitaires

```bash
# Dans chaque service
cd api-service && go test ./...
cd web-app && go test ./...
cd oauth-server && go build ./...  # (pas de tests séparés pour oauth)
```

## Variables d'environnement

| Variable   | Service     | Valeur par défaut      | Description                        |
|------------|-------------|------------------------|------------------------------------|
| `API_URL`  | web-app     | `http://localhost:8081`| URL de l'api-service               |
| `API_TOKEN`| web-app, api| `poc-token-123`        | Token Bearer partagé               |
| `OAUTH_URL`| web-app     | `http://localhost:8082`| URL du serveur OAuth (optionnel)   |
| `LOG_LEVEL`| tous        | `info`                 | Niveau de log (debug/info/error)   |
| `PORT`     | tous        | selon service          | Port d'écoute                      |

## Flux d'authentification

Pour la POC, un token fixe `poc-token-123` est utilisé. En production, remplacer par un vrai flux OAuth2 :

1. web-app appelle `POST /token` sur oauth-server pour obtenir un access token
2. web-app utilise ce token dans `Authorization: Bearer <token>` vers api-service
3. api-service valide le token via le middleware `AuthMiddleware`

Pour une intégration complète, désactiver le token fixe et implémenter la validation JWT.

## Génération Swagger

Si `swag` CLI est installé :

```bash
cd api-service
go install github.com/swaggo/swag/cmd/swag@latest
swag init
```

Le fichier `docs/docs.go` est inclus manuellement pour la POC.

## Structure

```
picosum/
├── Caddyfile           # Reverse proxy : routage web + Swagger, HTTPS commenté
├── docker-compose.yml
├── api-service/        # Service API (GET /sum, Swagger)
│   ├── handlers/       # Handler HTTP
│   ├── middleware/     # Auth + Request-ID
│   ├── internal/calculator/  # Logique métier pure
│   └── docs/           # Documentation Swagger
├── web-app/            # Application web (HTML/HTMX/Pico.css)
│   ├── handlers/       # Handler formulaire
│   ├── middleware/     # Request-ID
│   ├── internal/client/  # Client HTTP vers api-service
│   └── static/         # Assets embarqués (pico.css, htmx.js, app.js)
├── oauth-server/       # Serveur OAuth2 minimal
│   └── handlers/       # Endpoint /token
└── README.md
```
