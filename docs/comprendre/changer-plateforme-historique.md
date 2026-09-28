# Changer de plateforme agréée sans perdre son historique

**Réponse courte.** Depuis le décret du 27 juillet 2026, changer de plateforme
agréée (PA) est un droit garanti, pas une négociation avec votre prestataire :
vos données de facturation, vos adresses de routage dans l'annuaire et votre
historique de factures sont portables. Ce que la portabilité ne fait pas, en
revanche, c'est déplacer instantanément dix ans d'archives dans le système de
la nouvelle plateforme. Voici ce que ce droit recouvre concrètement, ce qu'il
ne couvre pas, et où se situe un outil comme gofact.

---

## Pourquoi cette question ne se posait pas avant

Avant l'entrée en vigueur du cadre réglementaire de 2026, rien n'obligeait une
plateforme de facturation à faciliter le départ d'un client. La portabilité —
récupérer ses données, garder trace de son historique — dépendait entièrement
de ce que le contrat commercial prévoyait, et beaucoup ne prévoyaient rien de
précis sur ce point.

La réforme change cette donne parce qu'elle change la nature de la relation :
une plateforme agréée (PA) n'est plus seulement un fournisseur logiciel, c'est
un intermédiaire obligatoire dans un dispositif fiscal. Un intermédiaire
obligatoire ne peut pas, dans le même mouvement, retenir ce qui vous
appartient au moment où vous voulez en changer. Le
[décret n° 2026-677 du 27 juillet 2026](https://www.legifrance.gouv.fr/jorf/id/JORFTEXT000054499487)
transpose cette logique dans le code général des impôts : le changement de
plateforme devient une procédure encadrée plutôt qu'un sujet laissé à la
bonne volonté de chacun.

## Un droit, pas une négociation

Le texte distingue trois choses qui doivent vous suivre lorsque vous quittez
une plateforme agréée pour une autre :

- **Vos données de facturation** — vos clients, vos fournisseurs, l'historique
  de ce qui a été déposé et transmis en votre nom.
- **Vos adresses électroniques de routage** — celles qui permettent à vos
  partenaires de vous joindre dans l'annuaire (PPF), quelle que soit la
  plateforme que vous utilisez à un instant donné. Sans mise à jour correcte
  de ces adresses au moment de la bascule, une facture qui vous est destinée
  peut continuer à être acheminée vers la plateforme que vous avez quittée.
- **Un service minimal de l'ancienne plateforme pendant la transition** —
  pour éviter une coupure brutale d'accès à vos documents le jour où vous
  changez de prestataire.

Le point le plus utile à retenir n'est pas la durée exacte de ce service
minimal : les textes d'application en ont discuté plusieurs valeurs au cours
des débats parlementaires, et une durée lue aujourd'hui peut ne plus être la
bonne dans quelques mois. Le point qui compte, et qui ne varie pas, c'est que
cette durée existe désormais par la loi, et que sa négociation ne dépend plus
du bon vouloir de la plateforme que vous quittez. Avant tout changement,
vérifiez la durée en vigueur directement auprès de votre plateforme actuelle
ou de la documentation officielle à jour plutôt que sur une date que vous
auriez lue ailleurs.

## Ce que la portabilité ne fait pas

Portabilité ne veut pas dire téléportation. Le jour où vous activez un compte
chez une nouvelle plateforme agréée, elle ne dispose d'aucune facture traitée
par l'ancienne : l'historique reste, dans un premier temps, chez le
prestataire que vous quittez, avec l'obligation de vous y donner accès pendant
la durée de service minimal évoquée plus haut.

Concrètement, changer de plateforme suppose une étape que la loi ne fait pas
à votre place : récupérer une copie de votre historique — export des factures,
des statuts de traitement, des accusés de réception — avant, ou au moment du
changement, plutôt que de compter sur un transfert automatique entre les deux
systèmes. Les deux plateformes n'ont, techniquement, aucune raison de se
parler directement.

## Ce qui casse en pratique lors d'un changement

Trois points de friction reviennent le plus souvent lorsqu'un changement de
plateforme agréée est mal préparé :

- **Une adresse de routage qui pointe encore vers l'ancienne plateforme.**
  Les champs qui portent cette adresse dans une facture (BT-34 pour
  l'émetteur, BT-49 pour le destinataire) doivent correspondre à ce qui est
  inscrit dans l'annuaire au moment du dépôt. Une mise à jour tardive de
  l'annuaire, pendant que vos partenaires facturent encore sur l'ancienne
  adresse, produit exactement le défaut qu'une facture valide au sens de la
  norme peut avoir : inacheminable, sans que rien dans le fichier ne le
  signale à l'avance.
- **Des factures « en transit » au moment de la bascule.** Une facture
  déposée juste avant le changement, dont le statut n'est pas encore
  définitif, doit continuer son cycle sur l'ancienne plateforme — changer de
  prestataire au milieu d'un cycle de validation n'annule pas ce cycle.
- **Un test des flux réels reporté après la bascule plutôt qu'avant.**
  Vérifier l'adressage, la réception et l'export vers son logiciel de gestion
  sur la nouvelle plateforme, avant qu'elle devienne la seule en service,
  évite de découvrir un défaut de configuration au moment où il n'y a plus de
  filet de sécurité.

## Où gofact se situe

gofact ne remplace pas cette procédure réglementaire et n'a pas vocation à la
simplifier davantage que ce que la loi prévoit déjà : **gofact n'est pas une
plateforme agréée**, et le changement de PA reste une démarche entre vous et
les deux plateformes concernées.

Ce que gofact change, c'est le point de départ de la question. Vos Factur-X
existent, en clair, dans un dossier que vous possédez déjà, avant même leur
dépôt sur une plateforme et indépendamment d'elle. Changer de plateforme
change le tuyau qui achemine le fichier vers vos partenaires et vers
l'administration ; le fichier lui-même — un PDF/A-3 avec son XML CII embarqué
verbatim — n'a jamais dépendu de cette plateforme pour exister ou rester
lisible. **gofact produit le fichier ; votre plateforme agréée le
transporte.** Le jour où vous changez de plateforme, cette phrase reste
vraie : seul le second tuyau change.

## Ce que vous pouvez vérifier avant de changer

1. **Demandez par écrit à votre plateforme actuelle les modalités exactes de
   portabilité** — délais, format d'export, durée du service minimal en
   vigueur à la date de votre demande. La loi les encadre, mais chaque
   plateforme documente sa procédure différemment.
2. **Vérifiez que vos adresses de routage sont mises à jour dans l'annuaire
   au moment de la bascule**, pas après — un décalage de quelques jours suffit
   à égarer une facture qui vous est destinée.
3. **Gardez, de votre côté, une copie indépendante de vos Factur-X**, hors de
   l'interface de la plateforme. C'est la seule chose qu'aucune procédure de
   portabilité ne peut vous retirer, quelle que soit l'issue du changement.

---

## Sources

- [Décret n° 2026-677 du 27 juillet 2026 relatif à la généralisation de la facturation électronique — Légifrance](https://www.legifrance.gouv.fr/jorf/id/JORFTEXT000054499487)
- [Facturation électronique et plateformes agréées — impots.gouv.fr](https://www.impots.gouv.fr/facturation-electronique-et-plateformes-agreees)
- [Je consulte la liste des plateformes agréées — impots.gouv.fr](https://www.impots.gouv.fr/je-consulte-la-liste-des-plateformes-agreees)

*Cet article décrit un dispositif général. Il ne constitue pas un conseil
fiscal ou juridique. Pour votre situation particulière, adressez-vous à votre
expert-comptable ou à votre plateforme agréée.*
