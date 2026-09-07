# Anatomie d'un Factur-X : ce qu'il y a vraiment dans le fichier

**Réponse courte.** Un fichier Factur-X est un PDF ordinaire qui contient,
caché à l'intérieur, un second fichier : un XML structuré que la machine de
votre client lit directement, sans jamais ouvrir le PDF. Les deux doivent
raconter exactement la même facture. Voici ce qu'il y a dans chacune des deux
couches, et pourquoi c'est construit ainsi.

---

## La couche que vous voyez : un PDF, et rien d'exotique

Ouvrez un Factur-X dans n'importe quel lecteur PDF : vous voyez une facture,
comme n'importe quel PDF que vous avez déjà envoyé par mail. C'est
intentionnel. Le format a été pensé pour qu'un humain n'ait besoin de rien de
spécial pour le lire.

Ce PDF a tout de même une contrainte que la plupart des PDF n'ont pas : c'est
un **PDF/A-3**. PDF/A est le profil d'archivage à long terme — le document
doit rester lisible dans vingt ans sans dépendre d'un logiciel, d'une police
ou d'une couleur externes. Le suffixe `-3` est le seul des profils PDF/A à
autoriser les fichiers embarqués, ce qui est précisément ce dont Factur-X a
besoin pour la couche suivante.

## La couche que vous ne voyez pas : un XML CII

À l'intérieur du PDF, un fichier nommé `factur-x.xml` porte la même facture,
mais structurée : un vendeur, un acheteur, des lignes, des montants, une TVA,
chacun dans un champ nommé. C'est ce fichier que le logiciel de votre client
lit — jamais le PDF.

Le format de ce XML s'appelle **CII** (Cross Industry Invoice), un schéma
défini par l'ONU (UN/CEFACT). Factur-X n'invente pas son propre XML : il
réutilise CII et lui applique un profil, c'est-à-dire un sous-ensemble de
champs obligatoires et optionnels adapté à la facturation courante.

C'est ici qu'il faut dissiper une confusion fréquente : Factur-X, UBL et CII
ne sont pas trois formats concurrents que vous choisissez au hasard. UBL et
CII sont deux syntaxes XML admises par la réforme française ; Factur-X est
une manière de transporter du CII **à l'intérieur d'un PDF** plutôt que de
l'envoyer nu. Un système peut recevoir l'un ou l'autre — c'est justement le
rôle de l'annuaire et des plateformes agréées de savoir dans quel format
chaque destinataire accepte ses factures.

## Ce que la norme EN 16931 impose au contenu

Le XML CII pourrait, en théorie, contenir n'importe quoi. Ce qui le rend
exploitable par n'importe quel logiciel européen, c'est la norme
**EN 16931** : elle nomme chaque donnée obligatoire d'une facture avec un
identifiant stable — BT-1 pour le numéro de facture, BT-5 pour la devise,
BT-31 pour l'identifiant TVA du vendeur — et définit des règles de cohérence
entre ces champs, elles aussi identifiées par un code : `BR-CO-15` vérifie que
le total hors taxes correspond à la somme des lignes, par exemple.

Deux champs de cette nomenclature méritent une attention particulière parce
qu'ils ne sautent pas aux yeux sur la facture affichée : **BT-34** et
**BT-49**, les adresses de routage de l'émetteur et du destinataire. Elles ne
sont pas votre adresse postale : ce sont des identifiants qui disent à quelle
plateforme agréée chacun est joignable dans l'annuaire. Une facture peut être
parfaitement lisible pour un humain et complètement inacheminable si ces deux
champs sont absents ou mal renseignés.

