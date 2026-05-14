# Dockerfile de la web-app — Build multi-stage
#
# Un build multi-stage utilise plusieurs instructions FROM dans le même Dockerfile.
# Seuls les fichiers copiés explicitement avec COPY --from=<étape> passent à l'étape suivante.
# L'image finale ne contient que le binaire Go — sans compilateur, sans modules Go,
# sans système de fichiers Alpine. Taille typique : ~10 Mo au lieu de ~800 Mo.
#
# Particularité de la web-app : les fichiers statiques (HTML, CSS, JS, pico.css, htmx.js)
# sont embarqués dans le binaire via la directive //go:embed au moment de la compilation.
# Il n'y a donc aucun volume ni répertoire statique à gérer dans l'image finale.

# ── Étape 1 : compilation ─────────────────────────────────────────────────────
#
# golang:1.21-alpine : image officielle Go sur Alpine Linux (~5 Mo, sans glibc).
# Alpine utilise musl libc, compatible avec la compilation statique (CGO_ENABLED=0).
FROM golang:1.21-alpine AS builder

# ca-certificates : certificats racines des autorités de certification publiques.
# Nécessaires pour deux usages dans la web-app :
#   1. go mod download : téléchargement des dépendances via HTTPS (proxy.golang.org).
#   2. Appels HTTPS sortants vers l'api-service (si déployé avec TLS).
# Ces certificats sont copiés dans l'image finale scratch (qui n'en a aucun).
RUN apk --no-cache add ca-certificates

# WORKDIR : répertoire de travail pour toutes les instructions suivantes.
# Crée /app s'il n'existe pas. Équivalent de mkdir -p /app && cd /app.
WORKDIR /app

# Optimisation du cache Docker par séparation des dépendances et du code source.
#
# Docker met en cache chaque couche (instruction). Si go.mod et go.sum n'ont pas
# changé, la couche "go mod download" est réutilisée sans re-télécharger les modules.
# Seule une modification de go.mod/go.sum invalide cette couche.
# En développement, cela évite de re-télécharger les dépendances à chaque build
# quand seul le code source change.
COPY go.mod go.sum ./
RUN go mod download

# Copie des sources après le téléchargement des dépendances.
# Le COPY . . copie tout le contexte de build (dossier du service), y compris
# les fichiers statiques (static/) qui seront embarqués par //go:embed.
COPY . .

# Compilation en binaire Linux statique.
#
# CGO_ENABLED=0 : désactive l'interface C/Go → binaire 100 % statique, sans
#   dépendance à la libc du système. Indispensable pour l'image finale scratch.
# GOOS=linux : cross-compilation vers Linux (fonctionne depuis macOS ou Windows).
# -ldflags="-s -w" : supprime la table des symboles (-s) et les infos debug (-w)
#   pour réduire la taille du binaire de ~20-30 % sans affecter le comportement.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o web-app .

# ── Étape 2 : image finale ────────────────────────────────────────────────────
#
# scratch est une image Docker vide : pas de shell, pas d'utilitaires, pas de libc.
# Surface d'attaque minimale — un attaquant dans le conteneur ne trouve aucun outil.
# Le binaire Go statique est autosuffisant : il n'a besoin de rien d'autre pour s'exécuter.
FROM scratch

# Les certificats CA sont copiés depuis le builder car scratch n'en a aucun.
# Sans eux, Go ne peut pas valider les certificats TLS des services appelés
# (api-service, oauth-server) en cas de déploiement HTTPS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Seul le binaire est copié — les fichiers statiques sont déjà à l'intérieur,
# incorporés par //go:embed lors de la compilation à l'étape builder.
COPY --from=builder /app/web-app /web-app

# Documentation du port d'écoute. N'ouvre pas réellement le port —
# c'est docker-compose.yml qui gère l'exposition réseau (ports:).
EXPOSE 8080

# ENTRYPOINT en forme JSON : pas d'interprétation shell (et scratch n'a pas de shell).
# Le processus aura le PID 1 et recevra directement les signaux SIGTERM/SIGINT
# pour un arrêt propre du serveur Fiber (graceful shutdown).
ENTRYPOINT ["/web-app"]
