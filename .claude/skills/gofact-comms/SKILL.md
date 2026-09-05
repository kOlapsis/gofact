---
name: gofact-comms
description: "Exécute le plan de communication de gofact — veille réforme, production de contenu, remplissage de la file, publication multicanal et relevé d'indicateurs. Utiliser ce skill pour toute tâche de communication, de contenu ou de diffusion autour de gofact : produire les posts de la semaine, rédiger un article de fond pour la documentation, publier un contenu en attente dans handoff/comms/queue, analyser les commentaires d'un post, faire le point sur les indicateurs, ou ajuster le calendrier éditorial. Se déclenche aussi sur les tâches planifiées « production hebdomadaire », « publication du jour » et « bilan hebdomadaire »."
---

# Exécution du plan de communication gofact

## Avant toute chose

Lire, dans cet ordre, et sans exception :

1. `handoff/comms/GUARDRAILS.md` — **contraignant**. Prime sur tout le reste.
2. `handoff/comms/STRATEGY.md` — positionnement, piliers, audiences.
3. `handoff/comms/CALENDAR.md` — le sujet prévu aujourd'hui.
4. `handoff/comms/published/` — ce qui est déjà parti. Ne jamais republier un angle.

Puis identifier laquelle des quatre routines ci-dessous s'applique.

---

## Routine 1 — Production hebdomadaire (lundi)

**Objectif :** remplir `handoff/comms/queue/` pour toute la semaine, en un seul passage.

1. **Veille.** Trois à cinq recherches web sur l'actualité de la réforme des
   sept derniers jours : décrets et arrêtés, évolutions de la liste des
   plateformes agréées, incidents et retours d'expérience, communication de la
   DGFiP. Objectif : détecter ce qui invalide ou renforce le calendrier.
2. **Arbitrage.** Si l'actualité offre un sujet nettement plus fort que celui
   prévu, le substituer et noter la substitution dans `handoff/comms/METRICS.md`. Sinon,
   suivre le calendrier. Ne pas chercher l'originalité à tout prix : le
   calendrier a été construit avec une logique de progression.
3. **Production.** Pour chaque créneau de la semaine, écrire le contenu complet,
   prêt à publier, dans `handoff/comms/queue/AAAA-MM-JJ-canal-slug.md` au format
   ci-dessous. Le skill `linkedin-post-generator` fournit les trois angles
   (contrariant, douleur, résultat) — en retenir **un seul** par créneau, celui
   qui sert le pilier prévu au calendrier.
4. **Article de fond.** Rédiger l'article de la semaine dans
   `docs/comprendre/`, l'ajouter à la navigation de `mkdocs.yml`, et le
   référencer dans la file. Un article fait 900 à 1600 mots, répond à une
   question réelle dès les trois premières lignes, et cite ses sources.
5. **Vérification.** Repasser chaque contenu contre `GUARDRAILS.md`, section par
   section. Toute affirmation réglementaire non listée comme établie est
   revérifiée à la source, ou retirée.
6. **Commit** sur la branche de travail, message `comms: file de la semaine SNN`.

---

## Routine 2 — Publication du jour (mardi, mercredi, jeudi)

1. Lire le fichier de `handoff/comms/queue/` daté d'aujourd'hui. Aucun fichier pour
   aujourd'hui : ne rien publier, ne rien improviser, s'arrêter là.
2. Relire le contenu une dernière fois contre `GUARDRAILS.md`.
3. Publier via `handoff/publish.sh <fichier>`. Le script route selon le champ
   `canal` de l'en-tête et selon les jetons présents dans l'environnement.
4. **Si le jeton du canal est absent**, le script sort en code 3 : envoyer alors
   le contenu à Benjamin par notification, prêt à coller, et laisser le fichier
   dans la file avec `statut: à-publier-manuellement`.
5. Publication réussie : déplacer le fichier vers `handoff/comms/published/`, y inscrire
   l'URL et l'horodatage, committer.
6. Trois heures après la publication d'un post LinkedIn, relire les commentaires
   et appliquer le skill `linkedin-lead-capture`. Répondre publiquement aux
   commentaires techniques et aux désaccords constructifs. **Aucun message privé
   automatisé** — les messages privés proposés sont soumis à Benjamin.

---

## Routine 3 — Bilan hebdomadaire (vendredi)

1. Relever les indicateurs, en commençant par la métrique nord : **somme des
   `download_count` des assets de la dernière release**, via
   `mcp__github__get_latest_release` (l'API GitHub publique est bloquée par le
   proxy, l'outil MCP ne l'est pas). Puis issues ouvertes par des tiers,
   étoiles, forks, trafic de la documentation si disponible, impressions et
   commentaires LinkedIn — dans cet ordre d'importance décroissante.
2. Consigner la ligne de la semaine dans `handoff/comms/METRICS.md`.
3. Comparer aux cibles de `STRATEGY.md` § 7 et au diagnostic de
   `handoff/distribution.md`. Une semaine où les téléchargements ne bougent pas est une
   semaine sans résultat, quel que soit le nombre d'impressions obtenues.
4. **Décider, et écrire la décision.** Un sujet qui ne produit rien deux
   semaines de suite sort du calendrier. Un sujet qui produit des conversations
   qualifiées est décliné. Modifier `CALENDAR.md` en conséquence.
5. Répondre aux commentaires et issues restés sans réponse.
6. Notifier Benjamin uniquement s'il y a une décision à prendre ou une
   conversation qualifiée à reprendre. Une semaine sans événement notable ne
   produit aucune notification.

---

## Routine 4 — Fenêtre mensuelle Reddit / HN

À ne déclencher que si le mois a produit un jalon objectif : une version, une
fonctionnalité visible, un chiffre mesuré. Sans jalon, on saute le mois.

- Rédiger en anglais pour Hacker News, en français pour r/france, en anglais
  pour r/selfhosted.
- Le texte décrit la mécanique technique, pas les bénéfices. Ce public déteste
  le registre promotionnel et le sanctionne durablement.
- Publier soi-même **uniquement** si un jeton du canal est présent. À défaut,
  transmettre à Benjamin — sur ces plateformes, un compte sans historique qui
  poste un lien est traité comme du spam.
- Rester disponible dans les six heures qui suivent pour répondre aux
  questions. Un fil laissé sans réponse est pire que pas de fil du tout.

---

## Format d'un fichier de file

```markdown
---
date: 2026-09-08
heure: "08:15"
canal: linkedin          # linkedin | x | reddit | hn | newsletter | docs
pilier: A                # A souveraineté | B conformité | C interface IA
sujet: Ce qui a changé le 1er septembre
statut: prêt             # prêt | publié | à-publier-manuellement | abandonné
url:                     # renseignée à la publication
---

[Le contenu exact à publier, sans commentaire ni balise.]
```

---

## Rappels qui coûtent cher quand on les oublie

- gofact **n'est pas** une plateforme agréée. Il produit le fichier ; la
  plateforme agréée le transporte. Cette précision figure dans tout contenu qui
  touche à la transmission.
- L'obligation d'émission pour les TPE est en **septembre 2027**, pas 2026.
  Écrire l'inverse, c'est mentir à son audience pour créer de l'urgence.
- Jamais d'anecdote inventée, jamais de retour utilisateur fictif, jamais de
  chiffre d'usage non mesuré.
- Le plafond de trois posts par semaine n'est pas un objectif. Sauter un
  créneau faute de bon sujet est un résultat acceptable.
- Publier moins et mieux : chaque contenu faible prélève de l'attention sur
  tous les suivants.
