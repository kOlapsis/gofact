# Premiers pas

Tout se fait en conversation avec votre IA, une fois le serveur MCP
[installé](installation.md). Ce qui suit montre le déroulé — et ce qui se passe
réellement derrière.

## Créer son organisation

> « Je veux émettre mes factures avec gofact. Je suis auto-entrepreneur. »

L'IA vous demandera votre identité — nom, SIRET, adresse, IBAN de règlement — et créera
votre **dossier d'organisation** (outil `init_organization`) : un dossier local qui
contiendra votre registre de numérotation, vos factures et votre configuration. Rien
n'est inventé : ce que vous ne donnez pas n'est pas rempli.

Vous pouvez aussi le faire au terminal :

```sh
gofact org init -path ~/Documents/Factures \
  -name "Studio Exemple" -siret 12345678900014 \
  -address "1 rue de l'Exemple" -postal-code 33000 -city Bordeaux \
  -iban FR7630001007941234567890185
```

En **franchise en base de TVA (art. 293 B)**, il n'y a rien d'autre à configurer : c'est
le régime par défaut, mentions légales comprises. Assujetti à la TVA ? Ajoutez votre
numéro intracommunautaire (`GOFACT_SELLER_VAT_NUMBER` dans le `.env` du dossier) et
l'IA précisera le taux par facture.

!!! warning "Vous facturiez déjà ? Reprenez votre numérotation"
    C'est **le** point critique : la séquence doit continuer sans trou ni doublon.
    L'IA vous demandera votre dernier numéro émis (ex. `2026011`) et gofact reprendra
    au suivant. Au terminal : `gofact org init … -last-number 2026011`, ou plus tard
    `gofact org set-counter -last-number 2026011`. Le compteur ne peut que monter —
    l'abaisser réutiliserait des numéros déjà émis, et gofact le refuse.

## Première facture

> « Fais-moi une facture pour ACME, 2 jours de développement à 600 € »

Déroulé réel, outil par outil :

1. **`search_client`** — ACME est cherché dans votre historique, puis dans l'annuaire
   public des entreprises (SIRENE) : l'IA vous fait confirmer le bon candidat, avec son
   SIRET et son adresse — vous ne saisissez rien.
2. **Composition du modèle** — c'est votre première facture, alors l'IA la conçoit
   *avec vous* : dites ce que vous voulez (logo, couleurs, ton, mentions), elle vous
   montre un **PDF d'aperçu** (`preview_invoice` — marqué SPÉCIMEN, rien n'est
   consommé) et itère jusqu'à ce que le rendu vous plaise. Cette facture deviendra le
   modèle de référence ; les suivantes en repartiront. Et rien n'est gravé dans le
   marbre : « change mon modèle, plus sobre » fonctionne à tout moment — l'IA itère en
   aperçu puis officialise le nouveau visuel à la facture suivante.
3. **`preview_next_number`** — l'IA vous annonce le numéro (ex. `2026001`) et
   récapitule : client, lignes, totaux. Le numéro n'est pas encore consommé.
4. **Votre accord**, puis **`create_invoice`** — en une transaction : numéro attribué,
   HTML + données + **PDF Factur-X** écrits dans le dossier, registre mis à jour,
   conformité vérifiée. En cas d'échec, rien n'est consommé : pas de trou dans la
   numérotation.

Le PDF est prêt à envoyer à votre client ; le dossier garde tout.

## Et ensuite

- « Refais une facture pour ACME, 3 jours ce mois-ci » — coordonnées et modèle sont
  réutilisés, le numéro suit.
- « Où en est ma facturation cette année ? » — `list_invoices` répond depuis le
  registre.
- « Envoie la 2026002 sur ma plateforme » — voir [Envoi PDP](pdp.md) : il faudra vos
  identifiants PDP, et votre confirmation explicite à chaque dépôt.

!!! tip "Plusieurs activités ?"
    Un dossier = une entité émettrice, avec sa propre identité et sa propre
    numérotation. Créez-en autant que nécessaire : l'IA vous demandera laquelle
    utiliser quand le contexte ne suffit pas. Détails dans
    [le dossier d'organisation](guide/organisation.md).
