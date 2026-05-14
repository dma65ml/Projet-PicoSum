package handlers

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/picosum/api-service/internal/calculator"
)

// SumHandler godoc
// @Summary Calcule la somme de deux entiers
// @Description Calcule A + B. Les deux valeurs doivent être des entiers entre 0 et 10.
// @Tags calcul
// @Accept json
// @Produce json
// @Param a query int true "Premier entier (0-10)"
// @Param b query int true "Second entier (0-10)"
// @Success 200 {object} map[string]int "Résultat de la somme"
// @Failure 400 {object} map[string]string "Paramètre invalide"
// @Failure 401 {object} map[string]string "Token invalide"
// @Router /sum [get]
// @Security BearerAuth
func SumHandler(c *fiber.Ctx) error {
	a, err := parseAndValidate(c.Query("a"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "paramètre 'a' invalide : entier entre 0 et 10 requis",
		})
	}

	b, err := parseAndValidate(c.Query("b"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "paramètre 'b' invalide : entier entre 0 et 10 requis",
		})
	}

	result := calculator.Add(a, b)
	slog.Info("calcul effectué",
		"request_id", c.Locals("requestID"),
		"a", a, "b", b, "sum", result,
	)
	return c.JSON(fiber.Map{"sum": result})
}

// parseAndValidate convertit une chaîne en entier et vérifie qu'il est dans [0, 10].
func parseAndValidate(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 10 {
		return 0, fiber.ErrBadRequest
	}
	return n, nil
}
