# FAQ & dépannage

## Général

**Mes données partent-elles quelque part ?**
Non. Pas de serveur gofact, pas de compte, pas de télémétrie. Les seules requêtes
sortantes sont celles que vous déclenchez (annuaire SIRENE, annuaire Peppol, votre
PDP), et `GOFACT_OFFLINE=1` coupe les annuaires. Seule la chaîne recherchée part sur le
réseau — jamais le contenu d'une facture.

**Pourquoi l'IA ne peut-elle pas choisir le numéro de facture ?**
Parce que la numérotation est une obligation légale (continue, sans trou) et qu'un
modèle de langage n'offre aucune garantie de ce type. Le serveur attribue le numéro
sous verrou, en une transaction avec l'écriture des fichiers — l'IA écrit un jeton
`{{NUMERO}}` et annonce le numéro prévu, c'est tout.

**Quels clients IA fonctionnent ?**
Tout client MCP : Claude Desktop, Claude Code, LM Studio, Cursor… `gofact install`
configure ceux qu'il détecte ; les autres se déclarent
[manuellement](guide/mcp.md#enregistrement-manuel).

**Faut-il un abonnement à une IA ?**
Il faut *une* IA compatible MCP. Avec LM Studio et un modèle local, l'ensemble
fonctionne entièrement hors ligne, annuaires exceptés.

## Erreurs fréquentes

**« vendeur non configuré — renseignez GOFACT_SELLER_NAME… »**
Aucune identité d'émetteur trouvée. Créez une organisation (`gofact org init` ou
`init_organization`), ou complétez le `.env` du dossier.

**« BR-50 : … l'IBAN du compte de règlement (BT-84) est requis »**
La facture annonce un paiement par virement sans donner de compte à créditer — le
Schematron Factur-X la rejetterait. Ajoutez `GOFACT_PAYEE_IBAN` au `.env` de
l'organisation.

**« facture non conforme EN 16931 : BR-… »**
gofact refuse d'émettre et nomme la règle et le champ fautif. Corrigez la donnée —
ne contournez pas en désactivant la validation : le document serait rejeté plus loin.

**« aucun navigateur trouvé pour le rendu »**
Installez Chrome, Edge, Brave ou Chromium — ou désignez un exécutable :
`GOFACT_CHROME=/chemin/vers/chrome`. Sous Linux, les paquets **snap/flatpak sont
écartés** : confinés, ils ne peuvent pas lire les fichiers que gofact leur soumet.

**« registre verrouillé — une autre attribution est en cours ? »**
Deux créations simultanées sur la même organisation : l'une attend l'autre (5 s). Un
verrou orphelin d'un processus tué est repris automatiquement après 30 s.

**« une entrée gofact différente existe déjà » (gofact install)**
Votre client MCP a déjà un serveur `gofact` pointant ailleurs. Vérifiez lequel doit
gagner, puis `gofact install -yes -force`. L'ancien fichier est sauvegardé en `.bak-…`.

**Le plugin Claude Code n'expose pas les outils gofact**
Le binaire n'est pas encore compilé (il faut la toolchain Go) : lancez une fois le
skill `creer-facture`, ou exécutez `scripts/mcp-launcher.sh` dans le dossier du plugin,
puis redémarrez la session. Sans Go : installez le binaire de release et
`gofact install -yes`.

## Factur-X

**Comment vérifier un PDF produit ?**
L'auto-contrôle tourne à chaque génération. Pour une preuve indépendante :
`go test -tags=ci ./internal/facturx -run TestOracle` (Java requis) fait juger la
chaîne par veraPDF + Schematron via Mustang. Vous pouvez aussi déposer le PDF sur un
validateur tiers — il embarque un XML standard.

**Puis-je modifier une facture émise ?**
Non — comptablement, une facture émise se corrige par un **avoir**
(`"type": "credit_note"`), jamais par retouche. Le registre et le journal gardent
l'historique.

**Où est le XML dans le PDF ?**
Pièce jointe `factur-x.xml` du PDF (visible dans tout lecteur qui affiche les pièces
jointes), déclarée en fichier associé PDF/A-3 avec la relation `Alternative`.
