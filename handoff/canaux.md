# État des canaux de distribution

> **À lire avant toute action de référencement.** Ce fichier est la source de
> vérité : une session qui ne le consulte pas refait du travail déjà livré.
> Chaque ligne porte une preuve vérifiable — une URL ou une commande — pour que
> l'état se contrôle en trente secondes plutôt que de se croire.
>
> Relevé du 2026-09-05 · Tenu à jour par la routine du vendredi

---

## Métrique nord

**Téléchargements d'assets de release.** Aucune télémétrie dans le produit ;
GitHub publie ces compteurs gratuitement. Relevé via
`mcp__github__get_latest_release`, en sommant `assets[].download_count`.

| Relevé | v1.1.0 archives | v1.1.0 bundles `.mcpb` | Étoiles | Forks | Issues tierces |
| --- | --- | --- | --- | --- | --- |
| 2026-09-05 | 6 | 42 | 5 | 1 | 0 |

Les 42 téléchargements de bundles sont du moissonnage de registres, pas des
humains. **Les six archives sont le seul chiffre qui parle d'usage réel.**

---

## Fait — ne pas refaire

| Canal | Preuve vérifiable | Entretien |
| --- | --- | --- |
| **Release GitHub** | `v1.1.0`, six archives + `checksums.txt` + six `.mcpb` | `release.yml`, sur tag ou déclenchement manuel |
| **Registre officiel MCP** | `curl "https://registry.modelcontextprotocol.io/v0/servers?search=gofact"` → `io.github.kOlapsis/gofact` 1.1.0, six paquets `mcpb` | `mcp-registry.yml`, à chaque release, OIDC sans secret |
| **Cask Homebrew** | `Casks/gofact.rb` dans ce dépôt, en 1.1.0 | `packaging.yml`, régénérée toutes les 6 h |
| **Manifeste Scoop** | `bucket/gofact.json` dans ce dépôt, en 1.1.0 | idem |
| **Paquets Linux + AUR** | `nfpms` et `aurs` dans `.goreleaser.yaml` (`gofact-bin`) | sur tag |
| **Topics GitHub** | 18 topics posés, dont `factur-x`, `en16931`, `mcp-server`, `france` | manuel |
| **Description GitHub** | « Facturation électronique française en local… » | manuel |
| **Documentation** | site MkDocs sur `https://kolapsis.github.io/gofact/`, déployé par `docs.yml` | à chaque push sur `main` touchant `docs/` |

---

## À faire — par ratio impact/effort

Chaque ligne dit **qui peut le faire**. La mention « formulaire web » signifie
qu'aucune session ne peut le soumettre seule : le contenu est prêt, la
soumission demande un humain.

| # | Canal | Où | Qui | Charge |
| --- | --- | --- | --- | --- |
| 1 | **Champ Website du dépôt** | réglages GitHub | Benjamin | 10 s — toujours vide |
| 2 | **Discussions GitHub** | réglages GitHub | Benjamin | 30 s — le public cible n'ouvre pas d'issue |
| 3 | **Répertoire de plugins Claude Code** | <https://clau.de/plugin-directory-submission> | Benjamin, après `claude plugin validate` | 1 h |
| 4 | **mcp.so** | <https://mcp.so/submit?type=server> | formulaire web | 20 min |
| 5 | **PulseMCP** | <https://www.pulsemcp.com/> bouton Submit | formulaire web | 20 min |
| 6 | **Glama** | indexé automatiquement, à réclamer | formulaire web | 15 min |
| 7 | **awesome-mcp-servers** (punkpeye) | PR sur le dépôt | une session avec accès à ce dépôt | 30 min |
| 8 | **LinuxFr — journal puis dépêche** | <https://linuxfr.org/> | Benjamin (compte requis) | brouillon prêt : `handoff/comms/queue/2026-09-linuxfr-journal-pdfa3.md` |
| 9 | **Framalibre** | <https://framalibre.org/contribuer/> | formulaire web | 45 min — audience libriste francophone, sous-estimée |
| 10 | **AlternativeTo** | <https://alternativeto.net/> | formulaire web | 30 min — SEO durable |
| 11 | **LibHunt / go.libhunt.com** | <https://www.libhunt.com/site/project_submit> | formulaire web | 15 min |
| 12 | **selfh.st/apps** | <https://selfh.st/apps/> | formulaire web | 30 min |
| 13 | **Show HN** | <https://news.ycombinator.com/showhn.html> | Benjamin | **une seule fois**, sur un vrai jalon |
| 14 | **r/selfhosted, r/golang, r/france** | — | Benjamin | exige un jalon, sinon contre-productif |

---

## Écarté, avec la raison

Ne pas y revenir sans que la condition ait changé.

| Canal | Pourquoi | Réexaminer |
| --- | --- | --- |
| **awesome-go** | Exige 5 mois d'historique, ≥ 80 % de couverture, Go Report A | février 2027 |
| **awesome-selfhosted** | La liste vise des services réseau, pas des outils en ligne de commande | jamais, sauf changement de leur charte |
| **Homebrew core** | Seuil de notoriété : ≥ 75 étoiles | à 75 étoiles |
| **Smithery** | Orienté TypeScript hébergé ; un binaire Go stdio y est mal ajusté | jamais |
| **SILL / code.gouv.fr** | Exige un référent agent public utilisateur | si un client public apparaît |
| **Annuaires de plateformes agréées** | gofact n'est **pas** une plateforme agréée — y figurer contredirait nos propres garde-fous | jamais |
| **Product Hunt** | Saturé, cycle de 24 h, audience peu française | jamais pour ce produit |
| **FNFE-MPE** | Cotisation non publique, lent | quand il y aura des utilisateurs à représenter |

---

## Règle qui a coûté cher

Le 5 septembre, deux sessions ont construit le même dispositif de publication au
registre MCP en parallèle. Le travail distant a été jeté, celui de la session
locale conservé.

**Avant d'ouvrir un chantier de distribution : lire ce fichier, puis vérifier
l'état réel** — `git log --oneline -30 origin/main`, la liste des workflows, et
la preuve vérifiable de la ligne concernée. Le statut d'un job CI ne prouve
rien : c'est le registre, la page ou le fichier qui fait foi.

Toute action livrée s'inscrit ici dans le même commit que le travail.
