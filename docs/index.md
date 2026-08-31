# gofact

**Dites à votre IA « fais-moi une facture pour ACME, 2 jours à 600 € » — gofact fait le
reste, en local.**

gofact produit des factures électroniques françaises conformes — **Factur-X** : un
PDF/A-3 archivable dix ans, avec la facture structurée (XML CII EN 16931) embarquée —
et peut les déposer sur une plateforme de dématérialisation (PDP). Il est pensé pour
l'indépendant et la petite structure qui subissent la réforme de la facturation
électronique, et dont l'assistant de gestion est désormais une IA : Claude, LM Studio,
Cursor… tout client [MCP](https://modelcontextprotocol.io).

## Le principe

Votre IA est l'interface ; gofact est le moteur. L'IA discute, rédige et compose la
facture — gofact garantit ce qui ne se négocie pas :

- **La numérotation légale.** Séquence continue, sans trou, jamais réutilisée.
  L'attribution est verrouillée et transactionnelle : c'est le serveur qui numérote,
  jamais le modèle de langage.
- **La conformité.** Règles métier EN 16931 vérifiées *avant* d'émettre (l'erreur nomme
  la règle et le champ fautif), puis auto-contrôle du PDF/A-3 produit. En intégration
  continue, chaque évolution du code est jugée par veraPDF et le Schematron officiel.
- **La cohérence visuelle.** La première facture fige le modèle HTML de
  l'organisation ; les suivantes en repartent, et toute dérive de structure est
  signalée.
- **L'archivage.** Chaque facture vit dans le dossier de son organisation : HTML,
  données structurées, PDF, registre et journal d'audit.

## Vos données restent chez vous

Aucun serveur à nous, aucun compte, aucune télémétrie. Les factures, les clients, les
compteurs : tout vit dans un dossier local qui vous appartient. Les seules requêtes
sortantes sont celles que vous déclenchez — recherche d'un client dans l'annuaire public
des entreprises (SIRENE), vérification d'adressabilité Peppol, dépôt sur *votre* PDP —
et `GOFACT_OFFLINE=1` les coupe toutes. Seule la chaîne recherchée part sur le réseau,
jamais le contenu d'une facture.

## Un binaire, un navigateur

gofact est un **binaire Go autonome** : pas de Ghostscript, pas de Java, pas de
dépendance à installer. Il ne lui faut qu'un navigateur (Chrome, Edge, Brave ou
Chromium — Edge est déjà sur tout Windows) pour transformer le HTML en PDF, fidèle au
`Ctrl+P`.

```
JSON  ── règles EN 16931 ─▶  refus si non conforme, avant toute production
      ── gofact ──────────▶  factur-x.xml (CII EN 16931)
HTML  ── navigateur ──────▶  PDF (rendu identique à l'impression)
PDF + XML ── Go pur ──────▶  Factur-X (PDF/A-3, XML embarqué octet pour octet)
Factur-X ── auto-contrôle ▶  relecture, structures et intégrité vérifiées
```

## Par où commencer

1. [Installer gofact](installation.md) — un script, ou le plugin Claude Code.
2. [Premiers pas](demarrage.md) — créer son organisation et sa première facture en
   conversation.
3. [Envoi PDP](pdp.md) — quand vous êtes prêt à entrer dans le circuit officiel.

## Licence

Logiciel libre, **GNU AGPL v3**. Le code est sur
[GitHub](https://github.com/kOlapsis/gofact) — issues et contributions bienvenues, au
rythme d'un projet maintenu en best effort.
