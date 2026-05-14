// Package calculator contient la logique métier pure de PicoSum.
//
// Principe de responsabilité unique (SRP) : ce package ne sait rien de HTTP,
// de Fiber ou des logs. Il ne fait que calculer. Cette séparation rend la
// fonction testable indépendamment du transport HTTP et facilite les
// refactorisations futures (ex. passer à gRPC ne touche pas ce package).
package calculator

// Add retourne la somme de deux entiers.
//
// Fonction pure : même entrée → même sortie, aucun effet de bord.
// Les fonctions pures sont les plus simples à tester (pas de mock nécessaire)
// et à raisonner sur leur comportement.
func Add(x, y int) int {
	return x + y
}
