# Garde-fous éditoriaux

> À lire **intégralement** avant toute production de contenu, y compris par une
> session automatisée. Ces règles priment sur le calendrier, sur le style et
> sur l'envie de publier.

---

## 1. Affirmations réglementaires

Le sujet est juridique. Une erreur factuelle publiée sous le nom de gofact
coûte la crédibilité du projet entier, et potentiellement de l'argent à
quelqu'un.

### Faits établis, réutilisables sans revérification

| Fait | Statut |
| --- | --- |
| 1er sept. 2026 — obligation de **réception** pour toutes les entreprises assujetties à la TVA | établi |
| 1er sept. 2026 — obligation d'**émission** pour grandes entreprises et ETI | établi |
| 1er sept. 2027 — obligation d'**émission** pour PME, TPE et micro-entreprises | établi |
| « Plateforme agréée » (PA) est la dénomination officielle qui remplace « PDP » | établi |
| Le PPF conserve deux missions : annuaire central et concentrateur de données | établi |
| Formats structurés admis : Factur-X, UBL, CII | établi |
| L'obligation couvre le B2B entre assujettis TVA ; le B2C et l'international relèvent de l'e-reporting | établi |

### Règles

1. **Tout chiffre, toute date, tout nom de dispositif est revérifié** à une
   source primaire ou de premier rang (impots.gouv.fr, data.gouv.fr, texte
   légal) avant publication, sauf s'il figure dans le tableau ci-dessus.
2. **Le nombre de plateformes agréées évolue.** Ne jamais citer « 166 » ou tout
   autre nombre sans le revérifier le jour même sur
   `data.gouv.fr/datasets/liste-des-plateformes-agreees-dgfip-pour-la-facturation-electronique`.
3. **Aucun conseil fiscal ou juridique individualisé.** On explique un
   dispositif, on ne dit jamais à quelqu'un ce qu'il doit faire dans sa
   situation. Renvoyer à son expert-comptable.
4. **En cas de doute non levé, on ne publie pas.** On publie le sujet suivant
   du calendrier et on signale le doute dans le journal de production.

---

## 2. Affirmations produit

### Interdit, sans exception

- Dire ou laisser entendre que **gofact est une plateforme agréée**, ou qu'il
  dispense de passer par une PA.
- Dire que gofact **transmet à la DGFiP**. Il dépose sur la PA de
  l'utilisateur ; c'est elle qui transmet.
- Promettre une conformité qui n'est pas vérifiée par le code. Ce qui est
  vérifié : règles métier EN 16931 en amont, structures Factur-X et intégrité
  du XML en aval, PDF/A-3b par veraPDF en CI.
- Annoncer une fonctionnalité non fusionnée sur `main`.
- Publier un chiffre d'usage (téléchargements, utilisateurs) non mesuré.

### Formulation de référence

> gofact produit le fichier ; votre plateforme agréée le transporte.

Cette phrase, ou un équivalent explicite, doit apparaître dans **tout** contenu
qui touche à la transmission.

---

## 3. Style

Hérité du positionnement de l'auteur et de la documentation existante :
technique, sobre, sans emphase.

**Interdits de vocabulaire :** « révolutionnaire », « game changer »,
« disruptif », « hack », « secret », « astuce », « booster », « transformer »,
« la solution ultime », « enfin une alternative », tout superlatif sur le
produit.

**Règles de forme :**

- Phrases courtes. Un paragraphe ne dépasse pas deux lignes sur mobile.
- Pas d'émoji dans le corps d'un texte. Un maximum en accroche, et seulement
  s'il porte du sens.
- Ne pas ouvrir par « Je », sauf avis personnel assumé.
- LinkedIn : 800 à 1500 signes, 3 à 5 hashtags en fin de post, jamais dans le
  corps.
- Articles de fond : titre qui répond à une question réelle, réponse dans les
  trois premières lignes, pas de suspense.

**Jamais inventer.** Ni anecdote, ni citation, ni retour d'utilisateur, ni
chiffre. Si une accroche appelle du vécu qui n'existe pas, utiliser une
formulation générique (« on voit souvent… ») ou changer d'accroche.

---

## 4. Ton vis-à-vis du marché

On ne nomme pas de concurrent pour le critiquer. On compare des **approches** :
l'abonnement contre le fichier local, le service géré contre l'outil qu'on
possède. Chaque approche a des avantages réels, et on les cite — un comparatif
qui ne concède rien n'est pas lu comme honnête, parce qu'il ne l'est pas.

Quand un contenu explique qu'une plateforme agréée est nécessaire, il ne le
présente pas comme une contrainte subie mais comme le fonctionnement du
dispositif.

---

## 5. Publication automatisée

Une session automatisée applique en plus :

- **Rien ne part sans passer par la file `handoff/comms/queue/`.** Pas de rédaction
  directe vers un canal.
- **Un contenu publié est déplacé dans `handoff/comms/published/`** avec sa date et son
  URL. Cet historique est la mémoire de la machine : il évite de republier deux
  fois le même angle.
- **Fenêtre horaire :** publication sociale entre 7h et 19h, heure de Paris,
  du lundi au vendredi. Jamais le week-end, jamais la nuit.
- **Plafond dur :** 3 posts LinkedIn par semaine, 1 post Reddit ou HN par mois.
  Le plafond n'est pas une cible à atteindre — sauter une publication faute de
  bon sujet est un résultat acceptable, et attendu.
- **Aucun message privé automatisé.**
- **En cas d'échec de publication**, ne jamais réessayer plus de deux fois, et
  notifier plutôt que d'insister.
