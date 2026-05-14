# Spécification fonctionnelle et technique – PicoSum

## Objectif
Réaliser PicoSum, une Preuve de Concept (POC) en un jour, démontrant une architecture de microservices avec :
- Une application web Go (Fiber) servant une interface utilisateur (HTML/HTMX/Alpine.js/Pico.css)
- Un service API Go (Fiber) exposant un calcul de somme, documenté via Swagger
- Un service d’authentification OAuth2 simpliste (token fixe ou validation basique)
- Conteneurisation avec Docker Compose (local)
- Logs permettant le diagnostic
- Tests unitaires minimaux

## Fonctionnalités

### Principales (obligatoires pour la POC)
1. **Page web** avec deux champs numériques A et B (valeurs de 0 à 10)
2. **Validation locale** avec Alpine.js (désactivation du bouton, message d’erreur)
3. **Envoi via HTMX** au serveur web (container `web-app`)
4. **Appel du service API** (container `api-service`) par le web-app, avec authentification OAuth2 simpliste
5. **Calcul de la somme** par l’API, retour JSON `{ "sum": N }` ou erreur HTTP 400
6. **Affichage du résultat** (somme ou message d’erreur) dans la page web
7. **Swagger UI** accessible sur `/swagger/index.html` du service API
8. **Logs** structurés (niveaux INFO/ERROR) avec corrélation des requêtes (X-Request-ID)
9. **Tests unitaires** : au moins validation des bornes 0-10 dans le service API

### Secondaires (optionnelles mais recommandées)
- Log des corps de requêtes/réponses en DEBUG (réglable par variable d’environnement)
- Page d’accueil statique avec instructions
- Mode “simulation d’erreur” (paramètre `?error=true` pour tester les pannes)

## Utilisateurs cibles
- Développeurs internes testant l’intégration des composants (besoin de logs clairs et de Swagger)

## Contraintes techniques et de ressources
- Délai : 1 jour (développement, tests, documentation)
- Plateforme : Docker Compose local (pas de Kubernetes ni Cloud)
- Langage : Go 1.21+
- Frameworks : Fiber v2, Swaggo, go-oauth2/oauth2 (simpliste)
- Front : HTMX, Alpine.js, Pico.css (embarqués avec `//go:embed`)
- Pas de base de données (aucune persistance)
- Tests : `testify` + `app.Test()` de Fiber

## Architecture générale souhaitée
[Utilisateur] → [web-app:8080] → [api-service:8081] → [OAuth2 simpliste]  
↓ ↓  
logs logs

- Communication interne via réseau Docker (noms de services : `web`, `api`, `oauth`)
- Authentification : token fixe ou JWT simple partagé (pas de flow complexe)
- Swagger exposé par l’API

## Défis potentiels et solutions
| Défi | Solution |
|------|----------|
| Génération et intégration Swagger dans le temps | Utiliser `swag init` avec annotations minimales ; accepter une doc simple mais fonctionnelle |
| Tests unitaires chronophages | Se limiter à tester le handler de somme (bornes) ; mocker l’appel OAuth avec un middleware de test |
| Coordination Docker Compose | Fournir un `docker-compose.yml` prêt à l’emploi avec dépendances explicites (`depends_on`) |
| Logs non corrélés | Injecter un middleware Fiber qui génère/transmet un `X-Request-ID` ; propager via `http.Client` |
| Simplicité OAuth2 | Implémenter un serveur OAuth minimal (endpoint `/token`) avec stockage mémoire, et un middleware API validant un token statique pour la POC |

## Critères de succès (par ordre de priorité)
1. `docker-compose up` lance les trois containers sans erreur.
2. La page web permet la saisie, l’envoi via HTMX, et affiche la somme ou une erreur (flux complet validé).
3. Les logs montrent la chaîne des appels (web → API → OAuth) avec identifiants de requête.
4. La documentation Swagger de l’API est accessible sur `/swagger`.
5. Au moins un test unitaire (validation 0-10) passe avec succès.

## Livrables attendus
- Code source des trois services Go (dossier `web-app`, `api-service`, `oauth-server`)
- Fichiers d’embedding (HTML, CSS, JS)
- `go.mod` / `go.sum`
- `docker-compose.yml` et `Dockerfile` pour chaque service
- `spec.md`, décisions techniques, risques (présent document)

## 2. Décisions techniques clés

|Domaine|Décision|Justification|
|---|---|---|
|Framework web|Fiber v2|Performant, API Express-like, intégration facile avec `//go:embed` et middleware.|
|Templating|Aucun moteur côté serveur ; HTML pur + Alpine.js/HTMX|Réduction de la complexité ; état géré par Alpine ; communication via JSON.|
|CSS|Pico.css|Compact, moderne, sans classes requises, facile à embarquer.|
|JS|Alpine.js + HTMX|Complémentarité : Alpine pour l’état local, HTMX pour les requêtes réseau.|
|Assets statiques|`//go:embed`|Pas de système de fichiers externe ; binaire unique par service.|
|Documentation API|Swaggo (swag init)|Standard Go pour OpenAPI ; génération automatique.|
|OAuth2 (POC)|Token fixe partagé (ex: `Bearer toto123`) + middleware de validation simple|Réduction du temps de développement ; permet de démontrer le flux d’authentification sans serveur OAuth complet. Alternative : mini serveur OAuth avec `go-oauth2/oauth2` en mémoire (si temps restant).|
|Logs|Log standard (`log/slog` de Go 1.21) avec format JSON ; middleware pour X-Request-ID|Suffisant pour diagnostic ; `slog` est natif et performant.|
|Communication inter-services|HTTP avec `http.Client` (pas de gRPC)|Simplicité, alignement avec le reste du stack.|
|Tests|`testify/assert` + `app.Test()` de Fiber|Légèreté, intégré, sans dépendance lourde.|
|Conteneurisation|Docker Compose, réseau bridge personnalisé|Standard pour les POC multi-containers locales.|

---

## 3. Risques identifiés et stratégies de mitigation

|Risque|Probabilité|Impact|Mitigation|
|---|---|---|---|
|**Swagger non généré / mal configuré**|Moyenne|Élevé (critère de succès)|Préparer un fichier `docs/docs.go` minimal à la main en dernier recours ; utiliser `swag fmt` et vérifier les annotations.|
|**Tests unitaires non finalisés faute de temps**|Élevée|Moyen (un seul test obligatoire)|Écrire d’abord le test de validation (le plus simple) avant même le handler ; utiliser TDD simplifié.|
|**Problème de réseau Docker entre containers**|Faible|Élevé|Utiliser `depends_on` et un réseau dédié ; tester localement avec `docker-compose up` régulièrement ; prévoir un script de vérification (`curl`).|
|**Logs non corrélés rendant le debug difficile**|Moyenne|Moyen|Implémenter le middleware `X-Request-ID` dès le début de la journée ; le propager systématiquement.|
|**OAuth2 simpliste trop trivial (accepté)**|Accepté|Faible (critère succès l’accepte)|Documenter que c’est une POC ; simuler la validation avec une en-tête fixe, mais expliquer comment passer à un vrai OAuth2.|
|**Manque de temps pour l’embarquement des assets**|Faible|Moyen|Tester avec CDN d’abord, puis passer à `//go:embed` ; les CDN sont une solution de repli acceptable pour la POC.|
|**Fiber incompatible avec `//go:embed` sur les templates**|Très faible|Faible|Utiliser `app.Static("/static", http.FS(staticFS))` – bien documenté.|

