# Archiver dix ans : ce que la loi demande et ce que produit gofact

**Réponse courte.** Vos factures doivent rester consultables dix ans au titre
du droit commercial, et six ans au titre du contrôle fiscal. Ces deux durées
coexistent, portent sur des textes différents, et aucune des deux ne se limite
à « garder un PDF quelque part » : la loi exige que le document reste
authentique, intègre et lisible pendant toute la période, pas seulement
présent. Voici ce que ces obligations recouvrent concrètement, et où se
situe un outil comme gofact.

---

## Deux obligations, deux textes, deux durées

La confusion la plus fréquente vient du fait qu'on parle d'« archivage » comme
d'une seule règle, alors qu'il y en a deux, qui ne courent pas sur la même
durée et ne relèvent pas de la même administration.

**L'obligation comptable** vient de l'article [L123-22 du Code de
commerce](https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000006219327) :
les documents comptables et les pièces justificatives — les factures en font
partie — sont conservés **dix ans**. Le compteur démarre à la clôture de
l'exercice concerné, pas à la date de la facture elle-même.

**L'obligation fiscale** vient de l'article [L102 B du Livre des procédures
fiscales](https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000041471233/) :
les livres, registres, documents ou pièces sur lesquels peut s'exercer le
droit de communication, d'enquête et de contrôle de l'administration sont
conservés **six ans**, à compter de la dernière opération mentionnée ou de la
date d'établissement du document.

Dans la pratique, ces deux durées ne s'excluent pas : on retient la plus
longue, soit dix ans, dès qu'un document a une portée à la fois comptable et
fiscale — ce qui est le cas de la quasi-totalité des factures. Le fait qu'un
contrôle fiscal ne remonte en principe qu'à six ans ne dispense donc pas de
garder le document quatre ans de plus au titre du droit commercial.

## Ce que la loi demande vraiment : pas juste « garder un fichier »

Conserver un document ne veut pas dire le laisser traîner dans un coin. Trois
propriétés doivent tenir pendant toute la durée légale :

- **L'authenticité de l'origine** — pouvoir démontrer qui a émis le document,
  sans ambiguïté.
- **L'intégrité du contenu** — le document n'a pas été modifié depuis son
  émission.
- **La lisibilité** — le document reste consultable en clair, y compris des
  années après, indépendamment du logiciel qui l'a produit.

Pour une facture électronique, ces trois propriétés sont garanties soit par
une **piste d'audit fiable** (une chaîne de contrôles documentés reliant la
facture à la transaction sous-jacente), soit par l'usage d'un **format
structuré** répondant au profil imposé par la réforme, qui porte cette
garantie dans sa structure même plutôt que dans un dossier de preuves annexe.

C'est là que le choix du format n'est pas un détail technique : un fichier
qui ne garantit ni l'authenticité ni l'intégrité par construction reporte
cette charge sur une piste d'audit qu'il faut documenter et maintenir à côté,
pendant dix ans.

## Pourquoi le format compte autant que la durée

Un PDF ordinaire peut très bien rester dix ans sur un disque. Le problème
n'est pas qu'il disparaisse : c'est qu'il dépende, pour rester lisible, d'un
environnement — une police installée, un lecteur compatible, un rendu de
couleurs — qui n'a aucune obligation de survivre dix ans à l'identique.

Le profil **PDF/A**, dont dépend Factur-X, existe précisément pour éliminer
cette dépendance : polices embarquées, pas de contenu externe référencé,
métadonnées normalisées. Le suffixe `-3` autorise en plus les fichiers
embarqués, ce qui permet à Factur-X de transporter le XML structuré de la
facture à l'intérieur du même fichier PDF que celui qu'un humain ouvre pour
la lire. Un seul fichier à archiver, qui porte les deux couches — lisible par
un humain, exploitable par une machine — au lieu de deux documents qu'il
faudrait garder synchronisés pendant dix ans.

## Ce qui casse une archive avant dix ans, en pratique

Le format n'est qu'une partie du problème. Ce qui met une archive en danger
sur une décennie, ce sont surtout des dépendances qu'on ne remarque qu'au
moment où elles cèdent :

- **Un service qui ferme.** Si vos factures ne sont consultables que via
  l'interface d'un éditeur, sa fermeture — ou simplement l'arrêt d'un
  abonnement — peut couper l'accès à un historique que la loi vous oblige
  pourtant à produire en cas de contrôle.
- **Un format propriétaire.** Un export non standard, lisible uniquement par
  le logiciel qui l'a produit, transforme la lisibilité en pari sur la
  pérennité de cet éditeur précis.
- **Une dépendance à un tiers pour simplement ouvrir le fichier.** Plus un
  document dépend d'éléments externes pour être lu, plus le risque de rupture
  augmente avec le temps qui passe.

Aucune de ces trois situations n'est hypothétique : ce sont exactement les
questions à se poser avant de confier la conservation de dix ans d'historique
à un service qu'on ne maîtrise pas.

## Où gofact se situe

Pour être clair sur ce point : **gofact n'est pas un service d'archivage à
valeur probante**, et il ne prétend pas l'être. Il ne délivre ni horodatage
qualifié, ni cachet électronique, ni piste d'audit documentée au sens de
l'administration fiscale.

Ce que gofact fait : produire, sur votre machine, un Factur-X conforme —
PDF/A-3 avec le XML CII EN 16931 embarqué verbatim, octet pour octet — et le
garder dans un dossier local que vous possédez, sans base de données ni état
caché. Ce fichier ne dépend, pour rester lisible dans dix ans, ni d'un compte
qui doit rester actif, ni d'un abonnement qui doit rester payé, ni d'un
service qui doit rester en ligne : c'est un fichier, dans un dossier, comme
n'importe quel autre document que vous conservez déjà.

**gofact n'est pas une plateforme agréée.** Il ne transmet rien lui-même à
votre client ni à l'administration ; c'est votre plateforme agréée qui
achemine la facture, aujourd'hui comme dans dix ans, quelle que soit celle
que vous aurez choisie d'ici là. Certaines plateformes agréées proposent en
option un service d'archivage à valeur probante — une garantie
supplémentaire que gofact ne fournit pas et n'a pas vocation à fournir : à
vous de l'évaluer si votre situation le justifie, avec votre
expert-comptable.

## Ce que vous pouvez vérifier cette semaine

1. **Retrouvez où vivent vos dix dernières années de factures** aujourd'hui —
   dans un dossier que vous possédez, ou uniquement dans l'interface d'un
   service tiers.
2. **Vérifiez le format d'export** proposé par vos outils actuels : un PDF/A
   ou un Factur-X se garde tel quel ; un format propriétaire demande une
   conversion, à faire maintenant plutôt que le jour du contrôle.
3. **Distinguez les deux durées** : ne classez pas comme réglé un document
   qui n'a que six ans, s'il porte aussi une dimension comptable qui en
   demande dix.

---

## Sources

- [Article L123-22 — Code de commerce, Légifrance](https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000006219327)
- [Article L102 B — Livre des procédures fiscales, Légifrance](https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000041471233/)
- [Facturation électronique et plateformes agréées — impots.gouv.fr](https://www.impots.gouv.fr/facturation-electronique-et-plateformes-agreees)

*Cet article décrit un dispositif général. Il ne constitue pas un conseil
fiscal ou juridique. Pour votre situation particulière, adressez-vous à votre
expert-comptable.*
