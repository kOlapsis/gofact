# Conformité

Une facture électronique française doit être un **Factur-X** au profil EN 16931 : un
PDF/A-3 (archivage ISO 19005-3) embarquant la facture structurée en XML CII. Voici
comment gofact y arrive, et comment il le prouve.

## La chaîne

```
Spec JSON ─ règles EN 16931 ─▶ refus si non conforme (avant toute production)
          ─ gofact ──────────▶ factur-x.xml (CII, profil EN 16931)
HTML ────── navigateur ──────▶ PDF (Blink/Skia, fidèle à l'impression)
PDF + XML ─ Go pur (pdfcpu) ─▶ Factur-X : PDF/A-3 + XML embarqué VERBATIM
Factur-X ── auto-contrôle ───▶ relecture du fichier écrit
```

L'assemblage PDF/A-3 est fait en Go : fichier embarqué (`/AF`, relation `Alternative`),
OutputIntent sRGB (profil ICC embarqué dans le binaire), paquet XMP identifiant
PDF/A-3b avec le schéma d'extension Factur-X, `/ID` de trailer. Le rendu du navigateur
n'est **pas retouché** — ni reconversion colorimétrique, ni ré-encodage de polices.

## Le XML n'est jamais re-sérialisé

Le XML CII est inséré **octet pour octet**. Un assembleur qui le ferait transiter par
son propre modèle perdrait des champs étendus — notamment les adresses électroniques de
routage (BT-34/BT-49) dont dépendent les PDP. C'est vérifié à chaque génération :
l'auto-contrôle relit le PDF écrit et compare le XML extrait, octet pour octet, à celui
qui a été produit.

## Deux contrôles, deux moments

**Avant d'émettre — les règles métier EN 16931.** Numéro, dates, identités, cohérence
des totaux (`BR-CO-*`), ventilation de TVA, motif d'exonération, IBAN exigé dès que le
paiement annoncé est un virement (`BR-50`)… gofact **refuse de produire** une facture
qu'il sait non conforme, en nommant la règle et le champ fautif — au moment où l'erreur
est encore corrigeable.

**Après avoir écrit — l'auto-contrôle structurel.** Le fichier produit est relu :
présence et forme de `/AF`, `/OutputIntents`, du XMP et de ses propriétés `fx:`, de
l'arbre `EmbeddedFiles`, intégrité du XML. Quelques millisecondes, aucune dépendance.

## La preuve externe : veraPDF en intégration continue

Un auto-contrôle ne suffit pas à un document fiscal : il faut un juge indépendant. À
chaque évolution du code, la CI fait valider une facture réelle par
**[Mustang](https://www.mustangproject.org/)** — l'implémentation de référence, qui
embarque **veraPDF** (validation PDF/A complète) et le **Schematron EN 16931 officiel**.

Verdict exigé : `PDF:valid XML:valid`, veraPDF `flavour 3b, isCompliant=true`, zéro
assertion en échec. Ce juge tourne en CI **uniquement** : jamais de Java, jamais de
téléchargement chez un utilisateur.

```sh
# Pour le reproduire (Java 11+ requis) :
go test -tags=ci ./internal/facturx -run TestOracle -v
```

## Ce que gofact ne couvre pas (encore)

- **Mono-taux de TVA** — suffisant en franchise ou au taux unique ; le multi-taux par
  facture n'est pas géré.
- **Devise ≠ EUR** : la contre-valeur de TVA en euros (BT-111) doit être fournie en
  amont ; la règle `BR-53` bloque plutôt que de laisser passer.
- **L'avoir** (`credit_note`) est géré ; les cas complexes (remises globales, acomptes
  déduits ligne à ligne au sens BG-20/21) ne le sont pas.
