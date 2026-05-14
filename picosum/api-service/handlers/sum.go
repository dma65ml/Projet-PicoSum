// Package handlers regroupe les handlers HTTP de l'api-service.
//
// Un handler Fiber a la signature func(*fiber.Ctx) error.
// Il lit la requête, exécute la logique métier et écrit la réponse.
// La gestion des erreurs remonte via le ErrorHandler central (défini dans main.go).
package handlers

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/picosum/api-service/internal/calculator"
)

// SumHandler godoc
// Les annotations ci-dessous sont lues par swaggo pour générer la doc OpenAPI.
// Chaque annotation correspond à un champ de la spécification OpenAPI 3.0.
//
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
	// c.Query() lit un paramètre d'URL (?a=5). Retourne "" si absent.
	a, err := parseAndValidate(c.Query("a"))
	if err != nil {
		// fiber.Map est un alias de map[string]any, pratique pour le JSON.
		// c.Status(...).JSON(...) fixe le code HTTP puis sérialise en JSON.
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

	// log/slog (Go 1.21) produit du JSON structuré. Les champs clé/valeur
	// permettent de filtrer et d'agréger les logs dans des outils comme Loki ou Datadog.
	// Le request_id (posé par RequestIDMiddleware) corrèle ce log avec ceux
	// de web-app et oauth-server pour la même requête utilisateur.
	slog.Info("calcul effectué",
		"request_id", c.Locals("requestID"),
		"a", a, "b", b, "sum", result,
	)

	// c.JSON sérialise la valeur en JSON et pose Content-Type: application/json.
	return c.JSON(fiber.Map{"sum": result})
}

// parseAndValidate convertit une chaîne en entier et vérifie la plage [0, 10].
//
// Cette fonction encapsule la validation des entrées utilisateur : toujours valider
// à la frontière du système (ici le paramètre HTTP) avant d'appeler la logique métier.
// fiber.ErrBadRequest est une *fiber.Error avec Code=400 ; le ErrorHandler central
// le capte et retourne une réponse cohérente sans exposer de détails internes.
func parseAndValidate(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 10 {
		return 0, fiber.ErrBadRequest
	}
	return n, nil
}
