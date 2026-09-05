---
date:
heure:
canal: linuxfr
type: journal
pilier: B
sujet: Ce qu'il faut vraiment écrire dans un PDF pour qu'il devienne un Factur-X
statut: brouillon — relecture Benjamin avant publication
url:
---

# Ce qu'il faut vraiment écrire dans un PDF pour qu'il devienne un Factur-X

Depuis le 1er septembre, toute entreprise assujettie à la TVA doit être capable
de *recevoir* une facture électronique. Les grandes entreprises et les ETI
doivent aussi en *émettre* ; les PME, TPE et micro-entreprises ont jusqu'au
1er septembre 2027. Trois formats structurés sont admis : Factur-X, UBL et CII.

Factur-X est le plus curieux des trois, parce que ce n'est pas vraiment un
format : c'est un PDF ordinaire dans lequel on a rangé un XML. L'humain ouvre
le PDF et lit sa facture ; la machine va chercher la pièce jointe et lit les
mêmes données, structurées. Les deux doivent dire la même chose.

J'ai passé un moment à comprendre ce que cela veut dire *concrètement*, au
niveau des objets PDF. Voici ce que j'ai trouvé, parce que la documentation
disponible est soit une norme à 300 € soit un tutoriel qui vous dit d'appeler
Ghostscript.

## Le malentendu de départ

« Factur-X = PDF + XML attaché » est une description exacte et parfaitement
inutilisable. Attacher un fichier à un PDF prend trois lignes avec n'importe
quelle bibliothèque. Le résultat n'est pas un Factur-X, et aucun validateur ne
l'acceptera.

La vraie contrainte est ailleurs : le PDF doit être un **PDF/A-3**. PDF/A est le
profil d'archivage — le document doit rester lisible dans vingt ans sans
dépendre de rien d'extérieur. Toutes les polices embarquées, aucune couleur
dépendant d'un périphérique, aucun contenu qui exécute quoi que ce soit. Le
suffixe `-3` est celui qui autorise les fichiers embarqués, ce qui est
précisément la raison pour laquelle Factur-X s'appuie dessus.

C'est donc un problème de conformité PDF/A, avec un XML dedans.

## Ce qu'il manque à un PDF produit par un navigateur

Je génère la facture en HTML et je la fais rendre par Chrome en mode headless.
Le PDF qui en sort est propre : Skia embarque les polices, ne produit pas de
transparence exotique, n'exécute rien. Mais il n'est pas PDF/A-3, et il lui
manque exactement quatre choses.

**1. Le XML, embarqué et déclaré.** Pas seulement dans l'arbre `/EmbeddedFiles`,
mais aussi référencé par `/AF` (*associated files*) au niveau du catalogue, avec
`AFRelationship` valant `Alternative` — le XML est une *autre* représentation du
même document, pas une annexe. Le nom du fichier doit être `factur-x.xml`, pas
autre chose.

**2. Un OutputIntent.** C'est le point qui coûte le plus de temps à trouver et
qui règle le plus de violations d'un coup. En PDF/A, tout usage de `DeviceRGB`
est interdit *sauf* si le document déclare un profil colorimétrique de sortie.
Il faut donc embarquer un profil ICC sRGB et le déclarer. Détail savoureux : la
clé `/S` de l'OutputIntent vaut `GTS_PDFA1` **y compris en PDF/A-3**. Ce n'est
pas une coquille, c'est la norme ; la valeur n'a jamais été renumérotée.

**3. Un paquet XMP.** Les métadonnées XML qui déclarent `pdfaid:part = 3` et
`pdfaid:conformance = B`. Trois pièges dedans :

- le `dc:title` du XMP doit être **identique** au `/Title` du dictionnaire
  `Info`, sinon non-conformité ;
- dès qu'on emploie un espace de noms hors norme PDF/A — et l'espace Factur-X en
  est un — il faut déclarer un **schéma d'extension** dans le XMP ;
- le paquet XMP ne doit **pas** être compressé. Un validateur doit pouvoir le
  lire sans décompresser quoi que ce soit.

