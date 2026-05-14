// Package handlers contient les handlers HTTP de la web-app.
//
// Un handler reçoit une requête HTTP, orchestre les appels nécessaires
// (ici vers api-service) et construit la réponse. Pour les interfaces HTMX,
// la réponse est un fragment HTML injecté dans la page — pas un JSON complet.
package handlers

import (
	"fmt"
	"html"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/picosum/web-app/internal/client"
)

// HandleSumWith retourne un handler Fiber qui délègue le calcul au caller fourni.
//
// Pattern "dependency injection via paramètre de fonction" :
// Au lieu d'appeler directement le client HTTP, on injecte une interface SumCaller.
// Avantages :
//   - Tests unitaires : on passe un mock qui ne fait pas de vrai appel réseau.
//   - Flexibilité : on peut remplacer l'implémentation HTTP par gRPC sans
//     modifier ce handler.
//
// C'est l'alternative Go au polymorphisme par héritage des langages OO :
// une petite interface (une seule méthode) suffit à rendre le code testable.
func HandleSumWith(caller client.SumCaller) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// c.FormValue lit un champ de formulaire HTML (Content-Type: application/x-www-form-urlencoded).
		// HTMX envoie les données du <form> dans ce format par défaut avec hx-post.
		aStr := c.FormValue("a")
		bStr := c.FormValue("b")

		a, err := parseAndValidate(aStr)
		if err != nil {
			// On retourne un fragment HTML avec la classe CSS "error-response"
			// définie dans app.css. Les attributs style= inline sont interdits
			// par la Content Security Policy (CSP) qui n'autorise que style-src 'self'.
			return c.Status(fiber.StatusBadRequest).SendString(
				`<span class="error-response">⚠ Valeur A invalide (entier 0-10 requis)</span>`,
			)
		}
		b, err := parseAndValidate(bStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(
				`<span class="error-response">⚠ Valeur B invalide (entier 0-10 requis)</span>`,
			)
		}

		// c.Locals récupère la valeur stockée par RequestIDMiddleware.
		// L'assertion de type ".(string)" est sûre : si la clé est absente, on obtient "".
		reqID, _ := c.Locals("requestID").(string)

		// c.UserContext() retourne le context.Context de la requête Fiber.
		// Le passer à CallSum permet d'annuler l'appel HTTP sortant si le client
		// se déconnecte avant la réponse (ex. fermeture de l'onglet, timeout navigateur).
		// Sans cela, la goroutine et la connexion TCP vers api-service continueraient
		// inutilement, gaspillant des ressources.
		sum, err := caller.CallSum(c.UserContext(), a, b, reqID)
		if err != nil {
			slog.Error("erreur appel api-service", "request_id", reqID, "error", err)

			// html.EscapeString est indispensable ici : le message d'erreur vient
			// d'un service externe et pourrait contenir des balises HTML malveillantes
			// (Cross-Site Scripting, XSS). Sans échappement, un service compromis
			// pourrait injecter du JS dans la page utilisateur.
			return c.Status(fiber.StatusBadGateway).SendString(
				fmt.Sprintf(`<span class="error-response">⚠ Erreur service : %s</span>`,
					html.EscapeString(err.Error())),
			)
		}

		slog.Info("somme transmise", "request_id", reqID, "a", a, "b", b, "sum", sum)

		// HTMX remplace le contenu de l'élément cible (hx-target="#result")
		// avec ce fragment HTML. Pas de rechargement de page — seul #result change.
		return c.SendString(fmt.Sprintf(
			`<strong>✓ %d + %d = %d</strong>`, a, b, sum,
		))
	}
}

// parseAndValidate convertit une chaîne en entier et vérifie la plage [0, 10].
//
// Toujours valider les entrées à la frontière du système (ici le formulaire HTML)
// avant de les transmettre à la logique métier ou à un service aval.
// Cette validation est redondante avec celle de l'api-service (défense en profondeur) :
// même si la web-app est contournée, l'API rejette les valeurs hors bornes.
func parseAndValidate(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 10 {
		return 0, fmt.Errorf("valeur %q invalide (doit être 0-10)", s)
	}
	return n, nil
}
