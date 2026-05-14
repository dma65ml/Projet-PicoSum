# Audit de sécurité PicoSum — Journal des corrections

Toutes les corrections de sécurité appliquées depuis le début du projet POC PicoSum.  
Services concernés : **web-app** (8080) · **api-service** (8081) · **oauth-server** (8082)

---

## Audit 1 — Corrections N1 à N6

### N1 · ErrorHandler sécurisé — oauth-server
**Fichier :** `oauth-server/main.go`  
**Risque :** Le handler d'erreur par défaut de Fiber expose la stack trace et les détails internes dans la réponse HTTP.  
**Correction :** Handler personnalisé qui retourne uniquement le code HTTP standard (`http.StatusText`) sans aucune information interne.

### N2 · Oracle de timing sur la comparaison de token — api-service
**Fichier :** `api-service/middleware/auth.go`  
**Risque :** `subtle.ConstantTimeCompare` retourne immédiatement pour des slices de longueurs différentes → oracle de timing exploitable pour deviner la longueur du token puis son contenu.  
**Correction :** Normalisation préalable des deux tokens par HMAC-SHA256 (clé interne fixe `picosum-token-comparator-v1`). Les deux arguments passés à `ConstantTimeCompare` ont toujours 32 octets, éliminant l'oracle.

### N3 · Injection XSS via message d'erreur — web-app
**Fichier :** `web-app/handlers/calc.go`  
**Risque :** Le message d'erreur retourné par l'api-service est injecté directement dans le HTML de la réponse, permettant une injection de balises si le message contient `<`, `>` ou `"`.  
**Correction :** Application systématique de `html.EscapeString()` sur tout message d'erreur externe avant insertion dans une réponse HTML.

### N4 · CSP stricte sans `unsafe-*` — web-app
**Fichiers :** `web-app/main.go`, `web-app/static/index.html`, `web-app/static/app.css`, `web-app/static/app.js`  
**Risque :** Alpine.js v3 utilise `new Function()` internement → nécessite `unsafe-eval` dans la CSP, rendant la politique inefficace contre les XSS.  
**Correction :**
- Suppression d'Alpine.js, remplacement par ~25 lignes de vanilla JS dans `app.js`.
- Extraction de tous les `<style>` inline et attributs `style=` vers `app.css`.
- CSP finale : `default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; font-src 'self'; form-action 'self'` — aucun `unsafe-*`.

### N5 · IP réelle derrière un reverse proxy — web-app + api-service
**Fichiers :** `web-app/main.go`, `api-service/main.go`  
**Risque :** Sans configuration, Fiber lit l'IP depuis l'en-tête `X-Forwarded-For` de manière non contrôlée, permettant le spoofing IP pour contourner le rate limiting.  
**Correction :** `ProxyHeader: os.Getenv("TRUSTED_PROXY_HEADER")` — l'en-tête de confiance est configurable via variable d'environnement. Vide par défaut (exposition directe) ; à valoriser uniquement derrière un proxy de confiance.

### N6 · HSTS — web-app + api-service
**Fichiers :** `web-app/main.go`, `api-service/main.go`  
**Risque :** Sans `Strict-Transport-Security`, un attaquant peut forcer un downgrade HTTP sur les navigateurs qui ne se souviennent pas d'avoir vu le site en HTTPS.  
**Correction :** En-tête `Strict-Transport-Security: max-age=31536000; includeSubDomains` ajouté via `securityHeaders()`.

---

## Audit 2 — Corrections N7 à N9

### N7 · Propagation du contexte de requête — web-app
**Fichier :** `web-app/handlers/calc.go`  
**Risque :** L'appel HTTP sortant vers l'api-service ignorait le contexte de la requête entrante. Si le navigateur se déconnecte (timeout, navigation), la goroutine et la connexion HTTP sortante continuaient à s'exécuter inutilement.  
**Correction :** Passage de `c.UserContext()` à `caller.CallSum()`, qui le transmet à `http.NewRequestWithContext()`. L'appel sortant est annulé si le contexte parent est annulé.

### N8 · Validation du `grant_type` OAuth2 — oauth-server
**Fichier :** `oauth-server/handlers/token.go`  
**Risque :** Le handler `/token` acceptait n'importe quel `grant_type`, y compris des valeurs non définies par la RFC 6749.  
**Correction :** Vérification explicite `if grantType != "client_credentials"` avec retour `400 unsupported_grant_type`.