**4. Un `/ID` dans le trailer.** PDF/A exige un identifiant de fichier. Chrome
n'en écrit pas.

C'est tout. Quatre ajouts, et le PDF passe veraPDF. Je ne retouche rien d'autre :
ni reconversion colorimétrique, ni ré-encodage des polices. Ce qui sort de Skia
pour une facture n'emploie que des constructions déjà admises en PDF/A-3.

## Le point qui m'a le plus surpris : l'embarquement verbatim

Beaucoup d'outils assemblent le Factur-X en désérialisant le XML, en le
remaniant, puis en le ré-émettant. C'est commode et c'est une mauvaise idée.

Le XML CII de la norme EN 16931 est extensible, et la réforme française s'appuie
sur cette extensibilité. Les adresses de routage entre plateformes — `BT-34`
pour l'émetteur, `BT-49` pour le destinataire — vivent dans des champs que tous
les outils ne connaissent pas. Un assembleur qui reconstruit le XML depuis son
propre modèle de données **perd silencieusement** ce qu'il ne modélise pas. La
facture reste valide au sens de la norme, et devient inacheminable.

J'écris donc le XML une fois, et je l'insère octet pour octet. Le flux embarqué
est exactement celui que j'ai produit, taille déclarée comprise. C'est moins
élégant qu'un aller-retour par un modèle objet, et c'est la seule façon de
garantir qu'on n'a rien perdu en route.

## Sans Java

L'outillage de référence du domaine est en Java : Mustang pour la validation,
veraPDF pour le PDF/A, souvent Ghostscript pour la conversion PDF/A. Ça marche
très bien, et ça suppose une JVM chez l'utilisateur.

Tout ce qui précède tient en Go pur avec
[pdfcpu](https://github.com/pdfcpu/pdfcpu) : lecture du PDF rendu, ajout des
objets, réécriture. Le profil ICC sRGB est embarqué dans le binaire — 2,5 ko.
Aucun CGO, donc un binaire statique par plateforme et rien à installer, hormis
un navigateur pour le rendu.

À noter tout de même : **je ne suis pas le premier**.
[`angelodlfrtr/go-invoice-generator`](https://github.com/angelodlfrtr/go-invoice-generator)
assemble déjà du Factur-X PDF/A-3B en Go pur, en cinq profils, validé mustang-cli
et veraPDF. « PDF/A-3 sans Java » n'est pas un exploit, c'est juste faisable et
peu documenté.

## La vérification, qui ne se délègue pas à soi-même

Un point de méthode qui vaut au-delà de ce sujet : un assembleur qui se valide
lui-même ne prouve rien. Ma CI fait juger la sortie du binaire par
**Mustang** — donc veraPDF pour le PDF/A et le Schematron officiel EN 16931 pour
les règles métier. Java n'existe que là, dans le job d'intégration, et jamais
chez l'utilisateur. C'est le seul endroit où un oracle extérieur a du sens.

## L'outil

Tout ceci est extrait de [gofact](https://github.com/kOlapsis/gofact), sous
AGPL-3.0. C'est un binaire Go qui prend une facture en HTML et un JSON de
données, et produit le Factur-X. Il fait aussi serveur MCP, ce qui permet de
demander une facture à un assistant en langage naturel — avec une réserve que je
tiens à formuler nettement : **la numérotation des factures n'est pas confiée au
modèle de langage**. La séquence légale, continue et sans trou, est tenue par le
programme dans un registre verrouillé ; le modèle écrit un jeton et le serveur
attribue le numéro, de façon transactionnelle. C'est le genre d'endroit où « le
modèle s'en occupe » n'est pas une réponse acceptable.

Deux limites à dire tout de suite : **gofact n'est pas une plateforme agréée**.
Il produit le fichier ; votre plateforme agréée le transporte jusqu'à votre
client et à l'administration. Et la TVA y est mono-taux pour l'instant.

Si le sujet vous intéresse et que vous voyez une erreur dans ce qui précède,
elle m'intéresse encore plus.
