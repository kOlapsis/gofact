# Envoi PDP

La réforme fait transiter les factures B2B par des **plateformes de dématérialisation
partenaires** (PDP). gofact dépose vos Factur-X sur la vôtre — c'est la seule opération
irréversible de l'outil, et elle est traitée comme telle.

## Prérequis

1. **Un compte chez une PDP.** gofact n'est pas une plateforme et ne s'intercale pas :
   le contrat, l'immatriculation et les identifiants sont les vôtres. Fournisseur
   implémenté : [SuperPDP](https://superpdp.tech) ; l'architecture accepte d'autres
   fournisseurs (`GOFACT_PDP`).
2. **Vos identifiants dans le `.env` du dossier** de l'organisation — jamais en
   conversation avec l'IA :

    ```sh
    SUPERPDP_CLIENT_ID="…"
    SUPERPDP_CLIENT_SECRET="…"
    # SUPERPDP_BASE="https://api.superpdp.tech"   # optionnel
    ```

3. **Un destinataire adressable.** La PDP route sur l'adresse électronique de
   l'acheteur (BT-49), pas sur son adresse postale. `find_routing_address` vérifie
   l'annuaire Peppol à partir du SIREN :

    - trouvé → `electronic_address` + `electronic_address_scheme` dans les données de
      la facture. Pour une PDP française, préférer le scheme `0225` ; si le dépôt est
      rejeté, réessayer en `0002` ;
    - introuvable → **ne pas tenter l'envoi** : livrez le PDF par un autre canal.

    Sans adresse déclarée, gofact route l'acheteur sur son SIREN en `0225` et inscrit
    cette adresse dans le sidecar JSON à la création. Un acheteur dont la seule
    adresse est un e-mail n'est pas déposé : `send_invoice` refuse et renvoie vers
    `find_routing_address`, plutôt que de laisser la plateforme rejeter.

!!! note "L'émetteur aussi doit être adressable"
    La PDP exige que le vendeur de la facture corresponde à la société du compte
    authentifié (identifiant légal BT-30) et porte sa propre adresse de routage. Par
    défaut gofact route l'émetteur sur son SIREN en scheme `0225`.

## Rejet

Un rejet arrive en général dans la seconde. `send_invoice` relit le cycle de vie
juste après le dépôt et remonte immédiatement un rejet (`fr:213`) avec ses motifs,
règle par règle — par exemple `BR-FR-05/BT-22 : La mention relative aux frais de
recouvrement (code PMT) est absente`. `get_invoice_status` expose les mêmes motifs,
et le journal du dossier trace l'événement `pdp_rejected`.

## Déposer

En conversation, l'IA n'appellera `send_invoice` qu'après **votre confirmation
explicite** — l'outil est marqué destructif et refuse de s'exécuter sans elle. Au
terminal :

```sh
gofact send -pdf "2026001 - ACME.pdf" -poll
```

## Suivre

Le cycle de vie s'affiche au dépôt et se consulte ensuite (`get_invoice_status`,
`-poll`) :

| Statut | Sens |
|---|---|
| `fr:200` | déposée sur la plateforme |
| `fr:201` | émise vers le destinataire |
| `fr:202` | reçue par la plateforme du destinataire |

Chaque dépôt est tracé dans le `journal.ndjson` du dossier : quoi, quand, sous quelle
référence.

## Signaler un encaissement

Quand le client a payé, signalez-le à la plateforme : le statut `fr:212` « Encaissée »
alimente la déclaration des paiements à l'administration (e-reporting). Une facture
payée dès l'émission se signale juste après son dépôt.

En conversation, l'IA appelle `report_payment` après votre confirmation. Au terminal :

```sh
gofact paid -number 2026001                                  # total, encaissé aujourd'hui
gofact paid -number 2026001 -date 2026-09-12                 # total, encaissé à cette date
gofact paid -number 2026001 -date 2026-09-12 -amount 300.00  # paiement partiel
```

Un encaissement daté ou partiel doit être ventilé par taux de TVA. gofact fait ce
calcul à partir des données de la facture, au prorata du TTC de chaque taux. La date
d'encaissement apparaît ensuite dans `list_invoices` et dans l'export comptable.

## Factures reçues

Les factures de vos fournisseurs arrivent sur la plateforme. `gofact sync` les
récupère : chaque nouvelle facture est rangée en PDF dans le dossier `recues/AAAA-MM/`
de l'organisation, et une notification s'affiche sur le bureau. Une facture reçue en
XML seul est convertie en PDF lisible par la plateforme.

Le fichier `pdp-sync.json` retient la dernière facture récupérée : chaque passage ne
rapatrie que les nouvelles, et un passage interrompu reprend où il s'était arrêté.

Pour être prévenu sans y penser, planifiez la commande (voir
[Export comptable](guide/export.md#planifier)). En conversation, l'IA voit les factures
reçues avec `list_received_invoices`.

## Rejets courants

| Message | Cause | Correction |
|---|---|---|
| `receiver address does not exist in peppol directory` | BT-49 absente ou fausse | `find_routing_address`, scheme `0225` puis `0002` |
| Vendeur ≠ compte | l'identité de l'organisation ne correspond pas au compte PDP | vérifier SIREN/SIRET du `.env` |
| Identifiants refusés | secret expiré ou mal copié | régénérer sur la plateforme, mettre à jour le `.env` |
