# Dockerfile de l'api-service — Build multi-stage
#
# Un build multi-stage utilise plusieurs instructions FROM dans le même Dockerfile.
# Chaque FROM démarre une nouvelle "étape" avec sa propre image de base.
# Seuls les fichiers explicitement copiés avec COPY --from=<étape> passent à l'étape suivante.
# Résultat : l'image finale ne contient que le strict nécessaire pour exécuter l'application,
# sans les outils de compilation (compilateur Go, bibliothèques système, cache de modules…).
#
# Comparaison de taille typique :
#   Image Go classique (non multi-stage) : ~800 Mo
#   Image finale avec scratch            : ~10 Mo
# Moins de surface = moins de vulnérabilités potentielles à patcher.

# ── Étape 1 : compilation ─────────────────────────────────────────────────────
#
# golang:1.21-alpine est l'image officielle Go basée sur Alpine Linux.
# Alpine est une distribution Linux minimaliste (~5 Mo) conçue pour les conteneurs.
# Elle utilise musl libc au lieu de glibc, ce qui permet de compiler des binaires
# statiques plus facilement (pas de dépendances dynamiques à la libc système).
# L'image golang:1.21-alpine inclut le compilateur Go, git et les outils de base.
FROM golang:1.21-alpine AS builder

# apk est le gestionnaire de paquets Alpine (équivalent d'apt sur Debian/Ubuntu).
# --no-cache : ne crée pas de cache local des paquets → image builder plus légère.
#
# ca-certificates contient les certificats racines des autorités de certification (CA)
# publiques (Let's Encrypt, DigiCert, GlobalSign…). Ils sont nécessaires pour que
# Go puisse valider les certificats TLS lors des appels HTTPS sortants (go mod download,
# appels vers des API externes). Sans eux, les connexions HTTPS échouent avec
# "certificate signed by unknown authority".
# Ces certificats sont ensuite copiés dans l'image finale scratch (voir ci-dessous).
RUN apk --no-cache add ca-certificates

# WORKDIR crée le répertoire /app s'il n'existe pas et définit le répertoire
# de travail pour toutes les instructions suivantes (RUN, COPY, CMD, ENTRYPOINT).
# Équivalent de : mkdir -p /app && cd /app
# Utiliser WORKDIR plutôt que RUN cd : WORKDIR persiste entre les instructions RUN.
WORKDIR /app

# Optimisation du cache Docker — copier go.mod et go.sum AVANT le code source.
#
# Docker construit les images en couches (layers). Chaque instruction crée une couche.
# Si une couche n'a pas changé depuis le dernier build, Docker la réutilise depuis
# son cache (pas de ré-exécution). Les couches sont invalidées dès qu'une couche
# précédente change.
#
# En copiant go.mod/go.sum en premier et en téléchargeant les dépendances séparément,
# on profite du cache : si seul le code source change (pas les dépendances), la couche
# "go mod download" est réutilisée → build beaucoup plus rapide en développement.
# Si on copiait tout en une fois (COPY . .), la moindre modification de code
# re-téléchargerait toutes les dépendances.
COPY go.mod go.sum ./
RUN go mod download

# Copie du reste des sources (après le téléchargement des dépendances).
# Le "." final désigne le répertoire courant dans l'image (WORKDIR = /app).
COPY . .

# Compilation du binaire Go en mode statique pour Linux.
#
# CGO_ENABLED=0 : désactive CGO (C Go), l'interface entre Go et les bibliothèques C.
#   - Avec CGO activé, le binaire dépend dynamiquement de la libc du système hôte.
#   - Avec CGO=0, Go produit un binaire entièrement statique (toutes les dépendances
#     sont incluses dans le binaire). Ce binaire peut s'exécuter sur n'importe quel
#     système Linux, y compris scratch (qui n'a aucune bibliothèque système).
#
# GOOS=linux : cible la compilation pour Linux, même si le build tourne sur macOS ou Windows.
#   C'est la cross-compilation : Go compile nativement pour une cible différente de l'hôte.
#   Combiné avec CGO_ENABLED=0, cela garantit un binaire Linux statique sur toute plateforme.
#
# -ldflags="-s -w" : options passées au linker Go pour réduire la taille du binaire.
#   -s : supprime la table des symboles (symbol table) → inutile à l'exécution.
#   -w : supprime les informations de debug DWARF → inutile hors débogage.
#   Effet typique : réduit la taille du binaire de 20-30 %.
#   Inconvénient : les stack traces en cas de panic perdent certaines informations.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api-service .

# ── Étape 2 : image finale ────────────────────────────────────────────────────
#
# "scratch" est une image Docker vide — elle ne contient absolument rien :
# pas de shell (/bin/sh), pas d'utilitaires (ls, cat, curl…), pas de libc,
# pas d'utilisateur, pas de système de fichiers de base.
#
# C'est l'image la plus sécurisée possible : une attaque qui parviendrait à
# exécuter du code dans le conteneur ne trouverait aucun outil à exploiter.
# Contre-partie : impossible de faire docker exec ... /bin/sh pour déboguer.
# En production, utiliser distroless (Google) comme alternative avec quelques
# utilitaires de diagnostic sans shell interactif.
FROM scratch

# Copie des certificats CA depuis l'étape builder.
# Sans ces certificats, le binaire Go ne peut pas vérifier les certificats TLS
# des serveurs avec lesquels il communique (appels HTTPS sortants).
# L'image scratch n'a aucun certificat par défaut.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copie du binaire compilé depuis l'étape builder.
# C'est le seul fichier exécutable dans l'image finale.
# Grâce à CGO_ENABLED=0, il est autosuffisant et ne dépend d'aucune bibliothèque externe.
COPY --from=builder /app/api-service /api-service

# EXPOSE documente le port sur lequel le service écoute.
# EXPOSE n'ouvre PAS réellement le port — c'est de la documentation pour les opérateurs
# et pour docker-compose. L'ouverture effective se fait dans docker-compose.yml (ports:).
EXPOSE 8081

# ENTRYPOINT définit le processus principal du conteneur.
# Forme JSON (tableau) recommandée : évite l'interprétation par /bin/sh
# (qui n'existe pas dans scratch de toute façon).
# Le processus lancé aura le PID 1 — il doit donc gérer les signaux SIGTERM
# (arrêt propre) et SIGINT (Ctrl+C). Le serveur Fiber de Go le fait nativement.
ENTRYPOINT ["/api-service"]