### N9 · Rate limiting global — web-app
**Fichier :** `web-app/main.go`  
**Risque :** Absence de protection contre le déni de service par saturation de requêtes.  
**Correction :** Middleware `limiter.New` (60 req/min/IP) appliqué globalement avant toutes les routes, couvrant `/`, `/static` et `/sum`.  
_Note : stockage en mémoire, compteurs réinitialisés au redémarrage du processus (acceptable pour un POC mono-instance)._

---

## Audit 3 — Corrections A1 à A5

### A1 · Attributs `style=` inline bloqués par la CSP + HTMX n'affiche pas les erreurs HTTP — web-app
**Fichiers :** `web-app/handlers/calc.go`, `web-app/static/app.css`, `web-app/static/app.js`  
**Risque (1) :** Les réponses d'erreur du handler Go contenaient `style="color:var(--pico-del-color)"` — attributs bloqués silencieusement par la CSP `style-src 'self'`, rendant les messages d'erreur invisibles.  
**Risque (2) :** HTMX v1.9 n'effectue pas de swap pour les réponses 4xx/5xx par défaut → les erreurs serveur n'apparaissent jamais dans `#result`.  
**Correction :**
- Remplacement de tous les `style="…"` par `class="error-response"` dans `calc.go`.
- Classe `.error-response { color: var(--pico-del-color, red); }` ajoutée dans `app.css`.
- Configuration dans `app.js` : `htmx.config.responseHandling = [{ code: '.*', swap: true }]` — active le swap pour tous les codes de statut HTTP.

### A2 · Token de repli silencieux — web-app + oauth-server
**Fichiers :** `web-app/internal/client/api_client.go`, `oauth-server/handlers/token.go`  
**Risque :** En l'absence de `API_TOKEN`, les deux composants utilisaient silencieusement `poc-token-123` sans aucune alerte dans les logs. Un oubli de configuration en production passerait inaperçu.  
**Correction :** `slog.Warn("[SECURITE] API_TOKEN non défini — token de développement utilisé. NE PAS déployer en production.")` ajouté avant l'utilisation du token de repli dans les deux fichiers.

### A3 · Absence de BodyLimit et de rate limiting — oauth-server
**Fichier :** `oauth-server/main.go`  
**Risque :** Sans `BodyLimit`, un attaquant peut envoyer des corps de requête arbitrairement grands pour épuiser la mémoire. Sans rate limiting, l'endpoint `/token` est vulnérable au brute-force et au DoS.  
**Correction :**
- `BodyLimit: 4 * 1024` ajouté dans `fiber.Config`.
- `ProxyHeader: os.Getenv("TRUSTED_PROXY_HEADER")` ajouté (cohérence avec les autres services).
- Middleware `limiter.New` (30 req/min/IP) ajouté avant les routes.  
_Note : stockage en mémoire, compteurs réinitialisés au redémarrage (acceptable pour un POC)._

### A4 · HSTS sans directive `preload` — web-app + api-service
**Fichiers :** `web-app/main.go`, `api-service/main.go`  
**Risque :** Sans `preload`, les navigateurs ne peuvent pas inscrire le domaine dans la liste HSTS préconstruite, laissant la première visite HTTP non protégée.  
**Correction :** En-tête mis à jour : `Strict-Transport-Security: max-age=31536000; includeSubDomains; preload`.

### A5 · Documentation de la limitation du rate limiter en mémoire
**Fichiers :** `web-app/main.go`, `oauth-server/main.go`  
**Risque :** Comportement non documenté pouvant induire en erreur lors d'un passage en production (compteurs perdus au redémarrage, pas de synchronisation entre instances).  
**Correction :** Commentaire explicite ajouté dans les deux fichiers mentionnant que le stockage est en mémoire et les conséquences opérationnelles.

---

## Hors périmètre POC

| Ref | Sujet | Justification |
|-----|-------|---------------|
| A6 | Protection CSRF | Pas de sessions utilisateur, pas de mutations d'état sensibles → hors périmètre pour un POC sans authentification utilisateur. |

---

## Récapitulatif par service

| Service | Corrections appliquées |
|---------|----------------------|
| **api-service** | N2 (timing oracle), N5 (ProxyHeader), N6 (HSTS), N6+A4 (HSTS preload) |
| **web-app** | N3 (XSS), N4 (CSP stricte, Alpine→vanilla JS), N5 (ProxyHeader), N6 (HSTS), N7 (contexte requête), N9 (rate limiting), A1 (inline style→classe CSS, HTMX config), A2 (warn token repli), A4 (HSTS preload), A5 (doc rate limiter) |
| **oauth-server** | N1 (ErrorHandler), N8 (grant_type), A2 (warn token repli), A3 (BodyLimit + rate limiting) |
