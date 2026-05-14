/**
 * app.js — Validation locale du formulaire PicoSum
 *
 * Pourquoi ce fichier existe-t-il ?
 * ----------------------------------
 * La Content Security Policy (CSP) du serveur interdit tout JavaScript inline :
 *   - Les attributs onclick="...", oninput="..." dans le HTML sont bloqués.
 *   - Les balises <script>...</script> directement dans le HTML sont bloquées.
 *   - "script-src 'self'" n'autorise que les fichiers .js servis par ce serveur.
 *
 * Pourquoi Alpine.js a-t-il été supprimé ?
 * -----------------------------------------
 * Alpine.js v3 utilise new Function() en interne pour évaluer les expressions
 * x-data="{ ... }". new Function() est équivalent à eval() et nécessite
 * la directive CSP "unsafe-eval" — ce qui annule toute protection contre le XSS.
 * Ce fichier remplace Alpine.js par ~25 lignes de vanilla JS, sans unsafe-eval.
 *
 * Rôle de ce fichier :
 *   1. Configurer HTMX pour afficher les réponses d'erreur HTTP (4xx/5xx).
 *   2. Valider les champs A et B en temps réel (sans aller-retour serveur).
 *   3. Activer le bouton "Calculer" uniquement quand les deux valeurs sont valides.
 */

/**
 * Configuration HTMX — activation du swap pour les réponses d'erreur.
 *
 * HTMX v1.9 ignore par défaut les réponses avec un code HTTP 4xx ou 5xx :
 * il ne met pas à jour le DOM, donc le message d'erreur du serveur n'apparaît jamais.
 * Ce comportement a été introduit pour éviter les effets de bord involontaires,
 * mais ici on veut afficher les erreurs dans #result.
 *
 * htmx.config.responseHandling est un tableau de règles { code, swap }.
 * L'expression régulière '.*' correspond à tous les codes HTTP.
 * swap: true → HTMX injecte toujours la réponse dans hx-target, quel que soit le code.
 */
htmx.config.responseHandling = [{ code: '.*', swap: true }];

/**
 * IIFE (Immediately Invoked Function Expression) — module auto-exécuté.
 *
 * Pourquoi envelopper le code dans (function() { ... }()) ?
 * Toutes les variables déclarées avec var à l'intérieur restent locales à cette
 * fonction et ne polluent pas l'espace de noms global (window). Sans cette
 * enveloppe, isValidBound et updateForm seraient accessibles globalement et
 * pourraient entrer en conflit avec d'autres scripts (ex. htmx, pico.css).
 *
 * 'use strict' active le mode strict d'ECMAScript :
 *   - Interdit les variables non déclarées (erreur au lieu de variable globale silencieuse).
 *   - Désactive certaines fonctionnalités dangereuses de JS (with, arguments.caller…).
 *   - Requis pour certaines optimisations des moteurs JS modernes.
 */
(function () {
    'use strict';

    /**
     * isValidBound — vérifie qu'une valeur est un entier dans [0, 10].
     *
     * @param {string} val - La valeur brute du champ input (toujours une chaîne).
     * @returns {boolean} true si la valeur est un entier valide entre 0 et 10.
     *
     * Pourquoi Number() plutôt que parseInt() ?
     * parseInt("5abc") → 5 (accepte le préfixe numérique — faux positif ici).
     * Number("5abc")   → NaN (strict — rejette tout ce qui n'est pas un nombre pur).
     *
     * Pourquoi Number.isInteger() et pas val % 1 === 0 ?
     * Number.isInteger(1.0) → true en JS (1.0 et 1 sont identiques).
     * Number.isInteger(1.5) → false. C'est exactement le comportement voulu.
     */
    function isValidBound(val) {
        var trimmed = val.trim(); // trim() ignore les espaces accidentels (ex. " 5 ")
        if (trimmed === '') return false; // champ vide → invalide, mais sans message d'erreur
        var n = Number(trimmed);
        return Number.isInteger(n) && n >= 0 && n <= 10;
    }

    /**
     * updateForm — met à jour l'état du formulaire à chaque frappe.
     *
     * Cette fonction est appelée à chaque événement 'input' sur les deux champs.
     * Elle applique le pattern "validation instantanée" (eager validation) :
     *   - Les messages d'erreur n'apparaissent qu'après une première saisie
     *     (aVal !== '' évite d'afficher une erreur sur un champ encore vierge).
     *   - Le bouton reste désactivé tant que les deux valeurs ne sont pas valides.
     *     Cela évite les appels serveur inutiles pour des données incorrectes.
     *
     * Cette validation est côté client uniquement — jamais faire confiance au client.
     * Le serveur (calc.go et api-service/sum.go) valide de nouveau à la réception.
     */
    function updateForm() {
        var aVal = document.getElementById('input-a').value;
        var bVal = document.getElementById('input-b').value;
        var aOk = isValidBound(aVal);
        var bOk = isValidBound(bVal);

        /**
         * Affichage conditionnel des messages d'erreur locaux.
         * style.display = 'block' / 'none' est la technique DOM classique pour
         * montrer/cacher un élément. L'alternative moderne est classList.toggle()
         * avec une classe CSS, mais ici style.display reste lisible et explicite.
         *
         * Les éléments #a-error et #b-error sont des <small> définis dans index.html
         * avec la classe CSS "error-msg" (display:none par défaut dans app.css).
         */
        document.getElementById('a-error').style.display = (!aOk && aVal !== '') ? 'block' : 'none';
        document.getElementById('b-error').style.display = (!bOk && bVal !== '') ? 'block' : 'none';

        /**
         * Activation du bouton submit.
         * L'attribut HTML "disabled" empêche la soumission du formulaire et le
         * clic sur le bouton. Pico.css lui applique automatiquement un style grisé.
         * HTMX respecte disabled : il ne soumet pas un formulaire avec un bouton désactivé.
         */
        document.getElementById('submit-btn').disabled = !(aOk && bOk);
    }

    /**
     * Attachement des écouteurs d'événements après chargement du DOM.
     *
     * DOMContentLoaded se déclenche quand le HTML est parsé et le DOM prêt,
     * sans attendre les images et feuilles de style (contrairement à window.load).
     * C'est le moment idéal pour attacher des écouteurs sur des éléments HTML.
     *
     * Pourquoi attendre DOMContentLoaded ?
     * Ce fichier est chargé en bas de <body> (après les éléments input), donc
     * document.getElementById trouverait les éléments même sans cet écouteur.
     * Mais c'est une bonne pratique défensive : si le script est déplacé dans
     * <head>, le code continuera de fonctionner correctement.
     *
     * L'événement 'input' se déclenche à chaque modification du champ :
     * frappe au clavier, coller (Ctrl+V), incrément avec les flèches du champ number.
     * C'est différent de 'change' qui ne se déclenche qu'à la perte de focus.
     */
    document.addEventListener('DOMContentLoaded', function () {
        document.getElementById('input-a').addEventListener('input', updateForm);
        document.getElementById('input-b').addEventListener('input', updateForm);
    });

}());
