# gofact — plan de communication

> Document de référence. Toute production de contenu (humaine ou automatisée)
> part d'ici. Révisé à chaque fin de phase.

Dernière révision : 2026-09-04 · Horizon : 2026-09 → 2027-09

---

## 1. La fenêtre

Le 1er septembre 2026, l'obligation de **réception** de factures électroniques
est entrée en vigueur pour **toutes** les entreprises assujetties à la TVA —
micro-entreprises comprises. L'obligation d'**émission** est en place pour les
grandes entreprises et les ETI ; elle tombera le **1er septembre 2027** pour les
PME, TPE et micro-entreprises.

Ce plan couvre les douze mois qui séparent ces deux dates. C'est la seule
fenêtre où l'audience cible de gofact — l'indépendant et la très petite
structure — cherche activement une réponse *avant* d'être en infraction.

Trois faits de marché structurent le positionnement :

| Fait | Conséquence pour gofact |
| --- | --- |
| 166 plateformes agréées (PA, ex-PDP), 4 seulement proposent une offre gratuite | Le marché se lit comme « choisir un abonnement ». gofact n'est pas dans cette liste et ne doit pas y prétendre. |
| Aucune solution open source parmi les PA agréées | Personne n'occupe le créneau « le format est libre, la plateforme est un tuyau ». C'est le nôtre. |
| La plupart des « générateurs de factures gratuits » produisent un PDF non structuré | Il existe une confusion massive à dissiper. Dissiper une confusion, c'est le meilleur contenu qui soit. |

**Le raccourci mental à installer :** *la plateforme est un tuyau, la facture est
un fichier. Le tuyau se change ; le fichier vous appartient.*

---

## 2. Ce que gofact est, et n'est pas

Cette section est contraignante. Elle prime sur toute formulation trouvée
ailleurs.

**gofact est** un binaire Go local qui produit une facture Factur-X conforme
(PDF/A-3 + XML CII EN 16931 embarqué octet pour octet), vérifie sa conformité
avant et après production, et sait la déposer sur la plateforme agréée de
l'utilisateur.

**gofact n'est pas** une plateforme agréée. Il ne transmet rien à la DGFiP. Il
ne remplace pas le passage par une PA — il produit le fichier que la PA
transporte.

Ne jamais laisser une phrase suggérer le contraire, y compris par omission.
C'est une exigence de conformité, pas de modestie : c'est aussi le meilleur
argument du produit. Un outil qui explique clairement ce qu'il ne fait pas est
le seul crédible dans un marché saturé de promesses de conformité.

---

## 3. Positionnement

> **La facturation électronique française, sans abonnement et sans serveur.
> Un binaire, votre navigateur, votre IA.**

Trois piliers. Tout contenu produit doit se rattacher explicitement à l'un
d'eux.

### Pilier A — Souveraineté

La réforme oblige à passer par une plateforme. Elle n'oblige pas à louer son
outil de facturation. gofact garde les factures, les clients et les compteurs
dans un dossier local. Pas de compte, pas de télémétrie, `GOFACT_OFFLINE=1`
coupe jusqu'aux annuaires.

*Public visé :* indépendants, self-hosters, tous ceux que l'abonnement fatigue.

### Pilier B — Conformité vérifiable

« Conforme » est un mot que tout le monde écrit. gofact le rend vérifiable :
règles EN 16931 appliquées **avant** production (l'erreur nomme la règle,
`BR-50`, `BR-CO-15`, et le champ), auto-contrôle du PDF après écriture,
veraPDF et le Schematron officiel en intégration continue publique. Le code est
lisible, la CI est publique.

L'argument technique le plus fort du produit : **l'embarquement verbatim**. Un
assembleur qui re-sérialise le XML via son propre modèle perd les champs
étendus — notamment les adresses de routage PA (BT-34 / BT-49). gofact ne
retouche jamais le XML qu'il a produit.

*Public visé :* développeurs, éditeurs, experts-comptables techniques.

### Pilier C — L'interface, c'est votre langue

Le serveur MCP fait de gofact le premier outil de facturation dont l'interface
est une conversation : « fais-moi la facture d'août pour ACME ». Avec une
garantie que le marketing IA ne donne jamais — **la numérotation légale n'est
pas confiée au modèle**. Le serveur numérote sous verrou, en transaction ;
l'IA écrit un jeton.

*Public visé :* utilisateurs de Claude Code / Claude Desktop / LM Studio,
audience IA agentique, public international.

---

## 4. Audiences, par ordre de priorité

