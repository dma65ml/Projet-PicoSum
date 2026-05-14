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
// Cette injection permet le test unitaire avec un mock.
func HandleSumWith(caller client.SumCaller) fiber.Handler {
	return func(c *fiber.Ctx) error {
		aStr := c.FormValue("a")
		bStr := c.FormValue("b")

		a, err := parseAndValidate(aStr)
		if err != nil {
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

		reqID, _ := c.Locals("requestID").(string)
		// N7 : propager le contexte de la requête pour annuler l'appel sortant
		// si le navigateur se déconnecte avant la réponse.
		sum, err := caller.CallSum(c.UserContext(), a, b, reqID)
		if err != nil {
			slog.Error("erreur appel api-service", "request_id", reqID, "error", err)
			// html.EscapeString évite l'injection XSS via un message d'erreur malveillant (C3)
			return c.Status(fiber.StatusBadGateway).SendString(
				fmt.Sprintf(`<span class="error-response">⚠ Erreur service : %s</span>`,
					html.EscapeString(err.Error())),
			)
		}

		slog.Info("somme transmise", "request_id", reqID, "a", a, "b", b, "sum", sum)
		return c.SendString(fmt.Sprintf(
			`<strong>✓ %d + %d = %d</strong>`, a, b, sum,
		))
	}
}

// parseAndValidate convertit une chaîne en entier et vérifie qu'il est dans [0, 10].
func parseAndValidate(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 10 {
		return 0, fmt.Errorf("valeur %q invalide (doit être 0-10)", s)
	}
	return n, nil
}
