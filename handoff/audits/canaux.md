# gofact — audit des canaux de distribution et de la concurrence open source

Analyste distribution · 2026-09-04
Cible : `github.com/kOlapsis/gofact` — binaire Go AGPL-3.0, Factur-X (PDF/A-3 + CII EN 16931), serveur MCP, plugin Claude Code.

---

## 0. État des lieux — ce qui conditionne tout le reste

Constaté sur le dépôt le 2026-09-04 :

| Fait | Conséquence directe |
| --- | --- |
| Premier commit **2026-08-31**, 22 commits | Le dépôt a **4 jours**. Toutes les listes « awesome » à critère d'ancienneté sont hors d'atteinte avant 2027. |
| **5 étoiles**, 0 fork | Homebrew core (≥75 ★ / 30 forks / 30 watchers) hors d'atteinte. Les annuaires qui trient par étoiles vous placeront en bas. |
| **Aucune release, aucun tag** | Bloquant dur. `install.sh` pointe vers une release qui n'existe pas : **le parcours d'installation annoncé dans le README est cassé**. Bloque aussi le registre MCP (mcpb), AUR, Homebrew tap, Scoop, awesome-go. |
| **Aucun topic GitHub**, pas de homepage renseignée | Perte sèche sur la recherche GitHub (`topic:factur-x` a 20 dépôts : vous n'y êtes pas). |
| Description GitHub = « MCP e-facturation service » | Ne contient ni `Factur-X`, ni `EN 16931`, ni `France`. Illisible pour l'audience cible. |
| `.goreleaser.yaml` prêt, non déclenché | Le travail est fait à 90 %, il manque `git tag v1.1.0 && goreleaser release`. |
| 7 fichiers de test / 41 fichiers Go | Couverture probablement < 80 % → awesome-go refusé même après 5 mois. |

**Règle de lecture du reste du document** : tout canal marqué « bloqué par T0 » attend uniquement la publication d'une release. C'est une heure de travail qui débloque une dizaine de canaux. C'est, de très loin, le meilleur ratio du document.

Un second point vaut d'être dit franchement : **votre plafond d'audience à court terme n'est pas la distribution, c'est la maturité perçue**. Un dépôt de 4 jours à 5 étoiles soumis à dix annuaires simultanément produit dix fiches vides. Les canaux ci-dessous sont donc séquencés, pas empilés.

---

## 1. Tableau maître — classé par ratio impact / effort décroissant

Légende effort : ⏱ = temps de mise en œuvre. Retour : appréciation honnête à 3 mois pour **ce** projet, à **ce** stade.

| # | Canal | Coût | Délai | ⏱ Effort | Retour attendu (3 mois) |
| --- | --- | --- | --- | --- | --- |
| 1 | **Release GitHub v1.1.0 (goreleaser)** | 0 € | immédiat | 1 h | Débloque 10 canaux. Répare le README. **Indispensable.** |
| 2 | **Topics + description + homepage GitHub** | 0 € | immédiat | 15 min | 10-40 visites/mois durables, gratuit, permanent |
| 3 | **Registre officiel MCP** | 0 € | < 1 h | 2-3 h | Le seul registre qui alimente les clients MCP par API. Fort effet de levier |
| 4 | **Répertoire de plugins Claude Code (Anthropic)** | 0 € | qq jours | 1 h | Distribution intégrée à Claude Code. Le canal le plus qualifié pour le pilier C |
| 5 | **mcp.so** | 0 € (payant optionnel) | 1-7 j | 20 min | Trafic SEO modeste mais permanent |
| 6 | **PulseMCP** | 0 € | qq jours | 20 min | Idem, avec compteur de visiteurs public utile comme métrique |
| 7 | **Glama** (auto-indexé, à réclamer) | 0 € | auto | 15 min | Passif. Indexation automatique déjà probable |
| 8 | **awesome-mcp-servers (punkpeye)** | 0 € | 1-14 j | 30 min | ~70 k ★ sur la liste, mais dilution extrême. Faible mais gratuit |
| 9 | **Framalibre** | 0 € | 1-3 sem. | 45 min | Audience libriste **francophone** exactement ciblée. Sous-estimé |
| 10 | **AlternativeTo** | 0 € | 2-7 j | 30 min | SEO durable sur « alternative à [SaaS de facturation] » |
| 11 | **LibHunt / go.libhunt.com** | 0 € | jours | 15 min | Backlink SEO + indexation Go. Passif |
| 12 | **Tap Homebrew perso + bucket Scoop perso** | 0 € | immédiat | 2 h | Supprime la friction d'installation macOS/Windows. Utile même à 0 utilisateur |
| 13 | **AUR (`gofact-bin`)** | 0 € | immédiat | 1 h | Aucun critère de notoriété. Public self-hoster exactement ciblé |
| 14 | **selfh.st/apps + Self-Host Weekly** | 0 € | 1-2 sem. | 30 min | Newsletter self-hosting très lue. Bon rapport |
| 15 | **Annuaires de skills/plugins Claude (aitmpl, awesome-claude-skills, claudepluginhub)** | 0 € | jours | 1-2 h | Volume faible par annuaire, cumul correct |
| 16 | **LinuxFr.org — dépêche** | 0 € | 2-7 j (modéré) | 4-6 h | **Le meilleur canal FR non-LinkedIn.** Audience libriste FR, effet durable |
| 17 | **r/selfhosted, r/golang** | 0 € | immédiat | 2 h | Pic ponctuel. Exige un vrai jalon, sinon contre-productif |
| 18 | **Show HN** | 0 € | immédiat | 4 h | Loterie à forte variance. À jouer **une seule fois**, sur la v1 avec release |
| 19 | **Compta Online / communautés experts-comptables** | 0 € | continu | 3-5 h/mois | Autorité, prescription. Lent mais c'est là que se décide l'équipement des TPE |
| 20 | **Comparateurs FR de facturation électronique** | 0 € à ~€€ | 2-6 sem. | 2 h | Trafic très qualifié, mais la plupart ne référencent que des PA agréées |
| 21 | **Product Hunt** | 0 € | 24 h | 8-12 h | Saturé, cycle de 24 h, audience peu française. Faible pour ce produit |
| 22 | **nixpkgs** | 0 € | 1-6 sem. | 3-4 h | Public restreint mais fidèle. Pas de seuil de notoriété |
| 23 | **awesome-go** | 0 € | ≥ fév. 2027 | 8-15 h | **Critères non remplis** (5 mois d'historique, ≥80 % couverture, release taggée, Go Report A). Cible T+6 mois |
| 24 | **awesome-selfhosted** | 0 € | ≥ jan. 2027 | 2 h | **Probablement inéligible** : la liste vise des services réseau, pas des CLI locales |
| 25 | **Homebrew core** | 0 € | ≥ 75 ★ | 2 h | Bloqué par la notoriété. Reporter |
| 26 | **FNFE-MPE (adhésion)** | cotisation annuelle (barème non public) | 1-3 mois | 10 h+ | Autorité et accès aux groupes de travail Factur-X. Cher, lent, mais c'est **la** légitimité du domaine |
| 27 | **SILL / code.gouv.fr** | 0 € | — | — | **Inéligible** : exige un référent agent public utilisant le logiciel |
| 28 | **Smithery** | 0 € | — | — | **Mauvais ajustement** : orienté TypeScript/hébergé. Un binaire Go stdio y est un citoyen de seconde zone |
| 29 | **Annuaires PA / PDP (choisirsaplateforme.fr, comparateur-efacturation…)** | — | — | — | **À ne pas viser.** gofact n'est pas une plateforme agréée ; y figurer serait faux et contraire à vos propres garde-fous |
| 30 | **apt / Debian** | 0 € | 6-18 mois | 30 h+ | Hors sujet à ce stade |

---

## 2. Registres et annuaires MCP — procédures exactes

### 2.1 Registre officiel `registry.modelcontextprotocol.io` — **priorité 1**

Le seul registre alimenté par API et consommé par les clients (community-owned, soutenu par Anthropic, GitHub, PulseMCP, Microsoft). API gelée en v0.1 depuis octobre 2025 : stable.

**Procédure exacte**

```sh
# 1. installer le publisher
curl -L "https://github.com/modelcontextprotocol/registry/releases/latest/download/mcp-publisher_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz" \
  | tar xz mcp-publisher && sudo mv mcp-publisher /usr/local/bin/
# (macOS : brew install mcp-publisher)

# 2. générer le manifeste
mcp-publisher init          # crée server.json

# 3. authentification — namespace io.github.kolapsis/*
mcp-publisher login github  # device flow : https://github.com/login/device

# 4. publier
mcp-publisher publish

# 5. vérifier
curl "https://registry.modelcontextprotocol.io/v0.1/servers?search=io.github.kolapsis/gofact"
```

**Le point technique qui vous concerne.** Les `registryType` acceptés sont `npm`, `pypi`, `cargo`, `nuget`, `oci`, `mcpb`. gofact n'est dans aucun registre de paquets → **le chemin est `mcpb`** : un bundle MCP hébergé sur **GitHub Releases** (fournisseur explicitement autorisé), déclaré ainsi :

```json
{
  "registryType": "mcpb",
  "identifier": "https://github.com/kOlapsis/gofact/releases/download/v1.1.0/gofact-mcp-linux-amd64.mcpb",
  "fileSha256": "…"
}
```

Deux contraintes à retenir : l'URL du `.mcpb` **doit contenir la chaîne `mcp`** (l'extension `.mcpb` suffit), et il faut **un `.mcpb` par plateforme/architecture**. Le `.mcpb` embarque le binaire — ce qui colle parfaitement à un binaire Go statique sans dépendance.

**Action concrète** : ajouter une étape `mcpb` au `.goreleaser.yaml` (6 artefacts : linux/darwin/windows × amd64/arm64) + un job CI `mcp-publisher publish` déclenché sur tag (auth GitHub OIDC, pas de secret à stocker).
**Coût** 0 € · **délai** publication immédiate · **effort** 2-3 h.
**Retour honnête** : peu de trafic *direct*, mais c'est la source amont que mcp.so, PulseMCP, Glama et les clients MCP moissonnent. C'est un multiplicateur, pas un canal.

### 2.2 mcp.so

- Formulaire : **https://mcp.so/submit?type=server** (aussi : ouverture d'issue sur `github.com/chatmcp/mcpso`).
- Champs : nom, description, fonctionnalités, informations de connexion.
- Gratuit avec revue manuelle ; offre « Premium » payante (lien dofollow, revue accélérée, mise en avant). **Ne pas payer** à ce stade.
- Délai 1-7 jours · effort 20 min.
- Retour : quelques dizaines de visites/mois, principalement SEO. Correct pour l'effort.

### 2.3 PulseMCP

- Bouton « Submit » dans la barre de navigation de **https://www.pulsemcp.com/**.
- Particularité utile : PulseMCP publie une **estimation de visiteurs hebdomadaires par serveur** — c'est une métrique externe gratuite pour votre `handoff/comms/METRICS.md`.
- Gratuit · quelques jours · 20 min.

### 2.4 Glama

- **https://glama.ai/mcp/servers** — indexation **automatique** des dépôts open source depuis GitHub (~37 000 serveurs mi-2026, plus de 80 000 annoncés désormais). Glama indexe chaque outil, schéma et annotation.
- Vous n'avez donc rien à soumettre : il faut **réclamer** la fiche. Pour un connecteur avec domaine, la revendication passe par un fichier `/.well-known/glama.json` sur le domaine.
- Effort 15 min · retour passif.
- **Conséquence pratique** : vos descriptions d'outils MCP (`create_invoice`, `send_invoice`…) seront affichées publiquement telles quelles. Elles doivent être rédigées comme de la documentation, pas comme des notes internes.

### 2.5 `awesome-mcp-servers` (punkpeye)

- Dépôt : **https://github.com/punkpeye/awesome-mcp-servers** · guide : `/blob/main/CONTRIBUTING.md`.
- Procédure : fork → une ligne, une PR → ordre **alphabétique** dans la bonne catégorie → format `- [nom](url) - Description.`
- Particularité : un agent automatisé peut ajouter `🤖🤖🤖` en fin de titre de PR pour un traitement accéléré.
- Délai 1-14 jours · effort 30 min.
- **Honnêteté** : la liste compte des milliers d'entrées. L'impact marginal est faible ; le coût aussi.

### 2.6 Smithery — **déconseillé**

**https://smithery.ai/** (6 000+ serveurs). Le flux de déploiement est bâti autour du runtime TypeScript officiel (`smithery.yaml`, `createServer`, `configSchema`, Node 18+, CLI npm) ; les autres langages passent par un conteneur Docker personnalisé. Il existe un mode « Local servers (beta) » via bundle MCP, mais un binaire Go stdio y reste un cas limite.
**Verdict** : effort 6-10 h pour une fiche mal ajustée. À reconsidérer si et seulement si Smithery généralise l'installation par `.mcpb`.

### 2.7 Autres, marginaux

`mcp.directory/submit`, `mcpserve.com/submit`, `mcpservers.org`. Gratuits, 10 min chacun, trafic anecdotique. À faire en une seule session « annuaires longue traîne », pas avant.

---

## 3. Annuaires de plugins Claude Code et marketplaces de skills

### 3.1 Répertoire officiel Anthropic — **priorité 2**

C'est le canal le mieux ajusté à gofact après le registre MCP : le dépôt **est déjà** un plugin (`.claude-plugin/marketplace.json` + `plugin.json` + skill `creer-facture` + serveur MCP déclaré).

- **Formulaire de soumission : https://clau.de/plugin-directory-submission**
  (miroirs mentionnés : `claude.ai/settings/plugins/submit`, `platform.claude.com/plugins/submit`)
- Le dépôt **`anthropics/claude-plugins-community`** est un **miroir en lecture seule** synchronisé chaque nuit depuis le pipeline de revue interne : **une PR directe n'aboutira pas**.
- `anthropics/claude-plugins-official` est curé par Anthropic à sa discrétion, **sans procédure de candidature**.
- Prérequis : dépôt **public** (le closed source est refusé), `claude plugin validate` passant, manifeste conforme, tous les fichiers référencés présents, le plugin n'accède pas à des ressources hors de son répertoire, instructions de skill claires, README expliquant installation et usage.
- Revue : automatisée (validation + criblage malware / collecte d'identifiants) puis humaine. **Quelques jours.** Un badge « Anthropic Verified » existe pour les plugins ayant passé une revue supplémentaire.
- Une fois approuvé, le plugin est épinglé sur un SHA et **la CI avance l'épingle automatiquement** : vos mises à jour sont diffusées sans nouvelle revue.

**Point de vigilance sérieux.** Votre `plugin.json` lance `${CLAUDE_PLUGIN_ROOT}/scripts/mcp-launcher.sh`, qui **compile le binaire au premier usage** et exige donc une chaîne Go. C'est un obstacle à la revue (accès hors répertoire, dépendance non déclarée) autant qu'à l'adoption. **À corriger avant soumission** : le lanceur doit télécharger le binaire de release correspondant à la plateforme (avec vérification du SHA-256), et ne se rabattre sur `go build` qu'en dernier recours. Encore un point qui dépend de T0.

**Retour attendu** : le meilleur du document en qualité d'audience. Ce sont des utilisateurs de Claude Code, donc techniques, donc l'audience 2 et 3 de votre `STRATEGY.md` réunies.

### 3.2 Listes et annuaires communautaires

| Annuaire | URL | Procédure | Effort | Retour |
| --- | --- | --- | --- | --- |
| `ComposioHQ/awesome-claude-skills` | https://github.com/ComposioHQ/awesome-claude-skills — voir `CONTRIBUTING.md` | Fork → répertoire de skill → `SKILL.md` → PR. Exige un **cas d'usage réel** (pas hypothétique), l'absence de doublon, une documentation et un test sur Claude.ai / Claude Code | 1 h | Liste la plus complète du segment. Correct |
| `composio-community/awesome-claude-plugins` | https://github.com/composio-community/awesome-claude-plugins | PR | 20 min | Faible mais gratuit |
| `travisvn/awesome-claude-skills` | https://github.com/travisvn/awesome-claude-skills | PR | 20 min | Alternative orientée Claude Code |
| `davila7/claude-code-templates` / aitmpl.com | https://github.com/davila7/claude-code-templates (guide de contribution sur docs.aitmpl.com) | PR d'un composant (agent / commande / skill / MCP) | 1-2 h | Catalogue + CLI, 1000+ composants, trafic réel |
| claudepluginhub.com | https://www.claudepluginhub.com/ | Soumission web, non affilié à Anthropic | 20 min | Longue traîne |
| awesome-skills.com | https://awesome-skills.com/ | Soumission web | 20 min | Longue traîne |
| claudecodecommands.directory | https://claudecodecommands.directory/ | Soumission web | 20 min | Longue traîne |

**Argument différenciant à mettre en avant partout dans ce segment** — et il est réellement inhabituel : *la numérotation légale n'est pas confiée au modèle*. Le serveur numérote sous verrou, en transaction ; l'IA n'écrit qu'un jeton. Dans un écosystème saturé de skills qui délèguent tout au LLM, c'est le genre de phrase qui fait cliquer un ingénieur.

---

## 4. Listes « awesome-* » — verdict par liste

### 4.1 awesome-go — **non éligible avant ~février 2027**

**https://github.com/avelino/awesome-go** · `CONTRIBUTING.md`.

Critères, cités : au moins **5 mois d'historique depuis le premier commit** ; couverture de tests **≥ 80 %** (≥ 90 % pour les paquets de données) ; **au moins une release versionnée `vX.Y.Z`** ; README + commentaires godoc sur les API publiques ; licence open source ; commits récents et réguliers ; aucune issue non traitée de plus de 6 mois. La PR doit fournir les liens **pkg.go.dev**, **Go Report Card (note A-, A ou A+)** et un rapport de couverture (Codecov/Coveralls). Un seul ajout par PR, ordre alphabétique, format `- [nom](url) - Description.`

État gofact : 4 jours d'historique, 0 release, 7 fichiers de test pour 41 fichiers Go. **Deux des trois blocages sont mécaniques (temps, release), le troisième est du travail réel (couverture).**

**Recommandation** : traiter la couverture ≥ 80 % comme un objectif de qualité — pas comme un ticket d'entrée. Elle sert d'abord vos propres arguments de conformité vérifiable (pilier B). Viser la PR awesome-go en **février 2027**, avec Go Report Card A comme jalon intermédiaire public.

### 4.2 awesome-selfhosted — **probablement inéligible, ne pas insister**

**https://github.com/awesome-selfhosted/awesome-selfhosted-data** (les entrées sont des fichiers `software/nom.yml`, kebab-case ; PR « add My Awesome software »).

Critère d'ancienneté : « première release publiée il y a **plus de 4 mois** ». Mais surtout, la liste vise « les **services réseau et applications web** auto-hébergeables » et exclut explicitement « les applications de bureau, mobiles ou **en ligne de commande** nécessitant un programme serveur ou de synchronisation séparé ».

gofact est une CLI locale + un serveur MCP stdio. **Il n'expose aucun service réseau.** Une soumission a de fortes chances d'être refusée sur ce motif. Honnêtement : c'est un canal que la fiche produit fait miroiter et que la nature du produit interdit.

**Contournement légitime** : viser **selfh.st/apps** (§6.4) qui accepte un périmètre plus large, plutôt que de forcer awesome-selfhosted.

### 4.3 awesome-invoicing — **n'existe pas**

Aucune liste « awesome-invoicing » de référence. Les points d'entrée réels sont les **topics GitHub** : `invoicing`, `invoice`, `e-invoice`, `factur-x`, `zugferd`, `en16931`, `peppol`. C'est gratuit, instantané, et vous n'y êtes pas.

**Action T0.2** : ajouter ces topics au dépôt. `topic:factur-x` ne compte que ~20 dépôts — c'est une niche où l'on est visible immédiatement.

### 4.4 Listes francophones

| Liste | URL | Verdict |
| --- | --- | --- |
| `JMousqueton/awesome-francais` — « outils connus avec leurs alternatives françaises » | https://github.com/JMousqueton/awesome-francais | **Ajustement conceptuel parfait** (gofact = alternative FR à un SaaS de facturation), mais **6 étoiles** : audience quasi nulle. À faire par principe (20 min), pas par calcul. Une catégorie « facturation / comptabilité » n'existe pas encore — la créer est un petit acte de fondation |
| `codegouvfr/awesome-codegouvfr` | https://github.com/codegouvfr/awesome-codegouvfr | Réservé aux codes financés par la sphère publique. **Inéligible** |
| `stephrobert/awesome-french-devops` | https://github.com/stephrobert/awesome-french-devops | Hors sujet (formation DevOps) |
| `alexislefebvre/awesome-french` | https://github.com/alexislefebvre/awesome-french | Hors sujet (traductions) |
| `e-invoice-be/awesome-peppol` | https://github.com/e-invoice-be/awesome-peppol | Pertinent sur le fond (EN 16931 / Peppol). Petite liste, PR simple. À faire |

---

## 5. Écosystème Factur-X et facturation électronique française

C'est le segment où le retour est le plus lent **et** le plus solide. Votre `STRATEGY.md` identifie l'audience 4 (experts-comptables, prescripteurs) comme « celle qui fait l'autorité » — c'est exactement ici.

### 5.1 FNFE-MPE — l'autorité du domaine

**https://fnfe-mpe.org/** · adhésion : **https://fnfe-mpe.org/membres/adhesion/**

Association créée en 2012, co-mainteneuse du standard **Factur-X** avec le FeRD allemand (ZUGFeRD) depuis octobre 2015. L'adhésion donne accès aux séances plénières, aux **groupes de travail** (normes, interopérabilité, bonnes pratiques, formation) et au statut de partenaire des **Journées de la Facture Électronique** (édition 2026 : 5-6 mai, ~2 000 participants, 50+ ateliers).

- **Coût** : cotisation annuelle selon le collège et la taille de la structure — le barème n'est pas public, il s'obtient via le formulaire d'adhésion. Pour un indépendant, prévoir l'ordre de grandeur d'une cotisation professionnelle, pas d'un abonnement SaaS.
- **Délai** : 1-3 mois (cycle associatif).
- **Effort** : 10 h+ sur l'année si l'on participe réellement à un groupe de travail.
- **Retour honnête** : **zéro trafic direct**. Mais c'est le seul canal qui transforme « un binaire sur GitHub » en « un acteur du sujet ». Un producteur d'outil Factur-X open source dans un groupe de travail Factur-X, c'est une position que personne d'autre n'occupe aujourd'hui.
- **Recommandation** : **ne pas adhérer maintenant.** Adhérer quand vous aurez (a) une release, (b) une CI publique passant le Schematron officiel, (c) trois retours d'utilisateurs. Sinon vous payez pour une chaise. Cible : décembre 2026 – janvier 2027, en visant les JFE 2027 comme échéance de visibilité.
- **Étape gratuite immédiate** : suivre les publications FNFE-MPE et citer précisément le standard dans votre documentation. La reconnaissance vient de la justesse technique avant de venir de la carte de membre.

### 5.2 Communautés d'experts-comptables

| Lieu | URL | Nature | Recommandation |
| --- | --- | --- | --- |
| **Compta Online** | https://www.compta-online.com/ | Média + forums, 21 ans d'existence, cœur de cible prescripteur. Dossiers actifs sur la réforme et sur ComptaTech 2026 | **Le meilleur du segment.** Participer aux fils sur la réforme en apportant de l'exactitude technique (différence PA/PPF, ce que garantit vraiment un PDF/A-3), signature discrète. Jamais de post promotionnel |
| **Ordre des experts-comptables Paris IdF** | https://www.oec-paris.fr/facture-electronique/ | Publie les outils « E-Factu » (diagnostic de conformité) et « Quelle PA » (comparateur de plateformes agréées) | gofact n'étant pas une PA, il n'entre pas dans « Quelle PA ». En revanche, l'OEC est un interlocuteur pour de la pédagogie. Effort élevé, retour incertain |
| **ComptaTech** | via Compta Online | Événement annuel, facturation électronique + IA à l'agenda 2026 | À surveiller pour 2027. Un retour d'expérience « facturation pilotée par IA, numérotation garantie hors modèle » y serait audible |

### 5.3 Comparateurs et annuaires français de la réforme

| Site | URL | Peut-on s'y faire référencer ? |
| --- | --- | --- |
| ChoisirSaPlateforme (ex-ChoisirSaPDP) | https://choisirsaplateforme.fr/ · contact@choisirsapdp.fr | Réservé aux **plateformes agréées**. gofact n'y a pas sa place. Contact possible pour une fiche « outil amont », sans garantie |
| comparatif-facture-electronique.fr | https://www.comparatif-facture-electronique.fr/ | Publie des avis d'outils **non-PA** (fiche Dolibarr existante). **Un e-mail de présentation est justifié** |
| facture-obligatoire.fr | https://facture-obligatoire.fr/logiciels/ | Section « logiciels » incluant l'open source (fiche Dolibarr). **Cible réaliste** |
| comparateur-efacturation.fr | https://comparateur-efacturation.fr/logiciel-facturation | Comparatif « 148 solutions agréées » — orienté PA |
| independant.io | https://independant.io/logiciel-facturation-gratuit/ | Comparatif « logiciels de facturation gratuits », audience indépendants **exactement** la vôtre. **Prise de contact recommandée** |

**Garde-fou impératif**, cohérent avec votre `STRATEGY.md` §2 : dans toute prise de contact, la phrase « gofact n'est pas une plateforme agréée » doit figurer **avant** toute autre. Un éditeur qui se fait référencer par ambiguïté dans un comparatif de PA se grille définitivement auprès des prescripteurs. C'est aussi, paradoxalement, l'accroche : *le seul outil de la liste qui commence par dire ce qu'il ne fait pas*.

---

## 6. Communautés où le public cible pose réellement ses questions

### 6.1 LinuxFr.org — **le meilleur canal FR après LinkedIn**

- **Dépêche** (modérée a priori, mise en avant) : **https://linuxfr.org/news/nouveau**
- **Journal** (publication directe, sans validation préalable) : via https://linuxfr.org/proposer-un-contenu
- Règles : une dépêche doit être **rédigée et étoffée**, poser le contexte, résumer, fournir des liens. Les dépêches trop courtes ou mal rédigées sont rejetées, le plagiat aussi, et « les dépêches purement commerciales et publicitaires seront rejetées ».
- **Effort réel** : 4-6 h d'écriture. C'est un article, pas une annonce.
- **Angle qui passera la modération** : ni « voici mon outil », mais « **PDF/A-3 sans Java ni Ghostscript : ce qu'il faut vraiment écrire dans un PDF pour qu'il soit un Factur-X** », avec gofact en illustration et en fin de texte. Votre argument de l'embarquement verbatim et des champs BT-34/BT-49 perdus par les assembleurs qui re-sérialisent est exactement le type de contenu que ce lectorat récompense.
- **Retour** : plusieurs milliers de lecteurs francophones libristes, archivés et indexés durablement. Le meilleur ratio du bloc « communautés ».
- **Tactique** : publier d'abord en **journal** (sans risque), et si l'accueil est bon, proposer une dépêche sur un angle différent.

### 6.2 Reddit

| Sub | Réalité | Recommandation |
| --- | --- | --- |
| **r/selfhosted** | Très actif, tolère les auto-annonces si le projet est réel et le post honnête (mention explicite « je suis l'auteur ») | **Oui**, après release. Angle : local-first, zéro télémétrie, `GOFACT_OFFLINE=1` |
| **r/golang** | Tolère les partages de projets ; exige du contenu technique et sanctionne le marketing | **Oui**, angle « PDF/A-3 en Go pur avec pdfcpu, ICC sRGB embarqué, zéro CGO » |
| **r/france** | Volume énorme, mais l'auto-promotion y est mal reçue et la modération stricte | **Non** en post dédié. **Oui** en réponse utile dans les fils sur la réforme |
| **r/vosfinances** (~432 k membres) | Communauté finances FR très active, questions fiscales/entrepreneuriales fréquentes | **Oui**, en réponse pédagogique uniquement, jamais en post promotionnel |
| **r/entrepreneur** (5,3 M, anglophone) | Hors cible (public US, sujet FR) | Non |
| Subs FR spécialisés freelances/auto-entrepreneurs | Existants mais petits et fragmentés | Faible priorité |

**Règle**, reprise de votre `STRATEGY.md` : maximum une fois par mois, sur un jalon objectif. Un post Reddit promotionnel sans substance coûte durablement plus qu'il ne rapporte.

### 6.3 Hacker News — Show HN

- **https://news.ycombinator.com/showhn.html** puis « submit ».
- Règles 2026 : un Show HN doit être **quelque chose que l'on peut essayer** (donc : release binaire obligatoire — encore T0). **Ne pas solliciter de votes** auprès de ses proches ou de ses utilisateurs. Les commentaires générés ou retouchés par IA sont **explicitement interdits** par les règles 2026.
- Format de titre efficace : `Show HN: gofact – Compliant French e-invoicing (Factur-X) as a single Go binary`.
- Fenêtre critique : les 30-60 premières minutes ; il faut ~30-50 votes dans l'heure pour espérer la une.
- **Effort** 4 h (préparer README anglais, GIF de démo, et être disponible 6 h pour répondre).
- **Retour honnête** : forte variance. Une première page = 3 000-15 000 visites et souvent 200-800 étoiles ; un échec = 30 visites. Le sujet (réglementation française) plafonne l'intérêt international, mais l'angle technique (PDF/A-3 pur Go, MCP, numérotation légale hors LLM) est exactement ce que HN aime.
- **À jouer une seule fois**, sur la v1, pas avant. Un second Show HN sur le même produit demande d'attendre longtemps et de montrer autre chose.

### 6.4 Self-hosting

- **selfh.st/apps** : https://selfh.st/apps/ — annuaire vivant avec filtres et indicateurs d'activité ; ajouts via les dépôts de https://github.com/selfhst.
- **Self-Host Weekly** : https://selfh.st/weekly/ — newsletter hebdomadaire très lue par le public self-hosté ; une nouvelle release y est régulièrement reprise.
- Effort 30 min · gratuit · bon rapport.

### 6.5 Discord / Slack francophones de freelances

| Communauté | Taille | Accès |
| --- | --- | --- |
| **Slack Freelance France** | 1 000+ freelances | https://slofile.com/slack/freelancefrance |
| **La Collab** | 3 500+ experts indépendants | https://lacollab.com/rejoindre-le-collectif-la-collab/ |
| Collectifs divers | fragmentés | Recensés sur https://www.lafabriquedunet.fr/agences/tendances/collectifs-freelances |

**Retour honnête** : ces espaces convertissent mal à froid et sanctionnent durement la promotion. Leur valeur réelle est **la recherche utilisateur** : y poser des questions (« comment comptez-vous gérer septembre 2027 ? ») pour nourrir la documentation, et laisser l'outil venir dans la conversation. Traiter comme un canal d'écoute, pas de diffusion. Aligné sur votre garde-fou « pas de sollicitation automatisée en message privé ».

### 6.6 Médias tech francophones

- **Korben.info** — https://korben.info/ · couvre régulièrement les outils libres et auto-hébergés, très forte audience FR. Pas de formulaire public de soumission : passer par le contact du site ou les réseaux. Faible probabilité, coût quasi nul, gain élevé si ça passe.
- **Next / LinuxFr / Toolinux** — même logique, effort de rédaction requis.

---

## 7. Gestionnaires de paquets — conditions d'admission réelles

| Gestionnaire | Condition réelle | Statut gofact | Recommandation |
| --- | --- | --- | --- |
| **Homebrew core** (`homebrew-core`) | Notoriété exigée : **≥ 30 forks OU ≥ 30 watchers OU ≥ 75 étoiles**. Justification assumée : « ce n'est pas une question de qualité », c'est pour éviter les formules abandonnées. Seuils **plus élevés pour l'auto-soumission** de casks (90/90/225) | 5 ★, 0 fork | **Bloqué.** Docs : https://docs.brew.sh/Acceptable-Formulae |
| **Tap Homebrew personnel** | **Aucune condition.** `github.com/kOlapsis/homebrew-tap`, installation `brew install kolapsis/tap/gofact` | Faisable dès T0 | **À faire.** GoReleaser génère la formule automatiquement (`brews:`) |
| **Scoop `Main` bucket** | Critères de popularité pour le bucket principal | Bloqué | Non |
| **Bucket Scoop personnel** | Aucune condition. GoReleaser gère (`scoops:`) | Faisable dès T0 | **À faire** — c'est le chemin Windows le plus propre, et Windows est votre plateforme la plus favorable (Edge préinstallé, aucune dépendance à installer) |
| **AUR** | **Aucun critère de notoriété.** Tout compte AUR peut publier un `PKGBUILD`. Deux paquets classiques : `gofact-bin` (release) et `gofact-git` | Faisable dès T0 | **À faire.** Le public Arch recoupe fortement votre audience 2 |
| **nixpkgs** | Pas de seuil de notoriété. PR sur https://github.com/NixOS/nixpkgs (voir `CONTRIBUTING.md`). Les paquets avec tests automatisés sont fusionnés plus vite ; indiquer les plateformes testées | Faisable après T0 | Oui, mais après les tapes/AUR. 3-4 h, délai 1-6 semaines |
| **apt / Debian** | Processus de parrainage long, exigences de packaging fortes | Hors d'atteinte | Non. Fournir un `.deb` en release (GoReleaser `nfpms:`) couvre 95 % du besoin réel |
| **`go install`** | Automatique via le module proxy | **Déjà acquis** | Documenter `go install github.com/kOlapsis/gofact@latest` en évidence dans le README |
| **pkg.go.dev** | Automatique | Acquis | Soigner les commentaires godoc : c'est aussi un prérequis awesome-go |

**Synthèse packaging** : ajouter à `.goreleaser.yaml` les sections `brews`, `scoops`, `nfpms` (deb/rpm) et une étape `.mcpb`. Une demi-journée de travail, et quatre canaux de distribution deviennent automatiques à chaque tag.

---

## 8. Concurrents open source directs

### 8.1 Bibliothèques et outils Factur-X / EN 16931

| Projet | Langage | Traction | Ce qu'il fait | Position de gofact |
| --- | --- | --- | --- | --- |
| **ZUGFeRD/mustangproject** https://github.com/ZUGFeRD/mustangproject | Java, Apache-2.0 | **457 ★**, 212 forks, v2.26.0 (25 août 2026) | La **référence de facto**. Lecture, écriture, validation, conversion. ZUGFeRD 1/2.5, Factur-X 1, CII XRechnung 3.0.2. Validation Schematron EN 16931 + veraPDF. Bibliothèque **et** CLI. Maven Central | **Vous ne le concurrencez pas — vous l'utilisez** (veraPDF via Mustang en CI). C'est intelligent et honnête. Différence : JVM requise vs binaire unique ; Mustang n'a ni HTML→PDF, ni MCP, ni numérotation légale, ni dépôt PA |
| **angelodlfrtr/go-invoice-generator** (+ `/facturx`) https://github.com/angelodlfrtr/go-invoice-generator | **Go**, Apache-2.0 | **146 ★** | Génération de facture PDF **et** sous-paquet `facturx` : CII XML embarqué, **5 profils**, `/AFRelationship`, OutputIntent ICC sRGB, XMP `pdfaid`/`pdfaExtension`/`fx:`, **pur Go sans dépendance**, validé mustang-cli + veraPDF | ⚠️ **Le concurrent le plus gênant.** Sur « assembler du Factur-X en Go », **gofact n'apporte rien de neuf** et arrive après, avec 30× moins d'étoiles. Vos différences réelles : bibliothèque vs **produit** (CLI + MCP + organisations + numérotation + dépôt PA), rendu **HTML→PDF fidèle au Ctrl+P** (eux : PDF généré par code), règles EN 16931 **avant** production, et embarquement verbatim explicitement garanti. **À dire clairement plutôt qu'à taire** — un README qui reconnaît l'existence de cette bibliothèque gagne en crédibilité ce qu'il perd en exclusivité |
| **horstoeko/zugferd** (+ `-laravel`, `-visualizer`, `-ublbridge`) | PHP | **434 ★** (+41/34/17) | Écosystème PHP complet : lecture/écriture ZUGFeRD/XRechnung/Factur-X, visualisation, pont vers UBL | Segments disjoints (PHP/Laravel). Aucune concurrence directe |
| **atgp/factur-x** https://github.com/atgp/factur-x | PHP | **153 ★** | Manipulation de PDF Factur-X / ZUGFeRD 2.0 | Disjoint |
| **akretion/factur-x** (PyPI `factur-x`) https://github.com/akretion/factur-x | Python | Référence FR côté Python, PyPI | Génère un Factur-X à partir d'un PDF + un XML conforme ; extrait le XML d'un Factur-X ; valide contre le XSD. Auteur : Alexis de Lattre (Akretion, intégrateur Odoo) | **Comparaison la plus instructive** : même geste (PDF + XML → Factur-X), même pays, mais bibliothèque Python destinée à être intégrée dans Odoo. gofact vise l'utilisateur final, pas l'intégrateur. Fork : `invoice-x/factur-x-ng` |
| **gflohr/e-invoice-eu** https://github.com/gflohr/e-invoice-eu | TypeScript | **216 ★** | Génère des e-factures EN 16931 depuis un **tableur ou du JSON** | Concurrent conceptuel proche de votre pipeline JSON→XML. Ne fait pas de PDF/A-3 fidèle au rendu, ni MCP |
| **OpenIndex/ZUGFeRD-Manager** | Kotlin | **104 ★** | **Application de bureau** de création et validation ZUGFeRD | Le concurrent le plus proche en tant que *produit fini*, mais orienté Allemagne et interface graphique |
| **zfutura/pycheval** | Python | 25 ★ | Génération + parsing Factur-X/ZUGFeRD | Disjoint |
| **jslno/node-zugferd** | TypeScript | 72 ★ | ZUGFeRD/Factur-X + embarquement PDF/A | Disjoint |
| **easybill/zugferd-php**, **easybill/e-invoicing** | PHP | 103 ★ / 16 ★ | SDK ZUGFeRD ; génération XML EN 16931 UBL + CII | Disjoint |
| **Securibox/facturx** | .NET | — | Lecture, création, validation Factur-X | Disjoint |
| **LandrixSoftware/ZUGFeRD-for-Delphi**, `stafyniaksacha/facturx`, `huguesmax/PDF-FacturX` (Perl), `klst-de/e-invoice` (Java) | divers | 16-23 ★ | Implémentations de niche | Disjoints |
| **josemmo/einvoicing** | PHP | — | EN 16931 + Peppol BIS | Disjoint |
| **backoffice-plus/e-invoice-validator**, **ZUGFeRD/ZUV** | XSLT | 20 / 30 ★ | Validateurs | Complémentaires, pas concurrents |

### 8.2 Le concurrent stratégique : `causa-prima-ai/scribo`

**https://github.com/causa-prima-ai/scribo** — 17 ★.

Se décrit comme un outil d'e-facturation **AI-native conforme EN 16931**, avec **MCP, CLI et API**. On décrit sa facture en langage naturel (« facture Acme GmbH 2 400 € pour le design de mai ») et il produit le document structuré. Endpoint MCP **hébergé** (`scribo.causaprima.ai/mcp`), compatible Claude Desktop, Cursor, Cline. Sortie ZUGFeRD = PDF/A-3 hybride + XML CII.

Trois faits décisifs :
1. **Propriétaire** — « © Causa Prima Germany GmbH. All rights reserved. Distributed for use against the public Scribo API; **not open-source**. » Le dépôt GitHub n'est qu'une façade cliente.
2. **SaaS** — backend central, compte requis, « free forever » avec vérification e-mail.
3. **Allemagne et États-Unis en production ; France / Factur-X annoncé « bientôt ».**

**Lecture honnête** : quelqu'un d'autre a eu la même intuition que votre pilier C (facturer en parlant à son IA), l'a exécutée avant, et **arrive sur la France**. Vos avantages sont réels mais périssables : open source AGPL, **local**, aucun compte, aucune télémétrie, France déjà couverte, et surtout la garantie que la numérotation légale ne passe pas par le modèle. Votre avance est une avance de **positionnement**, pas de calendrier. Ce qui plaide pour accélérer les canaux MCP (§2, §3) plutôt que les canaux lents.

### 8.3 ERP et suites open source

| Projet | Traction | Réalité 2026 |
| --- | --- | --- |
| **Dolibarr** https://github.com/Dolibarr/dolibarr | **7 573 ★** | Factur-X **natif depuis la v17**, profil EN 16931 complet en v19. Module e-facturation officiel sur le Dolistore et sur le dépôt communautaire GitHub. Un module tiers `hello-lemon/module-dolibarr-lemonfacturx` existe aussi. Dolibarr **n'est pas** intrinsèquement une plateforme agréée : il doit être raccordé à une PA (certaines sources annoncent une PA adossée à l'écosystème Dolibarr depuis avril 2026 — à vérifier avant toute citation) |
| **Odoo** — `facturx_community` https://apps.odoo.com/apps/modules/19.0/facturx_community | Écosystème massif | Factur-X pour Odoo 19, orienté réforme FR |
| **InvoiceNinja** (10 052 ★), **InvoiceShelf** (1 815 ★), **InvoicePlane** (3 133 ★), **Crater** (8 343 ★), **SolidInvoice**, **Invio** (1 170 ★), **Kimai** (4 957 ★) | Fortes | Facturation self-hosted, **sans conformité Factur-X FR**. Ce sont des concurrents d'usage, pas de conformité |
| **midday-ai/midday** (14 952 ★) | Très forte | Suite freelance (facturation, temps, finances, assistant IA). Pas de Factur-X FR |

**Verdict ERP** : sur « gérer mon activité », Dolibarr et Odoo écrasent gofact et le feront toujours. Ne jamais se positionner là. gofact est un **outil**, pas une suite — et c'est précisément ce que cherche l'indépendant qui refuse d'installer un ERP pour émettre douze factures par an.

### 8.4 SaaS français gratuits (concurrence d'usage réelle)

Facture.net (Codeur.com), Tiime, Shine Facture, Qonto, Kafeo — gratuits, adossés à une PA, zéro friction d'installation. **C'est votre vraie concurrence auprès de l'audience 1**, pas Mustang.

Vous ne gagnerez pas sur la facilité. Vous gagnez sur : pas de compte, pas de données chez un tiers, pas de dépendance à la survie d'une startup, et la portabilité du fichier. Votre raccourci mental (« la plateforme est un tuyau, la facture est un fichier ») est le bon, et il est directement opposable à ces offres.

### 8.5 Là où gofact n'apporte rien de neuf — à assumer

Formulé sans complaisance :

1. **Assembler un PDF/A-3 Factur-X en Go** : `angelodlfrtr/go-invoice-generator/facturx` le fait déjà, validé par les mêmes outils, avec 146 étoiles.
2. **Valider EN 16931** : Mustang est plus complet, plus ancien, plus testé. Vous l'utilisez — c'est le bon choix, mais ce n'est pas un différenciateur.
3. **Couverture fonctionnelle** : **TVA mono-taux**, devises ≠ EUR incomplètes, **une seule PA implémentée** (SuperPDP), profils Factur-X non tous couverts. Mustang et horstoeko couvrent tous les profils, plus UBL. Toute formulation laissant croire à une conformité générale serait fausse.
4. **Facturation en tant que produit métier** : Dolibarr, Odoo, InvoiceNinja sont hors de portée.
5. **Facturation conversationnelle par MCP** : Scribo existe, avec un an d'avance produit — mais propriétaire et SaaS.

**Ce qui reste, et qui est réellement à vous** :
- **Embarquement verbatim** du XML avec préservation explicite de BT-34/BT-49 (adresses de routage PA). Argument technique fort, vérifiable, et que **personne** ne met en avant.
- **Règles EN 16931 appliquées avant production**, avec l'erreur qui nomme la règle (`BR-50`, `BR-CO-15`) et le champ.
- **Numérotation légale sous verrou transactionnel, hors du modèle de langage.** Unique dans l'espace MCP.
- **HTML → PDF fidèle au rendu d'impression** : l'utilisateur garde sa mise en page.
- **Un seul binaire, aucune JVM, aucun Ghostscript, aucun téléchargement au premier lancement**, ICC sRGB embarqué.
- **Local-first, AGPL, français.** Aucun concurrent ne coche ces trois cases à la fois.

Le créneau réel est étroit : *indépendant ou TPE française, techniquement à l'aise, hostile à l'abonnement, utilisatrice d'IA agentique*. Il est étroit — et personne ne l'occupe.

---

## 9. Séquence recommandée

### Semaine 1 — débloquer (≈ 6 h)
1. `git tag v1.1.0` + `goreleaser release --clean` → **release publique**. Vérifier `install.sh` et `install.ps1` de bout en bout.
2. Topics GitHub : `factur-x`, `zugferd`, `en16931`, `e-invoicing`, `invoice`, `pdfa`, `mcp`, `mcp-server`, `claude-code`, `france`, `golang`. Description réécrite. Homepage → `https://kolapsis.github.io/gofact/`.
3. Publier la doc MkDocs sur GitHub Pages (le lien est déjà cité dans le modèle de release GoReleaser — il doit répondre).
4. Corriger `scripts/mcp-launcher.sh` : télécharger le binaire de release (vérification SHA-256), `go build` en dernier recours seulement.

### Semaine 2 — les canaux MCP et Claude (≈ 8 h)
5. Artefacts `.mcpb` dans GoReleaser + `mcp-publisher publish` en CI sur tag → **registre officiel MCP**.
6. Soumission au **répertoire de plugins Claude Code** (`clau.de/plugin-directory-submission`), après `claude plugin validate`.
7. mcp.so, PulseMCP, réclamation Glama, PR `awesome-mcp-servers`.

### Semaine 3 — packaging et annuaires généralistes (≈ 6 h)
8. Tap Homebrew + bucket Scoop + `nfpms` (deb/rpm) dans GoReleaser ; `PKGBUILD` AUR `gofact-bin`.
9. AlternativeTo, LibHunt, selfh.st/apps, Framalibre.
10. `awesome-claude-skills`, `claude-code-templates`, `awesome-peppol`, `awesome-francais`.

### Semaine 4-6 — communautés (≈ 12 h)
11. Journal LinuxFr sur l'angle technique PDF/A-3, puis dépêche si l'accueil est bon.
12. r/selfhosted et r/golang, espacés d'au moins deux semaines.
13. Prise de contact avec independant.io, facture-obligatoire.fr, comparatif-facture-electronique.fr.
14. Présence régulière sur Compta Online, en pédagogie pure.

### Mois 3 — le pic (≈ 8 h)
15. **Show HN** sur la v1.2, avec démo GIF, README anglais soigné et une journée dégagée pour répondre.

### Mois 4-6 — capitalisation
16. Couverture de tests ≥ 80 %, Go Report Card A, godoc complet → **awesome-go** en février 2027.
17. Adhésion **FNFE-MPE** si trois retours d'utilisateurs réels ont été recueillis, en visant les JFE 2027.
18. nixpkgs.
19. Homebrew core dès 75 étoiles.

---

## 10. Estimation globale — honnête

Si tout ce qui précède est exécuté correctement sur six mois, l'ordre de grandeur raisonnable est :

| Source | Étoiles GitHub (cumul 6 mois) | Utilisateurs réels |
| --- | --- | --- |
| Registre MCP + annuaires MCP + plugin Claude Code | 30-80 | 20-60 |
| LinuxFr (dépêche réussie) | 20-60 | 15-40 |
| Show HN (espérance, variance énorme : 0 ou 500) | 0-400 | 0-200 |
| Reddit (r/selfhosted + r/golang) | 20-60 | 10-40 |
| Annuaires passifs (AlternativeTo, LibHunt, selfh.st, Framalibre) | 10-30 | 5-25 |
| Gestionnaires de paquets | 5-15 | 10-40 (conversion, pas découverte) |
| Écosystème FR (comparateurs, Compta Online, FNFE-MPE) | 5-20 | 5-30, **mais la meilleure qualité de contacts** |

Votre cible `STRATEGY.md` de **25 étoiles à S+12** est atteignable par les seuls canaux MCP et le packaging, sans dépendre d'un coup de chance. En revanche, votre indicateur le plus important — « issues et retours d'utilisateurs externes » — ne viendra ni des annuaires ni de Show HN. Il viendra de LinuxFr, de r/selfhosted et de Compta Online : les trois seuls endroits de cette liste où quelqu'un lit *et* répond.

Et il ne viendra de nulle part tant qu'il n'y a pas de release.

---

### Annexe — URLs de soumission, liste plate

```
https://registry.modelcontextprotocol.io                      (via mcp-publisher)
https://github.com/modelcontextprotocol/registry/blob/main/docs/modelcontextprotocol-io/quickstart.mdx
https://clau.de/plugin-directory-submission
https://mcp.so/submit?type=server
https://www.pulsemcp.com/          (bouton « Submit »)
https://glama.ai/mcp/servers       (auto-indexé, réclamation)
https://github.com/punkpeye/awesome-mcp-servers
https://github.com/ComposioHQ/awesome-claude-skills
https://github.com/composio-community/awesome-claude-plugins
https://github.com/davila7/claude-code-templates
https://www.claudepluginhub.com/
https://awesome-skills.com/
https://alternativeto.net          (« Suggest new application », compte requis)
https://www.libhunt.com/site/project_submit
https://framalibre.org/contribuer/
https://selfh.st/apps/  ·  https://github.com/selfhst
https://linuxfr.org/news/nouveau  ·  https://linuxfr.org/proposer-un-contenu
https://news.ycombinator.com/showhn.html
https://www.reddit.com/r/selfhosted/  ·  https://www.reddit.com/r/golang/
https://github.com/avelino/awesome-go                 (bloqué jusqu'à ~fév. 2027)
https://github.com/awesome-selfhosted/awesome-selfhosted-data  (probablement inéligible)
https://github.com/e-invoice-be/awesome-peppol
https://github.com/JMousqueton/awesome-francais
https://docs.brew.sh/Acceptable-Formulae              (core : 75★/30 forks/30 watchers)
https://aur.archlinux.org/                            (aucun seuil)
https://github.com/NixOS/nixpkgs                      (aucun seuil)
https://fnfe-mpe.org/membres/adhesion/                (payant, différer)
https://www.compta-online.com/
https://independant.io/logiciel-facturation-gratuit/
https://facture-obligatoire.fr/logiciels/
https://www.comparatif-facture-electronique.fr/
https://www.producthunt.com/                          (faible priorité)
```
