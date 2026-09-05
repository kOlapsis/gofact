# Journal de mesure

Relevé chaque vendredi par la routine de bilan. Une ligne par semaine, jamais
de ligne rétroactive : une semaine non relevée reste vide.

La première colonne est la métrique nord : téléchargements cumulés des assets
de release, tous systèmes confondus. Relevés avec `mcp__github__get_latest_release`
en sommant `assets[].download_count` — l'API GitHub publique est bloquée par le
proxy réseau de la session, l'outil MCP ne l'est pas.

| Semaine | Téléch. cumulés | Δ semaine | Issues tierces | Étoiles | Forks | Contenus publiés | Conversations qualifiées | Décision de la semaine |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S0 (2026-09-04) | 3 | — | 0 | 5 | 1 | 0 | 0 | Mise en place du dispositif ; métrique nord arrêtée |

## Journal des arbitrages

Toute substitution de sujet, tout créneau sauté et toute modification du
calendrier est consignée ici, avec sa raison.

- **2026-09-04** — Création du plan. Périmètre arrêté : gofact seul, quatre
  canaux, publication automatisée là où un jeton est disponible.
