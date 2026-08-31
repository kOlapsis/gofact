package facturx

import (
	"fmt"
	"strings"
)

// Règles métier EN 16931 vérifiées sur le modèle, avant sérialisation du XML.
//
// C'est délibérément l'inverse de la démarche habituelle : plutôt que de
// produire un XML puis de le soumettre à un validateur, on refuse de produire un
// document qu'on sait non conforme. L'erreur arrive au moment où elle est encore
// réparable, et porte un identifiant de règle que l'appelant — un humain ou une
// IA — peut relayer tel quel.
//
// La couverture n'est pas exhaustive : EN 16931 compte plusieurs centaines de
// règles, dont beaucoup portent sur des structures que gofact n'émet pas. Sont
// implémentées ici celles qui encadrent ce que gofact produit réellement et que
// des données d'entrée erronées peuvent violer. La vérification exhaustive reste
// le rôle de veraPDF et du Schematron en intégration continue.

// RuleViolation est une règle EN 16931 non respectée.
type RuleViolation struct {
	ID      string // identifiant normatif, ex. « BR-06 »
	Message string
}

func (v RuleViolation) Error() string { return v.ID + " : " + v.Message }

// RuleError agrège les violations d'un document.
type RuleError struct{ Violations []RuleViolation }

func (e *RuleError) Error() string {
	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Error())
	}
	return "facture non conforme EN 16931 :\n  - " + strings.Join(parts, "\n  - ")
}

