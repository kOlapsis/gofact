# Facturation électronique 2026 : ce qui s'applique vraiment à un indépendant

**Réponse courte.** Depuis le 1er septembre 2026, vous devez être capable de
**recevoir** une facture électronique. Vous n'êtes pas obligé d'en **émettre**
avant le 1er septembre 2027, sauf si vous êtes une grande entreprise ou une
ETI — ce que vous n'êtes probablement pas si vous lisez cette page.

Le reste de cet article détaille ce que ça implique concrètement, et ce que
vous pouvez faire cette semaine sans dépenser un euro.

---

## Deux dates, deux obligations distinctes

La réforme sépare deux choses que le langage courant confond en permanence.

| | Réception | Émission |
| --- | --- | --- |
| Grandes entreprises et ETI | 1er septembre 2026 | 1er septembre 2026 |
| PME, TPE, micro-entreprises | 1er septembre 2026 | **1er septembre 2027** |

Autrement dit : tout le monde doit pouvoir recevoir depuis septembre 2026, y
compris l'auto-entrepreneur qui facture trois clients par an. L'obligation
d'émettre, elle, est décalée d'un an pour les petites structures.

Quand vous lisez « la facturation électronique est obligatoire depuis
septembre 2026 », c'est vrai — mais seulement de la moitié réception. Cette
imprécision, très répandue, sert souvent à créer une urgence commerciale.

## Ce que « recevoir » veut dire techniquement

Recevoir une facture électronique, ce n'est pas recevoir un PDF en pièce
jointe. C'est être joignable sur une **plateforme agréée**.

Une plateforme agréée — PA, anciennement PDP — est un opérateur immatriculé par
la DGFiP, habilité à émettre, recevoir et transmettre les factures entre
entreprises, et à remonter les données fiscales à l'administration. Il en existe
plus d'une centaine.

À côté, le **Portail Public de Facturation** conserve deux missions : il tient
l'**annuaire central**, qui indique sur quelle plateforme chaque entreprise
reçoit ses factures, et il joue le rôle de **concentrateur** des données
fiscales destinées à l'administration.

La conséquence pratique est simple. Si vous n'êtes déclaré sur aucune
plateforme, vos fournisseurs déjà passés à l'électronique ne savent pas où vous
adresser leurs factures. Ce n'est pas une amende immédiate ; c'est une facture
qui n'arrive pas, ou qui arrive par un chemin dégradé.

## Ce qu'est vraiment une facture électronique

Une facture électronique au sens de la réforme est un fichier **structuré** :
des données identifiées, lisibles par une machine sans reconnaissance de
caractères. Trois formats sont admis : Factur-X, UBL et CII.

**Factur-X** est celui qui domine en France, parce qu'il ne force personne à
choisir entre l'humain et la machine. Un fichier Factur-X est un PDF/A-3 —
donc un PDF normal, ouvrable et archivable dix ans — dans lequel est **embarqué**
le XML normalisé conforme à EN 16931. Votre client ouvre le PDF s'il veut le
lire ; son logiciel lit le XML s'il veut l'intégrer.

Cette norme EN 16931 définit chaque champ et chaque règle de contrôle. Le total
hors taxes, le montant de TVA, l'identifiant du vendeur : tout porte un nom
précis et obéit à des règles identifiées, du type `BR-50` ou `BR-CO-15`. Un
outil sérieux vous cite ces codes quand il refuse d'émettre.

!!! warning "Un PDF « joli » n'est pas une facture électronique"

    Beaucoup de générateurs de factures gratuits produisent un PDF sans XML
    embarqué. Ce fichier reste, du point de vue du dispositif, une image de
    facture. Il ne vous mettra pas en conformité en 2027.

## Faut-il payer un abonnement ?

Le passage par une plateforme agréée est obligatoire pour transmettre. Payer un
abonnement pour **produire** ses factures ne l'est pas.

Ce sont deux couches différentes, et la confusion entre les deux est
entretenue par la structure du marché : la plupart des offres vendent les deux
ensemble. Sur la centaine de plateformes agréées, seule une poignée propose une
formule gratuite, et aucune n'est open source.

La distinction utile à garder en tête :

> **La plateforme est un tuyau. La facture est un fichier.**
> Le tuyau se change. Le fichier vous appartient — pendant dix ans.

Si votre outil de production est une couche que vous possédez, changer de
plateforme agréée est une reconfiguration. Si votre outil de production *est*
la plateforme, en changer signifie migrer dix ans d'historique.

## Ce que vous pouvez faire cette semaine

Quatre actions, aucune n'engage de dépense.

1. **Vérifier votre situation d'entreprise.** Micro, TPE, PME : votre échéance
   d'émission est septembre 2027. Vérifiez-le auprès de votre expert-comptable
   plutôt que sur un article de blog — celui-ci compris.
2. **Choisir votre plateforme de réception** et vous y déclarer, pour être
   joignable dans l'annuaire. C'est l'action urgente, et souvent la seule.
3. **Regarder à quoi ressemble un Factur-X.** Produisez-en un, ouvrez-le,
   inspectez le XML embarqué. Une heure suffit à démystifier le sujet.
4. **Ne pas décider de votre outil d'émission dans l'urgence.** Vous avez douze
   mois. C'est exactement le temps qu'il faut pour choisir sans regretter.

## Où gofact se situe

Pour être clair sur un point qui compte : **gofact n'est pas une plateforme
agréée**. Il ne transmet rien à l'administration et ne vous dispense pas d'en
choisir une.

gofact produit le **fichier** : il transforme une facture HTML en Factur-X
conforme — PDF/A-3 avec le XML CII EN 16931 embarqué octet pour octet —
applique les règles EN 16931 avant de produire quoi que ce soit, relit le
fichier écrit pour vérifier ses structures, puis sait le déposer sur la
plateforme agréée que *vous* avez choisie.

C'est un binaire qui tourne sur votre machine. Pas de compte, pas de serveur,
pas de télémétrie. Vos factures et vos clients restent dans un dossier local
qui vous appartient.

Pour commencer : [Installation](../installation.md) puis
[Premiers pas](../demarrage.md).

---

## Sources

- [Calendrier de la facturation électronique — Cegid](https://www.cegid.com/fr/facture-electronique-obligatoire/calendrier-facture-electronique/)
- [Liste des plateformes agréées DGFiP — data.gouv.fr](https://www.data.gouv.fr/datasets/liste-des-plateformes-agreees-dgfip-pour-la-facturation-electronique)
- [Le rôle de l'annuaire du PPF — La Poste](https://www.laposte.fr/entreprise-collectivites/actualites/reforme-facture-electronique-annuaire-ppf-au-coeur-dispositif)
- [Facturation électronique : la réforme à anticiper — Bpifrance](https://conseil.bpifrance.fr/publications/facturation-electronique-obligatoire-un-tournant-digital-pour-les-entreprises-francaises)

*Cet article décrit un dispositif général. Il ne constitue pas un conseil
fiscal. Pour votre situation particulière, adressez-vous à votre
expert-comptable.*
