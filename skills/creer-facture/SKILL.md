---
name: creer-facture
version: 3.0.0
description: Crée une facture électronique française conforme (Factur-X PDF/A-3 + XML CII EN 16931) en s'appuyant sur les outils MCP gofact — numérotation légale, registre, modèle figé, envoi PDP. Gère factures standard, acomptes et soldes.
triggers:
  - créer une facture
  - nouvelle facture
  - facturer
  - facture d'acompte
  - facture de solde
  - générer une facture
allowed-tools:
  - Read
  - Bash
  - AskUserQuestion
---

# Créer une facture

Le travail passe par les **outils MCP `gofact`** (déclarés par ce plugin) : c'est le serveur
qui attribue les numéros, écrit le registre, génère le PDF Factur-X et garantit la
conformité EN 16931. Ce skill décrit le déroulé et les règles de composition — il ne
recalcule rien, n'écrit rien et n'invente rien que les outils savent faire.

> Si les outils `gofact` (`list_organizations`, `create_invoice`…) n'apparaissent pas :
> le binaire n'a pas encore été compilé. Lancer une fois
> `"${CLAUDE_PLUGIN_ROOT}/scripts/mcp-launcher.sh" </dev/null` (Bash) puis redémarrer la
> session, ou demander à l'utilisateur d'exécuter `gofact install -yes`.

## Déroulé

### 1 — L'organisation
`list_organizations`. Plusieurs → demander laquelle. Aucune → collecter l'identité auprès
de l'utilisateur (nom, SIRET, adresse, IBAN — **ne rien inventer**) et `init_organization`.

### 2 — Le client
`search_client` avec le nom donné. L'outil cherche d'abord l'historique (coordonnées
validées par l'usage, adresse de routage comprise — **toujours les réutiliser**), puis
l'annuaire SIRENE. Faire confirmer le bon candidat par l'utilisateur. Pour un envoi PDP,
compléter avec `find_routing_address` (SIREN → adressabilité Peppol) ; destinataire
introuvable = pas d'envoi PDP, livraison du PDF par un autre canal.

### 3 — Les prestations
Collecter : type (`standard` · `acompte` · `solde`), lignes (libellé, quantité en jours ou
unités, PU HT), date d'émission (défaut aujourd'hui), échéance (défaut à réception),
devis de référence le cas échéant.
- Acompte/solde : une seule ligne (`unit` = `unit`, `quantity` = `"1.00"`,
  `amount_ht_cents` = le montant), pourcentage et devis rappelés dans le libellé.
- **Tous les montants en CENTIMES** dans la spec (`1 250,50 €` → `125050`).
- Affichage : espace fine pour les milliers, décimales seulement si non rondes
  (`840 €`, `1 120 €`, `1 250,50 €`).

### 4 — Le HTML
`get_invoice_template` :
- **Modèle figé existant** → repartir de ce HTML tel quel, n'adapter que les contenus
  (client, lignes, montants, dates). Les factures d'une même entité doivent se ressembler.
- **Pas de modèle** (première facture) → composer une facture A4 soignée : CSS embarqué,
  **polices système** (pas de webfont), **pas de `<a href>`**, logo vectoriel inline s'il
  y en a un, mentions légales françaises (pénalités de retard 3× taux légal, indemnité de
  recouvrement 40 €, escompte : néant), et le régime de TVA de l'organisation.
- Dans les deux cas : le jeton `{{NUMERO}}` à l'emplacement du numéro — jamais un numéro
  écrit à la main.

### 5 — Confirmation puis création
`preview_next_number`, puis récapituler à l'utilisateur : numéro annoncé, client, lignes,
totaux HT/TTC, date. **Après son accord seulement** : `create_invoice` (HTML + spec).
L'outil attribue le numéro, écrit HTML/JSON/PDF dans le dossier de l'organisation, inscrit
le registre et vérifie la conformité — en une transaction. Relayer le chemin du PDF et
tout avertissement retourné (dérive du modèle notamment).

### 6 — Envoi PDP (optionnel)
Uniquement si l'utilisateur le demande, avec une confirmation explicite (« déposer la
facture N sur la PDP ? ») : `send_invoice` avec `confirm=true`, puis `get_invoice_status`
pour le cycle de vie (`fr:200` déposée → `fr:201` émise → `fr:202` reçue).

## Règles à ne pas transgresser
- Le numéro vient **toujours** du serveur : jamais inventé, jamais réutilisé, jamais
  antidaté. `preview_next_number` annonce, `create_invoice` attribue.
- Une erreur d'outil (`BR-50` : IBAN manquant, « vendeur non configuré »…) se **relaie**
  à l'utilisateur avec l'action corrective qu'elle nomme — on ne contourne pas, on ne
  réessaie pas en dégradé.
- Aucun secret (identifiants PDP) ne se demande ni ne se répète en conversation : ces
  valeurs vont dans le `.env` du dossier de l'organisation.
- Ne jamais éditer les fichiers du dossier de l'organisation à la main (registre,
  journal, factures émises) : tout passe par les outils.