La transposition française ajoute ses propres règles, préfixées `BR-FR-*` :
c'est là, par exemple, que se nichent les mentions légales obligatoires en
droit français (pénalités de retard, indemnité forfaitaire de recouvrement,
conditions d'escompte) qui n'ont pas d'équivalent direct dans la norme
européenne.

## Pourquoi la reconstruction du XML est le point de rupture le plus fréquent

Le XML CII est extensible : un logiciel peut y ajouter des champs qu'un autre
logiciel ne connaît pas, sans casser la validité du document. C'est une
qualité du format — et un piège pour qui l'implémente mal.

Beaucoup d'outils assemblent leur Factur-X en désérialisant le XML dans un
modèle de données interne, puis en le ré-émettant au moment de produire le
fichier. Si ce modèle ne connaît pas BT-34 ou BT-49, il ne les recopie pas à
l'identique : il les *oublie*, silencieusement. Le validateur EN 16931 ne
signale rien, parce que ces champs sont optionnels du point de vue de la
norme générale. La facture reste valide, et devient inacheminable — un défaut
qui n'apparaît qu'au moment du dépôt sur la plateforme, jamais avant.

La façon d'éviter ce piège n'a rien de sophistiqué : ne pas reconstruire le
XML qu'on a déjà écrit. C'est le choix que fait gofact — le XML produit une
fois est celui qui est embarqué, octet pour octet, sans nouvel aller-retour
par un modèle objet. Moins élégant qu'une désérialisation propre ; c'est
aussi la seule garantie de ne rien perdre en route.

## Ce qui distingue une facture valide d'une facture acceptée

Un fichier peut être un Factur-X valide — bon PDF/A-3, bon XML, règles
EN 16931 respectées — et être tout de même refusé par une plateforme agréée,
pour des raisons qui ne relèvent pas du format : un identifiant TVA erroné,
un destinataire introuvable dans l'annuaire, une mention légale manquante
propre à la réglementation française. La validité du fichier est une
condition nécessaire ; elle ne dit rien sur l'exactitude des informations
qu'il contient.

C'est pour cette raison qu'un outil sérieux applique les règles de cohérence
**avant** de produire le fichier, refuse d'émettre quand l'une d'elles casse,
et nomme la règle en cause plutôt que de renvoyer une erreur générique. Ce
qu'un validateur ne peut pas faire, en revanche, c'est deviner que vous avez
donné le mauvais SIREN à votre client.

## Où gofact se situe

gofact produit le **fichier** : il transforme une facture HTML en Factur-X
conforme — PDF/A-3 avec le XML CII EN 16931 embarqué verbatim — applique les
règles EN 16931 avant la production, puis relit le fichier écrit pour
vérifier ses structures. Il sait ensuite le déposer sur la plateforme agréée
que *vous* avez choisie.

Pour être clair sur un point qui compte : **gofact n'est pas une plateforme
agréée**. Il ne transmet rien lui-même à l'administration ni à votre client —
c'est votre plateforme agréée qui achemine le fichier, avec la bonne adresse
de routage.

Pour aller plus loin : [Conformité](../conformite.md) détaille ce que gofact
vérifie précisément, et [Envoi PDP](../pdp.md) explique le dépôt sur une
plateforme.

---

## Sources

- [Norme EN 16931 — présentation et champs](https://www.economie.gouv.fr/entreprises/facturation-electronique-entreprises)
- [Arrêté du 27 juillet 2026 relatif à la généralisation de la facturation électronique — Légifrance](https://www.legifrance.gouv.fr/jorf/id/JORFTEXT000054499535)
- [Décret n° 2026-677 du 27 juillet 2026 — Légifrance](https://www.legifrance.gouv.fr/jorf/id/JORFTEXT000054499487)
- [Liste des plateformes agréées DGFiP — data.gouv.fr](https://www.data.gouv.fr/datasets/liste-des-plateformes-agreees-dgfip-pour-la-facturation-electronique)
- [Annuaire de la facturation électronique — Dext](https://dext.com/fr/ressources/blog-actualite-comptable/single/annuaire-ppf-comment-ca-marche)

*Cet article décrit un dispositif général. Il ne constitue pas un conseil
fiscal. Pour votre situation particulière, adressez-vous à votre
expert-comptable.*
