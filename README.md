# Projet PicoSum

POC d'architecture microservices Go avec durcissement sécurité progressif.

## Objectif

Démontrer la conception et le sécurisation itérative d'une application distribuée Go minimaliste :
une calculatrice (A + B) décomposée en trois services indépendants.

## Architecture

```
[Navigateur] → [Caddy:80] ──┬──→ [web-app:8080]
                             └──→ [api-service:8081]  (/swagger/*)
                                         ↑
                                  [oauth-server:8082]
```

| Service        | Port (interne) | Rôle                                          |
|----------------|----------------|-----------------------------------------------|
| `caddy`        | 80 (public)    | Reverse proxy, routage, en-têtes de sécurité  |
| `web-app`      | 8080           | Interface HTMX + Pico.css (interne)           |
| `api-service`  | 8081           | API REST Go + Swagger + AuthMiddleware         |
| `oauth-server` | 8082           | Serveur OAuth2 simplifié (token fixe POC)     |

## Démarrage rapide

```bash
cd picosum
docker-compose up --build
```

Interface : http://localhost  
Swagger API : http://localhost/swagger/index.html

## Sécurité

14 corrections appliquées en 3 audits successifs — voir [`sec-audit.md`](sec-audit.md) pour le détail complet.

Points clés :
- CSP stricte sans `unsafe-*` (Alpine.js remplacé par vanilla JS)
- Comparaison de token HMAC-SHA256 (anti oracle de timing)
- Rate limiting par IP sur les trois services
- HSTS avec `preload`
- Alertes de démarrage si token par défaut utilisé

## Structure du dépôt

```
├── picosum/              # Code source des services
│   ├── api-service/      # API REST Go + Swagger
│   ├── web-app/          # Interface HTMX + Pico.css
│   ├── oauth-server/     # Serveur OAuth2 simplifié
│   ├── Caddyfile         # Configuration du reverse proxy
│   └── docker-compose.yml
├── sec-audit.md          # Journal des corrections de sécurité
├── architecture.md       # Décisions d'architecture
├── spec.md               # Spécifications fonctionnelles
└── prompt-plan.md        # Plan d'implémentation
```

## Tests

```bash
cd picosum/api-service && go test ./...   # 26 tests
cd picosum/web-app && go test ./...       # 9 tests
```
