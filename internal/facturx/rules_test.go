package facturx

import (
	"errors"
	"strings"
	"testing"
)

// validInvoice construit une facture minimale mais conforme, servant de base aux
// tests : chaque cas la dégrade sur un seul point.
func validInvoice(t *testing.T) Invoice {
	t.Helper()
	setSellerEnv(t)
	inv, err := franchiseSpec().ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}
	if err := inv.Validate(); err != nil {
		t.Fatalf("la facture de référence doit être conforme : %v", err)
	}
	return inv
}

// violations renvoie les identifiants de règles signalés par Validate.
func violations(t *testing.T, inv Invoice) []string {
	t.Helper()
	err := inv.Validate()
	if err == nil {
		return nil
	}
	var re *RuleError
	if !errors.As(err, &re) {
		t.Fatalf("erreur de type inattendu : %T", err)
	}
	ids := make([]string, 0, len(re.Violations))
	for _, v := range re.Violations {
		ids = append(ids, v.ID)
	}
	return ids
}

func hasRule(ids []string, id string) bool {
	for _, got := range ids {
		if got == id {
			return true
		}
	}
	return false
}

func TestValidateAcceptsConformingInvoice(t *testing.T) {
	inv := validInvoice(t)
	if err := inv.Validate(); err != nil {
		t.Errorf("Validate = %v, want nil", err)
	}
}

func TestValidateRejectsCreditTransferWithoutIBAN(t *testing.T) {
	inv := validInvoice(t)
	inv.PayeeIBAN = ""
	ids := violations(t, inv)
	if !hasRule(ids, "BR-50") {
		t.Errorf("BR-50 attendue pour un virement sans IBAN, obtenu %v", ids)
	}
	// Le message doit nommer la variable à renseigner : il est relayé à
	// l'utilisateur, éventuellement par une IA.
	if err := inv.Validate(); !strings.Contains(err.Error(), envPayeeIBAN) {
		t.Errorf("le message doit nommer %s, obtenu %q", envPayeeIBAN, err)
	}
}

func TestValidateCatchesInconsistentTotals(t *testing.T) {
	inv := validInvoice(t)
	inv.GrandTotal += 100 // TTC ≠ HT + TVA
	ids := violations(t, inv)
	if !hasRule(ids, "BR-CO-15") {
		t.Errorf("BR-CO-15 attendue pour un TTC incohérent, obtenu %v", ids)
	}

	inv = validInvoice(t)
	inv.Lines[0].LineTotal += 1 // somme des lignes ≠ total HT
	if ids := violations(t, inv); !hasRule(ids, "BR-CO-10") {
		t.Errorf("BR-CO-10 attendue pour une somme de lignes incohérente, obtenu %v", ids)
	}
}

func TestValidateRequiresPartyIdentity(t *testing.T) {
	inv := validInvoice(t)
	inv.Buyer.Name = ""
	inv.Seller.CountryCode = ""
	ids := violations(t, inv)
	for _, want := range []string{"BR-07", "BR-09"} {
		if !hasRule(ids, want) {
			t.Errorf("%s attendue, obtenu %v", want, ids)
		}
	}
}

func TestValidateExemptCategoryNeedsReason(t *testing.T) {
	inv := validInvoice(t)
	inv.VATBreakdown[0].ExemptionReason = ""
	inv.VATBreakdown[0].ExemptionReasonCode = ""
	if ids := violations(t, inv); !hasRule(ids, "BR-E-10") {
		t.Errorf("BR-E-10 attendue pour une exonération sans motif, obtenu %v", ids)
	}
}

func TestValidateExemptSellerNeedsTaxID(t *testing.T) {
	inv := validInvoice(t)
	inv.Seller.VATNumber = ""
	inv.Seller.TaxID = ""
	if ids := violations(t, inv); !hasRule(ids, "BR-E-02") {
		t.Errorf("BR-E-02 attendue pour un vendeur exonéré sans identifiant fiscal, obtenu %v", ids)
	}
}
