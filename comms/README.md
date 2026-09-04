# La machine de communication gofact

Ce dossier n'est pas de la documentation : c'est le dispositif lui-même. Une
session automatisée le lit, produit, publie et mesure sans intervention.

```
comms/
├── STRATEGY.md      le plan — positionnement, piliers, audiences, objectifs
├── GUARDRAILS.md    les règles contraignantes — à lire avant toute production
├── CALENDAR.md      le calendrier éditorial S1→S12, daté
├── METRICS.md       le relevé hebdomadaire et le journal des arbitrages
├── queue/           contenus datés, prêts à publier
└── published/       archive de ce qui est parti, avec URL — la mémoire du dispositif
```

Le comportement d'exécution est décrit dans `.claude/skills/gofact-comms/SKILL.md`,
chargé automatiquement par toute session travaillant sur ce dépôt.

## Les trois tâches planifiées

| Quand (Paris) | Tâche | Ce qu'elle fait |
| --- | --- | --- |
| Lundi 08:00 | Production hebdomadaire | Veille réforme, écrit les trois posts et l'article de la semaine, remplit `queue/`, commit |
| Mar. / mer. / jeu. 08:15 | Publication du jour | Publie le contenu daté du jour, archive dans `published/`, relit les commentaires trois heures plus tard |
| Vendredi 17:00 | Bilan | Relève les indicateurs, met à jour `METRICS.md`, ajuste `CALENDAR.md`, répond aux fils en attente |

Les trois tâches sont rattachées à la session Claude Code
`session_01N6raL8fnbX1oGGBxKFqKtb`, qui porte le dépôt et les accès GitHub.
Elles reprennent donc la conversation là où elle s'est arrêtée, plutôt que de
repartir de zéro.

| Tâche | Identifiant | Cron (UTC) |
| --- | --- | --- |
| Production hebdomadaire | `trig_018pCzdqxh4WSrxyHCiMbWhc` | `0 6 * * 1` |
| Publication du jour | `trig_01Ac9uzKVBtDLMeT1Lcrj51J` | `15 6 * * 2,3,4` |
| Bilan hebdomadaire | `trig_012q2uwgS7s4o7WAAAF8YYcw` | `0 15 * * 5` |

`published/` et `METRICS.md` restent malgré tout la mémoire de référence du
dispositif : ils doivent être tenus à jour à chaque passage, pour qu'une
session repartie de zéro puisse reprendre sans rien republier deux fois.

> **Heure d'hiver.** Les tâches sont planifiées en UTC. Paris passe de UTC+2 à
> UTC+1 le 25 octobre 2026 : décaler alors chaque tâche d'une heure plus tard
> en UTC pour conserver l'heure locale.

## Publication

`scripts/publish.sh <fichier>` route selon le champ `canal` de l'en-tête et
selon les jetons présents dans l'environnement. Sans jeton pour le canal
concerné, il sort en **code 3** — la session bascule alors sur le repli manuel :
le contenu est envoyé à Benjamin, prêt à coller, et reste dans la file.

Le dispositif fonctionne donc dès maintenant, en mode assisté, et devient
automatique canal par canal à mesure que les accès sont ouverts.

Test sans appel réseau :

```sh
./scripts/publish.sh --dry-run comms/queue/2026-09-08-linkedin-ce-qui-a-change.md
```

## Accès à ouvrir

**Aucun secret ne va dans ce dépôt.** Les jetons sont des variables
d'environnement de l'environnement d'exécution Claude Code.

| Canal | Variables | Comment l'obtenir | Sans cet accès |
| --- | --- | --- | --- |
| **LinkedIn** | `LINKEDIN_ACCESS_TOKEN`, `LINKEDIN_AUTHOR_URN` | Créer une app sur `developer.linkedin.com`, y ajouter le produit *Share on LinkedIn*, autoriser les portées `w_member_social` et `openid profile`, récupérer le jeton OAuth 2.0 et l'URN du membre | Post envoyé en notification, à coller à la main |
| **X** | `X_ACCESS_TOKEN` | App sur `developer.x.com`, OAuth 2.0 contexte utilisateur, portée `tweet.write` | Recyclage manuel |
| **Documentation** | — | Autorisation de fusionner sur `main` les modifications limitées à `docs/` | Article livré sur la branche, fusion manuelle |
| **Reddit / HN** | — | Volontairement non automatisé | Texte fourni, publication par Benjamin |
| **Newsletter** | à définir | Dépend du fournisseur retenu | Contenu fourni |

Le jeton LinkedIn expire au bout de soixante jours. Une tâche mensuelle de
renouvellement sera ajoutée le jour où l'accès sera en place.

**Reddit et Hacker News restent manuels par choix**, pas par manque d'accès.
Sur ces plateformes, un compte sans historique qui poste un lien est traité
comme du spam, et la sanction porte sur le projet, durablement.
