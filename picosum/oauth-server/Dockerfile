# Dockerfile de l'oauth-server — Build multi-stage
#
# Un build multi-stage sépare la phase de compilation de l'image finale.
# L'étape "builder" contient le compilateur Go et tous les outils nécessaires
# à la compilation. L'image finale "scratch" ne contient que le binaire produit.
# Résultat : image finale ~10 Mo au lieu de ~800 Mo, surface d'attaque minimale.

# ── Étape 1 : compilation ─────────────────────────────────────────────────────
#
# golang:1.21-alpine : image officielle Go basée sur Alpine Linux.
# Alpine est ~5 Mo (vs ~120 Mo pour Debian) grâce à musl libc et BusyBox.
# La variante alpine est préférée pour les images builder car elle télécharge
# plus vite et occupe moins d'espace dans le cache Docker local.
FROM golang:1.21-alpine AS builder

# ca-certificates : bundle des certificats racines CA publics (Mozilla).
# Requis ici pour go mod download qui télécharge les modules via HTTPS.
# Ces certificats sont également copiés dans l'image finale pour permettre
# d'éventuels appels HTTPS sortants futurs (ex. webhook de révocation de token).
RUN apk --no-cache add ca-certificates

# WORKDIR définit /app comme répertoire courant pour les instructions suivantes.
# Préférer WORKDIR à RUN mkdir + RUN cd : WORKDIR persiste entre les instructions.
WORKDIR /app

# Stratégie de cache Docker en deux temps :
#
# 1) Copier go.mod et go.sum (descripteurs de dépendances) et télécharger les modules.
#    Cette couche est mise en cache par Docker tant que go.mod/go.sum ne changent pas.
#    Si seul le code Go change, docker build réutilise cette couche sans re-télécharger.
#
# 2) Copier le code source ensuite (COPY . .).
#    Cette couche change à chaque modification de code, mais ne re-déclenche pas
#    le téléchargement des dépendances grâce à la séparation ci-dessus.
#
# Sans cette séparation (COPY . . en une fois), toute modification de code
# invaliderait le cache des dépendances → re-téléchargement à chaque build.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compilation du binaire oauth-server en mode statique pour Linux.
#
# CGO_ENABLED=0 : désactive l'interface Go/C (CGO).
#   Sans CGO, Go lie statiquement toutes les dépendances dans le binaire.
#   Le binaire résultant ne dépend d'aucune bibliothèque dynamique (.so) du système.
#   Ceci est obligatoire pour fonctionner dans scratch (aucune libc disponible).
#
# GOOS=linux : indique au compilateur Go de cibler le système d'exploitation Linux.
#   Permet de compiler depuis n'importe quel OS hôte (macOS, Windows, Linux).
#   Go supporte nativement la compilation croisée sans outils supplémentaires.
#
# -ldflags="-s -w" : drapeaux du linker pour réduire la taille du binaire.
#   -s : omet la table des symboles de débogage (symbol table).
#   -w : omet les informations DWARF (utilisées par les débogueurs comme dlv).
#   Impact : le binaire est 20-30 % plus petit. Les stack traces restent lisibles
#   en production car Go les génère à partir des informations dans le binaire,
#   pas uniquement depuis les symboles de débogage.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o oauth-server .

# ── Étape 2 : image finale ────────────────────────────────────────────────────
#
# "scratch" est l'image de base vide de Docker Registry.
# Elle ne contient : ni shell, ni libc, ni système de fichiers, ni utilisateur root.
# C'est l'image finale la plus petite et la plus sécurisée possible.
#
# Alternative pour faciliter le débogage en production :
#   FROM gcr.io/distroless/static-debian12
# distroless (Google) fournit les certificats CA et quelques fichiers système
# sans shell interactif — bon compromis entre sécurité et débogage.
FROM scratch

# Copie des certificats CA depuis le builder.
# scratch ne contient aucun certificat — sans eux, toute connexion TLS sortante
# échouerait avec "x509: certificate signed by unknown authority".
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copie du seul binaire nécessaire à l'exécution.
# Grâce à CGO_ENABLED=0, ce binaire est autosuffisant et ne nécessite
# aucune bibliothèque dynamique du système hôte.
COPY --from=builder /app/oauth-server /oauth-server

# EXPOSE est une métadonnée informative — elle n'ouvre pas le port réseau.
# Elle indique aux opérateurs et à docker-compose quel port écoute l'application.
# L'exposition réelle est contrôlée par docker-compose.yml (ports: 127.0.0.1:8082:8082).
EXPOSE 8082

# ENTRYPOINT en forme de tableau JSON (exec form) :
#   - Le binaire est lancé directement, sans passer par /bin/sh -c "...".
#   - Avantages : pas de shell requis (scratch n'en a pas), les signaux Unix
#     (SIGTERM pour arrêt propre, SIGKILL pour arrêt forcé) sont transmis
#     directement au processus Go (PID 1) sans passer par un shell intermédiaire.
ENTRYPOINT ["/oauth-server"]
