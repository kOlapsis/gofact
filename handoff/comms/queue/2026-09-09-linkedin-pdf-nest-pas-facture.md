---
date: 2026-09-09
heure: "12:30"
canal: linkedin
pilier: B
sujet: Un PDF envoyé par mail n'est pas une facture électronique
statut: prêt
url:
---
Un PDF envoyé par mail n'est pas une facture électronique.

C'est le malentendu numéro un depuis septembre, et il coûtera cher à ceux qui le découvrent tard.

Une facture électronique au sens de la réforme, c'est un fichier structuré. Du XML que la machine du destinataire lit directement : sans OCR, sans reconnaissance de caractères, sans qu'un humain retape quoi que ce soit.

Le format le plus répandu en France s'appelle Factur-X. Sa particularité tient en une phrase : c'est un PDF lisible par un humain, avec le XML normalisé embarqué à l'intérieur. Un seul fichier, deux lectures.

Dans ce XML, la norme EN 16931 impose des champs identifiés. Le total hors taxes porte un nom précis. La TVA aussi. L'identifiant du vendeur aussi. Chacun a une règle de validation, et chaque règle a un code — BR-50, BR-CO-15 — que votre outil devrait être capable de vous citer quand il refuse.

Le reste — la mise en page, le logo, la police — n'a aucune valeur dans le dispositif.

Ce qui veut dire qu'un générateur de factures qui produit un beau PDF ne vous met pas en conformité. Il produit une image de facture.

La question à poser à votre outil actuel n'est donc pas « est-ce qu'il fait des PDF ». C'est : est-ce qu'il embarque le XML CII EN 16931, ou est-ce qu'il dessine ?

#facturx #facturationelectronique #EN16931 #conformite #opensource
