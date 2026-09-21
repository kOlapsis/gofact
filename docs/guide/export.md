# Export comptable

L'API de la plateforme ne permet pas de donner à votre comptable un accès en lecture.
gofact lui prépare donc chaque mois un dossier qu'il n'a qu'à ouvrir.

## Ce que contient l'export

```
<destination>/<organisation>/2026-09/
├── emises/              les PDF Factur-X émis dans le mois
├── recues/              les PDF des factures fournisseurs reçues dans le mois
└── recapitulatif.csv    une ligne par facture
```

Le récapitulatif donne, pour chaque facture : le sens (émise ou reçue), le numéro,
la date, le client ou le fournisseur, les montants HT, TVA et TTC, la devise, la date
d'encaissement quand elle a été signalée, et le fichier. Il utilise le point-virgule
comme séparateur et la virgule décimale : il s'ouvre directement dans Excel ou
LibreOffice en français.

Une facture est rangée dans le mois de sa date d'émission. L'export peut être refait
autant de fois que nécessaire : un PDF identique n'est pas recopié, et le
récapitulatif est régénéré à chaque fois.

## Choisir la destination

Renseignez `GOFACT_EXPORT_DIR` dans le `.env` de l'organisation (ou dans
`~/.config/gofact/.env` pour toutes vos organisations). Le plus simple est de viser
un dossier synchronisé et partagé avec le comptable (Nextcloud, Dropbox, OneDrive…).

```sh
GOFACT_EXPORT_DIR="/home/moi/Nextcloud/Comptabilité"
```

## Exporter à la demande

```sh
gofact export -month 2026-09              # vers GOFACT_EXPORT_DIR
gofact export -month 2026-09 -to ~/export # ou vers un autre dossier
```

En conversation : « exporte les factures de septembre pour le comptable » (outil
`export_invoices`).

## Planifier

`gofact sync` récupère les factures reçues, affiche une notification pour chaque
nouvelle facture, puis met à jour l'export du mois en cours et du mois précédent si
`GOFACT_EXPORT_DIR` est renseigné. Pour qu'il tourne tout seul :

```sh
gofact schedule              # montre ce qui serait installé (rien n'est modifié)
gofact schedule -yes         # installe, toutes les heures
gofact schedule -every 30m -yes
gofact schedule -remove -yes # retire la planification
```

| Système | Mécanisme |
|---|---|
| Linux | timer systemd utilisateur (`~/.config/systemd/user/gofact-sync.timer`) |
| macOS | LaunchAgent (`~/Library/LaunchAgents/com.kolapsis.gofact-sync.plist`) |
| Windows | tâche planifiée `gofact-sync` |

!!! note "Identifiants de la tâche planifiée"
    La tâche planifiée ne voit pas les réglages de l'extension Claude Desktop. Les
    identifiants SuperPDP doivent se trouver dans le `.env` de l'organisation ou dans
    `~/.config/gofact/.env`.
