# Distribution de gofact

> Document de pilotage. Un seul objectif : que gofact soit installé et utilisé
> par des indépendants et des TPE françaises. Tout le reste en découle.
>
> Établi le 2026-09-04 · Revu tous les vendredis

---

## La métrique nord

**Téléchargements d'assets de release, par semaine.**

Le produit ne collecte aucune télémétrie, et ne le fera pas : c'est le cœur de
son positionnement. GitHub publie en revanche le nombre de téléchargements de
chaque asset de release, gratuitement et sans rien demander à l'utilisateur.
C'est la seule mesure d'usage honnête disponible, et elle suffit.

**Base au 2026-09-04**, release `v0.1.0` publiée le 31 août :

| Asset | Téléchargements |
| --- | --- |
| `gofact_0.1.0_linux_amd64.tar.gz` | 3 |
| `gofact_0.1.0_darwin_arm64.tar.gz` | 0 |
| `gofact_0.1.0_darwin_amd64.tar.gz` | 0 |
| `gofact_0.1.0_linux_arm64.tar.gz` | 0 |
| `gofact_0.1.0_windows_amd64.zip` | 0 |
| `gofact_0.1.0_windows_arm64.zip` | 0 |
| **Total** | **3** |

Trois. Dont vraisemblablement quelques-uns de l'auteur. C'est le chiffre à
partir duquel on travaille, et il vaut mieux le regarder en face que le
maquiller.

**Métriques secondaires**, dans cet ordre d'importance :

1. Issues et discussions ouvertes par des tiers — la preuve qu'on utilise, pas
   qu'on admire. Actuellement : **0**.
2. Étoiles GitHub — indicateur de notoriété, pas d'usage. Actuellement : **5**.
3. Trafic de la documentation.
4. Impressions et commentaires sociaux — le plus loin de la vérité.

---

## Diagnostic

Le funnel n'a pas un problème de conversion. Il a un problème d'entrée.

```
   audience atteinte     ~0        ← le vrai goulot
          ↓
   visite du dépôt        ?
          ↓
   étoile                 5
          ↓
   téléchargement         3
          ↓
   première facture       ?        ← non mesurable, à estimer par audit
          ↓
   usage récurrent        0 signal
```

Avec cinq étoiles et zéro issue, aucune conclusion n'est possible sur la
qualité du produit ou de sa documentation : l'échantillon est vide. Optimiser
une page d'atterrissage que personne ne visite est du travail perdu.

**Conséquence sur l'ordre des priorités.** Tant que le trafic est nul, l'effort
va à la mise en visibilité et à la levée des blocages durs, pas au raffinement.
Le raffinement viendra quand il y aura des utilisateurs pour dire ce qui cloche.

**Une réserve, tenue en permanence.** Envoyer du trafic vers un produit qui
échoue à l'installation brûle l'audience une fois pour toutes. On ne pousse
aucun canal tant que le parcours d'installation n'a pas été vérifié de bout en
bout sur les trois systèmes. C'est l'objet de l'audit d'embarquement.

---

## Les trois paris du trimestre

### Pari 1 — Le fichier trouvé, pas le produit cherché

Personne ne cherche « gofact ». Des dizaines de milliers d'indépendants
cherchent « comment recevoir une facture électronique », « Factur-X gratuit »,
« que faire pour 2027 ». La documentation doit répondre à ces questions mieux
que quiconque, et gofact être la conclusion évidente de la réponse.

*Levier :* un article de fond par semaine dans `docs/comprendre/`, indexable,
sourcé, honnête sur les limites du produit. Le contenu éditorial de `comms/`
sert ce pari.

### Pari 2 — Être présent là où l'audience regarde déjà

Un logiciel qui n'est référencé nulle part n'existe pas. Registres MCP,
annuaires de plugins, listes *awesome*, annuaires de logiciels libres
francophones, gestionnaires de paquets. Chaque référencement est un travail
ponctuel dont le rendement court des années.

*Levier :* soumissions systématiques, priorisées par ratio impact/effort. Objet
de l'audit des canaux.

### Pari 3 — Le chemin le plus court entre la découverte et la première facture

Le taux de conversion réel d'un outil en ligne de commande se joue dans les dix
premières minutes. Chaque étape, chaque variable à renseigner, chaque message
d'erreur obscur coûte une fraction des arrivants.

*Levier :* mesurer le temps réel jusqu'à la première facture, puis le réduire.
Objet de l'audit d'embarquement.

---

## Règles d'arbitrage

- **Un canal se teste, il ne se croit pas.** Toute action de distribution est
  consignée avec sa date, son coût en temps et ce qu'elle a produit. Un canal
  sans effet mesurable au bout de quatre semaines est abandonné, pas rejoué
  « en mieux ».
- **La vérité avant le volume.** Une affirmation fausse sur la réforme coûte
  plus cher que dix publications manquées. `comms/GUARDRAILS.md` s'applique à
  toute la distribution, pas seulement au contenu éditorial.
- **On ne brûle pas une communauté.** Reddit, Hacker News, LinuxFr et les
  forums d'indépendants sanctionnent durablement la promotion déguisée. On y
  entre avec une contribution réelle, ou on n'y entre pas.
- **Le produit passe avant la promotion.** Un blocage d'installation prime sur
  toute publication prévue au calendrier, y compris le jour même.

---

## Ce que je décide seul, ce qui remonte

**Décidé sans demander** — modifications du dépôt sur la branche de travail,
contenu éditorial, priorisation des canaux, correctifs d'embarquement,
documentation, référencements gratuits et réversibles, planification des tâches
automatisées.

**Soumis avant exécution** — tout ce qui est irréversible ou engage le nom de
Benjamin auprès de tiers : publication d'une version étiquetée, fusion sur
`main`, création de comptes, prise de contact avec une organisation, dépôt dans
un gestionnaire de paquets qui suppose un mainteneur identifié, toute dépense.

**Remonté sans attendre le vendredi** — un blocage d'installation, une erreur
réglementaire publiée, un canal qui se retourne.

---

## Journal des décisions

| Date | Décision | Raison |
| --- | --- | --- |
| 2026-09-04 | Métrique nord : téléchargements de release, pas étoiles | L'absence de télémétrie est un choix produit ; les téléchargements sont la seule mesure d'usage honnête disponible |
| 2026-09-04 | Aucune poussée de trafic avant vérification du parcours d'installation | Envoyer du monde sur une installation cassée brûle l'audience une seule fois |
| 2026-09-04 | Notes de version transformées en page d'atterrissage | La page de release est le premier écran de tout arrivant par un annuaire ; elle affichait une liste de hachés git |
