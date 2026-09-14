# La numérotation légale : verrou, transaction, et pourquoi un LLM n'y touche pas

**Réponse courte.** Non : votre IA ne doit jamais choisir le numéro d'une
facture, quelle que soit la qualité du modèle. La numérotation est un
invariant légal — unique, chronologique, continue — et un modèle de langage
ne peut garantir aucun des trois. Dans gofact, le numéro est attribué par un
mécanisme déterministe, sous verrou, au moment exact où la facture s'écrit ;
l'IA n'en connaît jamais la valeur avant qu'elle existe.

Le reste de cet article explique pourquoi cette séparation est nécessaire, et
comment elle est construite.

---

## Ce que la loi exige d'un numéro de facture

L'article 242 nonies A de l'annexe II au code général des impôts impose que
chaque facture porte un numéro unique, fondé sur une séquence chronologique
et continue. Trois exigences tiennent dans cette phrase :

- **Unique** — un numéro n'est jamais réattribué, même si la facture
  correspondante est annulée après coup.
- **Chronologique** — les numéros suivent l'ordre réel d'émission, pas un
  ordre reconstitué après coup.
- **Continue** — aucun numéro ne doit manquer. Un trou dans la séquence (une
  facture 2026042 qui succède directement à 2026040) se lit, lors d'un
  contrôle, comme une facture dissimulée.

Cette règle existait bien avant la réforme de la facturation électronique. Ce
qui change avec elle, c'est le support : le numéro fait désormais partie d'un
fichier structuré, lu par des machines, et une incohérence s'y repère plus
facilement qu'à l'œil sur une pile de PDF.

## Pourquoi un modèle de langage ne peut pas la tenir

Un grand modèle de langage compose du texte plausible à partir de ce qu'on
lui donne. C'est exactement ce qui le rend utile pour rédiger une facture à
partir d'une conversation — et exactement ce qui le rend inapte à tenir un
compteur.

Trois raisons concrètes :

1. **Il n'a pas d'état garanti entre deux appels.** Rien n'empêche,
   structurellement, deux conversations menées en parallèle d'aboutir au même
   numéro si c'est le modèle qui le propose lui-même.
2. **Il ne connaît pas la panne.** Si la génération du fichier échoue après
   qu'un numéro a été « annoncé », rien ne garantit qu'il ne sera pas
   réutilisé — ou, à l'inverse, qu'il ne laissera pas un trou si personne ne
   s'en aperçoit.
3. **Il n'est pas déterministe.** Deux exécutions d'une même instruction
   peuvent produire des sorties différentes. Un numéro de facture n'a pas le
   droit d'être probable ; il doit être exact.

Aucune de ces limites n'est un défaut du modèle : ce sont des propriétés
normales d'un système de génération de texte, appliquées à une tâche qui
exige l'inverse — un état persistant, une opération atomique, un résultat
reproductible.

## Comment gofact sépare les deux rôles

Le serveur MCP de gofact expose un outil, `create_invoice`, qui est la seule
porte d'entrée pour créer une facture. L'IA lui fournit le contenu — client,
lignes, montants — mais jamais le numéro : le modèle de facture qu'elle
compose contient un jeton, `{{NUMERO}}`, que gofact substitue lui-même au
moment où le fichier s'écrit.

Ce moment est verrouillé et transactionnel :

- **Verrouillé.** Un fichier de verrou, posé en exclusion mutuelle sur le
  dossier de l'organisation, garantit que deux appels concurrents ne peuvent
  jamais obtenir le même numéro. Un verrou plus vieux que trente secondes est
  considéré comme l'héritage d'un processus mort et repris automatiquement.
- **Transactionnel.** L'attribution du numéro, l'écriture du fichier et
  l'inscription au registre se font en une seule opération. Si l'une des
  trois étapes échoue, rien n'est persisté : ni compteur avancé, ni entrée de
  registre, ni trou dans la séquence.

Un outil séparé, `preview_next_number`, existe pour *annoncer* le numéro
probable avant confirmation — utile pour que votre IA vous dise « ce sera la
facture 2026042 » avant de la produire. Mais cet outil ne consomme rien :
rien ne garantit que ce numéro sera encore libre au moment de la création
réelle, et c'est volontaire. Seule l'attribution sous verrou fait foi.

!!! info "Reprendre une numérotation existante"

    Si vous avez déjà émis des factures ailleurs, un outil dédié,
    `initialize_numbering`, permet de faire repartir le compteur d'une année
    au bon endroit. Il ne peut que le faire **monter** : l'abaisser
    réutiliserait des numéros déjà émis, ce qui est une irrégularité
    comptable que gofact refuse d'introduire.

## Ce que ça change, concrètement, pour vous

Quand vous dites à votre IA « fais-moi la facture d'août pour ce client », le
déroulé est le suivant : le modèle retrouve ou vous demande les informations
manquantes, compose la facture, vous la montre pour validation — et ce n'est
qu'à la confirmation que gofact attribue le numéro et écrit le fichier. Vous
validez un contenu, jamais un numéro : celui-ci n'existe pas encore au moment
où vous regardez la facture.

Cette séparation a une conséquence utile en cas d'erreur : si vous annulez
avant confirmation, aucun numéro n'a été consommé. La séquence légale ne
connaît que les factures réellement écrites.

## Ce qui reste de votre responsabilité

gofact garantit la mécanique de la séquence — continuité, unicité,
atomicité. Il ne garantit pas que vous ne créiez pas la même facture deux
fois si vous le demandez deux fois de suite : c'est un choix de contenu, pas
un problème de numérotation, et c'est à vous de le repérer avant de valider.
Le registre des factures, consultable à tout moment, reste la référence pour
vérifier ce qui a réellement été émis.

## Où gofact se situe

Pour être clair sur ce point aussi : la numérotation légale n'a rien à voir
avec la transmission de la facture. gofact produit le fichier et tient le
compteur sur votre machine ; c'est votre plateforme agréée qui achemine
ensuite la facture vers son destinataire et vers l'administration. **gofact
ne transmet rien lui-même.**

Pour aller plus loin sur le fonctionnement du serveur : [Parler à son IA —
le serveur MCP](../guide/mcp.md).

---

## Sources

- [Article 242 nonies A — Code général des impôts, annexe II — Légifrance](https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000050811276)
- [Numérotation des factures : les règles à respecter — Le Coin des Entrepreneurs](https://www.lecoindesentrepreneurs.fr/numeroter-correctement-ses-factures-correctement/)

*Cet article décrit un dispositif général. Il ne constitue pas un conseil
fiscal ou juridique. Pour votre situation particulière, adressez-vous à votre
expert-comptable.*