// Validate applique les règles métier à une facture. Renvoie une *RuleError si
// au moins une règle est violée.
func (inv Invoice) Validate() error {
	var v []RuleViolation
	add := func(id, format string, a ...any) {
		v = append(v, RuleViolation{ID: id, Message: fmt.Sprintf(format, a...)})
	}

	// Identification du document.
	if strings.TrimSpace(inv.Number) == "" {
		add("BR-02", "la facture doit porter un numéro (BT-1)")
	}
	if inv.IssueDate.IsZero() {
		add("BR-03", "la facture doit porter une date d'émission (BT-2)")
	}
	if inv.Type == "" {
		add("BR-04", "la facture doit porter un code de type de document (BT-3)")
	}
	if strings.TrimSpace(inv.Currency) == "" {
		add("BR-05", "la facture doit porter un code de devise (BT-5)")
	}

	// Parties.
	if strings.TrimSpace(inv.Seller.Name) == "" {
		add("BR-06", "le vendeur doit avoir un nom (BT-27)")
	}
	if strings.TrimSpace(inv.Buyer.Name) == "" {
		add("BR-07", "l'acheteur doit avoir un nom (BT-44)")
	}
	if strings.TrimSpace(inv.Seller.CountryCode) == "" {
		add("BR-09", "l'adresse du vendeur doit porter un code pays (BT-40)")
	}
	if strings.TrimSpace(inv.Buyer.CountryCode) == "" {
		add("BR-11", "l'adresse de l'acheteur doit porter un code pays (BT-55)")
	}

	// Lignes.
	if len(inv.Lines) == 0 {
		add("BR-16", "la facture doit comporter au moins une ligne")
	}
	for _, l := range inv.Lines {
		if strings.TrimSpace(l.Name) == "" {
			add("BR-25", "la ligne %s doit porter une dénomination (BT-153)", l.ID)
		}
		if strings.TrimSpace(l.Quantity) == "" {
			add("BR-22", "la ligne %s doit porter une quantité (BT-129)", l.ID)
		}
		if l.UnitCode == "" {
			add("BR-23", "la ligne %s doit porter une unité de mesure (BT-130)", l.ID)
		}
	}

	// Cohérence des totaux : c'est ici que se logent les erreurs d'arrondi.
	var sumLines int64
	for _, l := range inv.Lines {
		sumLines += l.LineTotal
	}
	if sumLines != inv.LineTotal {
		add("BR-CO-10", "la somme des lignes (%d) doit égaler le total HT BT-106 (%d)", sumLines, inv.LineTotal)
	}
	if inv.TaxBasisTotal+inv.TaxTotal != inv.GrandTotal {
		add("BR-CO-15", "le total TTC BT-112 (%d) doit égaler BT-109 + BT-110 (%d + %d)",
			inv.GrandTotal, inv.TaxBasisTotal, inv.TaxTotal)
	}
	if inv.DuePayable != inv.GrandTotal {
		add("BR-CO-16", "le net à payer BT-115 (%d) doit égaler le total TTC BT-112 (%d)",
			inv.DuePayable, inv.GrandTotal)
	}

	// Ventilation de TVA.
	if len(inv.VATBreakdown) == 0 {
		add("BR-12", "la facture doit comporter une ventilation de TVA (BG-23)")
	}
	var sumBasis, sumTax int64
	for _, s := range inv.VATBreakdown {
		sumBasis += s.BasisAmount
		sumTax += s.CalculatedTax
		if s.CategoryCode == "" {
			add("BR-47", "chaque ventilation de TVA doit porter un code de catégorie (BT-118)")
		}
		// Catégorie E : exonéré. La mention et le motif sont obligatoires.
		if s.CategoryCode == "E" {
			if s.RatePct != "0.00" {
				add("BR-E-05", "en catégorie exonérée le taux (BT-119) doit être zéro, trouvé %s", s.RatePct)
			}
			if s.CalculatedTax != 0 {
				add("BR-E-09", "en catégorie exonérée la TVA calculée (BT-117) doit être zéro")
			}
			if strings.TrimSpace(s.ExemptionReason) == "" && strings.TrimSpace(s.ExemptionReasonCode) == "" {
				add("BR-E-10", "une ventilation exonérée doit porter un motif d'exonération (BT-120 ou BT-121)")
			}
		}
	}
	if len(inv.VATBreakdown) > 0 && sumBasis != inv.TaxBasisTotal {
		add("BR-CO-13", "la somme des bases de TVA (%d) doit égaler BT-109 (%d)", sumBasis, inv.TaxBasisTotal)
	}
	if len(inv.VATBreakdown) > 0 && sumTax != inv.TaxTotal {
		add("BR-CO-14", "la somme des TVA calculées (%d) doit égaler BT-110 (%d)", sumTax, inv.TaxTotal)
	}

	// Moyen de paiement : annoncer un virement (30) ou un prélèvement SEPA (58)
	// sans donner le compte à créditer rend la facture inexploitable, et le
	// Schematron Factur-X la rejette (FX-SCH-A-000132, BR-DE-23-a).
	if inv.PaymentMeans == "30" || inv.PaymentMeans == "58" {
		if strings.TrimSpace(inv.PayeeIBAN) == "" {
			add("BR-50", "moyen de paiement %s (virement) : l'IBAN du compte de règlement (BT-84) "+
				"est requis — renseignez %s", inv.PaymentMeans, envPayeeIBAN)
		}
	}

	// Un vendeur sans numéro de TVA doit porter un identifiant fiscal, faute de
	// quoi une facture exonérée n'est pas recevable.
	if inv.Seller.VATNumber == "" && inv.Seller.TaxID == "" && hasExemptCategory(inv) {
		add("BR-E-02", "un vendeur exonéré sans n° de TVA (BT-31) doit porter un identifiant fiscal (BT-32)")
	}

	// Devise de TVA différente de la devise du document : la contre-valeur en
	// devise de comptabilisation devient obligatoire.
	if inv.Currency != "" && !strings.EqualFold(inv.Currency, "EUR") && inv.TaxTotal != 0 && inv.VATEur == 0 {
		add("BR-53", "devise %s : la contre-valeur de TVA en EUR (BT-111) est requise", inv.Currency)
	}

	if len(v) == 0 {
		return nil
	}
	return &RuleError{Violations: v}
}

func hasExemptCategory(inv Invoice) bool {
	for _, s := range inv.VATBreakdown {
		if s.CategoryCode == "E" {
			return true
		}
	}
	return false
}
