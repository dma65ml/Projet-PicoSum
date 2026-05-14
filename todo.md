# TODO – Plan d’implémentation PicoSum (1 jour)

## Setup initial (30 min)
- [ ] Créer la structure de dossiers (web-app, api-service, oauth-server)
- [ ] Écrire les fichiers `go.mod` pour les trois services
- [ ] Créer le `docker-compose.yml` de base (services sans build détaillé)
- [ ] Initialiser un dépôt Git (`.gitignore` pour Go et binaires)

## Module 1 – Service API (2h)
- [ ] Implémenter `calculator/sum.go` et son test
- [ ] Écrire `middleware/request_id.go`
- [ ] Écrire `middleware/auth.go` (token fixe) et son test
- [ ] Écrire `handlers/sum.go` avec annotations Swagger
- [ ] Écrire `main.go` assemblant routes, middlewares, Swagger
- [ ] Générer la doc Swagger (`swag init`)
- [ ] Écrire les tests unitaires (`sum_test.go`, `auth_test.go`)
- [ ] Créer le `Dockerfile` pour api-service
- [ ] Tester localement : `go run .` et vérifier `/sum? a=5&b=3`

## Module 2 – Web application (2h)
- [ ] Créer `static/index.html` (avec Alpine, HTMX, Pico.css)
- [ ] Télécharger et placer `pico.min.css` (et éventuellement `alpine.min.js`) dans `static/`
- [ ] Implémenter `middleware/request_id.go` (identique à api)
- [ ] Implémenter `client/api_client.go` (appel à api-service avec token)
- [ ] Implémenter `handlers/calc.go` (reçoit formulaire, appelle client)
- [ ] Écrire `main.go` avec `//go:embed static` et routes
- [ ] Écrire tests `calc_test.go` et `api_client_test.go`
- [ ] Créer `Dockerfile` pour web-app
- [ ] Tester localement avec `API_URL=http://localhost:8081`

## Module 3 – Serveur OAuth simpliste (30 min)
- [ ] Écrire `oauth-server/main.go` (endpoint /token retour token fixe)
- [ ] Créer `Dockerfile` pour oauth-server
- [ ] Tester avec `curl -X POST http://localhost:8082/token`

## Module 4 – Intégration Docker Compose (1h)
- [ ] Finaliser `docker-compose.yml` avec les trois services et variables d’env
- [ ] Vérifier que `docker-compose up --build` fonctionne
- [ ] Tester le flux complet : http://localhost:8080
- [ ] Vérifier les logs : `docker-compose logs`
- [ ] Accéder à Swagger : http://localhost:8081/swagger/index.html

## Tests et validation (1h)
- [ ] Exécuter tous les tests unitaires (`go test ./...` dans chaque service)
- [ ] (Optionnel) Écrire un test d’intégration simple
- [ ] Valider les critères de succès :
  - [ ] Les trois containers démarrent sans erreur
  - [ ] La page web affiche le résultat de la somme (ex: 5+3=8)
  - [ ] Les logs montrent X-Request-ID corrélé
  - [ ] Swagger est accessible
  - [ ] Au moins un test unitaire passe

## Documentation et livrables (30 min)
- [ ] Rédiger `README.md` (build, run, tests, structure)
- [ ] Copier `spec.md`, `architecture.md`, `implementation-plan.csv`, `prompt-plan.md`, `todo.md` dans le livrable
- [ ] Créer une archive ZIP ou tag Git

## Post-livraison (si temps restant)
- [ ] Améliorer les logs (ajouter corps des requêtes en DEBUG)
- [ ] Ajouter un endpoint `/health` sur chaque service
- [ ] Ajouter un mode `error` simulé pour tester les pannes
