// Validation locale du formulaire A+B.
// Remplace Alpine.js pour supprimer le besoin de CSP unsafe-eval (N4).
// A1 : active le swap HTMX pour les réponses 4xx/5xx (désactivé par défaut en HTMX v1.9).
htmx.config.responseHandling = [{ code: '.*', swap: true }];

(function () {
    'use strict';

    function isValidBound(val) {
        var trimmed = val.trim();
        if (trimmed === '') return false;
        var n = Number(trimmed);
        return Number.isInteger(n) && n >= 0 && n <= 10;
    }

    function updateForm() {
        var aVal = document.getElementById('input-a').value;
        var bVal = document.getElementById('input-b').value;
        var aOk = isValidBound(aVal);
        var bOk = isValidBound(bVal);

        document.getElementById('a-error').style.display = (!aOk && aVal !== '') ? 'block' : 'none';
        document.getElementById('b-error').style.display = (!bOk && bVal !== '') ? 'block' : 'none';
        document.getElementById('submit-btn').disabled = !(aOk && bOk);
    }

    document.addEventListener('DOMContentLoaded', function () {
        document.getElementById('input-a').addEventListener('input', updateForm);
        document.getElementById('input-b').addEventListener('input', updateForm);
    });
}());