| # | Audience | Ce qu'elle vit | Ce qu'on lui apporte | Canal |
| --- | --- | --- | --- | --- |
| 1 | Indépendant / TPE à l'aise techniquement (dev freelance, consultant, agence) | « Je dois m'équiper avant sept. 2027 et je refuse un abonnement de plus » | Une porte de sortie crédible et gratuite | LinkedIn, docs SEO |
| 2 | Développeurs, communauté open source, self-hosters | Curiosité technique : PDF/A-3 en Go pur, MCP, zéro dépendance | Un objet technique bien fait, lisible, réutilisable | GitHub, Reddit, HN, X |
| 3 | Utilisateurs d'IA agentique / Claude Code | Cherchent des usages MCP réels, pas des démos | Un cas d'usage MCP qui touche de l'argent réel et légal | LinkedIn, X, communautés MCP |
| 4 | Experts-comptables, éditeurs, prescripteurs | Doivent conseiller leurs clients TPE, et se méfient de l'open source | De la pédagogie exacte, sans survente ; un outil qui dit ses limites | Docs, LinkedIn, newsletter |

L'audience 1 convertit, l'audience 2 fait le volume, l'audience 3 fait la
singularité, l'audience 4 fait l'autorité.

---

## 5. Rôle de chaque canal

| Canal | Rôle | Cadence | Mesure |
| --- | --- | --- | --- |
| **Docs (MkDocs, `docs/comprendre/`)** | Capter la recherche de la réforme. Actif qui travaille 12 mois. | 1 article de fond / semaine | Trafic organique, requêtes |
| **LinkedIn** | Audience professionnelle FR. Notoriété + conversations. | 3 posts / semaine (mar. mer. jeu.) | Impressions, commentaires qualifiés, clics dépôt |
| **GitHub** | Le point d'atterrissage. Tout mène ici. | README vivant, notes de version soignées | Stars, forks, issues, installs |
| **Reddit / HN** | Pics d'adoption. À jouer rarement et sur du vrai. | 1 fois par mois **maximum**, sur un jalon réel | Trafic, qualité des retours |
| **X / newsletter** | Recyclage et rappel. | Hebdomadaire, dérivé du contenu principal | Secondaire |

**Règle Reddit / HN :** on ne poste que quand il y a une raison objective (une
version, une fonctionnalité, un chiffre). Un post promotionnel sans substance
sur ces plateformes coûte plus cher qu'il ne rapporte, et durablement.

---

## 6. Séquence en quatre phases

### Phase 1 — Dissiper la confusion (semaines 1-2, sept. 2026)

L'obligation de réception vient de tomber. Le marché est en état de question.
On ne vend rien : on explique. Le contenu le plus performant de toute la
séquence est ici, et il est presque entièrement pédagogique.

Angles : ce qui a changé le 1er septembre exactement · réception ≠ émission ·
un PDF n'est pas une facture électronique · PA, PPF, annuaire : le vocabulaire
· ce que vous avez vraiment jusqu'en 2027.

### Phase 2 — Établir la preuve technique (semaines 3-6)

On passe de « voilà la réforme » à « voilà comment c'est fait ». Contenu qui
démontre la compétence plutôt que de l'affirmer.

Angles : l'embarquement verbatim et les champs perdus · PDF/A-3 sans Java ni
Ghostscript · pourquoi l'IA ne choisit pas le numéro de facture · lire une
erreur `BR-*`.

### Phase 3 — Adoption (semaines 7-10)

On va chercher les utilisateurs là où ils sont. Show HN, Reddit, communautés
MCP. Prérequis : la Phase 2 a produit des preuves citables.

Angles : Show HN en anglais · retours d'installation réels · première
contribution externe · comparatif honnête des chemins possibles pour un indé.

### Phase 4 — Autorité et cap sur 2027 (semaines 11-12, puis rythme de croisière)

On devient une référence sur le sujet et on prépare la vraie échéance de
l'audience : le 1er septembre 2027.

Angles : bilan des premiers mois de réforme · ce qui casse en vrai chez les
gens · le compte à rebours 2027 · la doc comme produit.

---

## 7. Objectifs et mesure

Objectifs à 12 semaines, volontairement modestes et vérifiables :

| Indicateur | Départ (2026-09-04) | Cible S+12 |
| --- | --- | --- |
| Étoiles GitHub | 5 | 25 |
| Articles de fond publiés | 0 | 12 |
| Posts LinkedIn publiés | 0 | 36 |
| Conversations qualifiées (commentaire ou message décrivant un besoin réel) | 0 | 20 |
| Issues / retours d'utilisateurs externes | 0 | 10 |
| Contributeurs externes | 0 | 1 |

L'indicateur qui compte réellement est le cinquième. Des étoiles sans une
seule issue signifient que le produit est admiré et non utilisé.

Relevé consigné chaque vendredi dans `comms/METRICS.md`.

---

## 8. Ce qu'on ne fera pas

- Pas de comparatif à charge nommant des concurrents pour les démolir. On
  compare des approches, pas des marques.
- Pas de peur comme levier. « Vous êtes en infraction » est un ressort
  malhonnête quand l'échéance de l'audience est en 2027.
- Pas de promesse de conformité qui dépasse ce que le code vérifie.
- Pas de volume pour le volume. Un post inutile coûte de l'attention pour tous
  les suivants.
- Pas de sollicitation automatisée en message privé. Les échanges se font en
  public d'abord.
