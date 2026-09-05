# Choisir sa plateforme agréée quand on facture dix fois par an

**Réponse courte.** Si vous émettez une poignée de factures par an, vous n'avez
besoin ni du plan le plus cher, ni d'un outil qui vous oblige à saisir vos
factures directement sur son interface. Cherchez une plateforme agréée qui
propose une formule gratuite ou quasi gratuite pour un faible volume, qui
accepte de recevoir un fichier déjà produit ailleurs, et qui vous laisse
récupérer votre historique le jour où vous en changez.

Le reste de cet article détaille comment vérifier ces trois points avant de
signer quoi que ce soit.

---

## Ce qu'une plateforme agréée fait, et rien de plus

Une plateforme agréée (PA, anciennement PDP) est un opérateur immatriculé par
la DGFiP. Son rôle : acheminer vos factures électroniques vers vos clients et
fournisseurs, et transmettre à l'administration les données fiscales qui en
découlent. C'est un tuyau réglementé, pas un logiciel de facturation.

Le PPF n'achemine plus rien depuis 2024 ; il tient l'annuaire — le registre qui
indique où chaque entreprise reçoit ses factures — et sert de concentrateur
pour les données fiscales. Être déclaré dans cet annuaire, sur une PA de votre
choix, est l'action réellement urgente : c'est elle qui rend vos factures
reçevables.

Ce que ce rôle **n'inclut pas** : produire la facture. La confusion vient du
marché, qui vend presque toujours les deux ensemble — production et
transmission dans le même abonnement — parce que c'est plus simple à
commercialiser. Rien dans la réforme n'oblige à acheter les deux à la même
enseigne.

## Le nombre de plateformes évolue en continu

La liste des plateformes agréées s'allonge chaque mois : de nouvelles
immatriculations sont délivrées régulièrement par la DGFiP, et un nombre cité
dans un article a de bonnes chances d'être déjà dépassé au moment où vous le
lisez. La liste à jour, à la date où vous choisissez, est publiée par la
DGFiP et consolidée sur data.gouv.fr — c'est la seule source à consulter pour
un chiffre exact, jamais un article de blog, celui-ci compris.

## Quatre critères, dans l'ordre où ils comptent pour un petit volume

### 1. Le modèle de prix a-t-il un sens à dix factures par an ?

Beaucoup d'offres facturent un forfait mensuel calibré pour des dizaines ou
centaines de factures. À votre volume, ce forfait paie surtout des
fonctionnalités que vous n'utiliserez jamais. Cherchez explicitement une
formule gratuite ou un tarif au dépôt : elles existent, mais restent
minoritaires dans le marché, et il faut les demander explicitement — elles ne
sont pas toujours mises en avant sur la page tarifs.

### 2. La plateforme accepte-t-elle un fichier que vous avez produit ailleurs ?

C'est le critère le plus souvent oublié, et le plus structurant. Certaines
plateformes n'acceptent que les factures saisies ou importées via leur propre
interface de production ; d'autres acceptent le dépôt d'un fichier Factur-X,
UBL ou CII déjà conforme, produit par l'outil de votre choix.

Si vous voulez garder la main sur votre outil de production — quel qu'il soit
— vérifiez ce point avant de signer. Une plateforme qui n'accepte que sa
propre saisie vous transforme, de fait, en client captif de son interface,
même si le contrat ne dure qu'un mois.

### 3. Que devient votre historique si vous partez ?

L'obligation d'archivage court sur dix ans. Demandez, avant de signer, sous
quel format et par quel moyen vous pourrez exporter l'intégralité de votre
historique de factures le jour où vous changez de plateforme — y compris si
ce jour arrive parce que la plateforme a fermé, pas seulement parce que vous
l'avez décidé.

Une réponse vague ou absente sur ce point est en soi une réponse.

### 4. La déclaration dans l'annuaire est-elle immédiate et lisible ?

Vous devez pouvoir vérifier, depuis votre propre compte, l'adresse à laquelle
vous êtes déclaré dans l'annuaire — et la corriger si un fournisseur vous
signale que ses envois n'arrivent pas. Une plateforme qui ne montre pas cette
information oblige à passer par son support pour un problème que vous
pourriez diagnostiquer seul en trente secondes.

## Ce qui ne devrait pas entrer dans le choix

L'immatriculation elle-même n'est pas un critère différenciant : c'est un
prérequis binaire. Une plateforme immatriculée par la DGFiP répond au cahier
des charges technique et sécuritaire imposé par l'administration ; en
choisir une plutôt qu'une autre ne se joue pas là, mais sur les quatre points
ci-dessus, qui relèvent du contrat et de l'usage, pas de la conformité.

## Où gofact se situe dans ce choix

Pour que ce soit net : **gofact n'est pas une plateforme agréée**. Il ne
transmet rien à l'administration et ne remplace le passage par aucune PA.

Ce que gofact fait : produire, sur votre machine, un Factur-X conforme —
PDF/A-3 avec le XML CII EN 16931 embarqué octet pour octet, règles métier
vérifiées avant l'écriture du fichier, structure recontrôlée après. Ce fichier
est ensuite celui que vous déposez sur la plateforme agréée que *vous* avez
choisie, avec les critères ci-dessus.

Aujourd'hui, gofact sait déposer directement sur une plateforme partenaire
implémentée dans le code ; l'architecture est construite pour en accueillir
d'autres, mais l'intégration reste à écrire au cas par cas. Dans tous les
cas, la production du fichier ne dépend pas de la plateforme retenue : c'est
précisément ce découplage qui vous laisse choisir sur les critères ci-dessus,
et en changer sans réécrire votre historique.

## Ce que vous pouvez vérifier cette semaine

1. **Consultez la liste à jour** des plateformes agréées sur data.gouv.fr, à
   la date du jour — pas un chiffre lu ailleurs.
2. **Contactez deux ou trois candidates** avec les quatre questions
   ci-dessus, par écrit, pour garder une trace des réponses.
3. **Écartez celles qui n'acceptent pas le dépôt d'un fichier externe**, si
   garder votre outil de production compte pour vous.
4. **Vérifiez l'export d'historique** avant de signer, pas après.

Le choix n'a rien d'irréversible dans l'absolu — mais il coûte plus cher à
défaire qu'à faire correctement la première fois.

---

## Sources

- [Facturation électronique et plateformes agréées — impots.gouv.fr](https://www.impots.gouv.fr/facturation-electronique-et-plateformes-agreees)
- [Liste des plateformes agréées DGFiP — data.gouv.fr](https://www.data.gouv.fr/datasets/liste-des-plateformes-agreees-dgfip-pour-la-facturation-electronique)
- [Guide pratique de démarrage au 1er septembre 2026 — impots.gouv.fr (PDF)](https://www.impots.gouv.fr/sites/default/files/media/1_metier/2_professionnel/EV/2_gestion/290_facturation_electronique/guide_pratique_facturation_electronique.pdf)

*Cet article décrit un dispositif général. Il ne constitue pas un conseil
fiscal ou juridique. Pour votre situation particulière, adressez-vous à votre
expert-comptable.*
