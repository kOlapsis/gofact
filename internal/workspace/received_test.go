package workspace

import (
	"testing"

	"github.com/kolapsis/gofact/internal/facturx"
)

func TestSplitByRateKeepsTheExactTotal(t *testing.T) {
	breakdown := []facturx.VATSubtotal{
		{RatePct: "20.00", BasisAmount: 10000, CalculatedTax: 2000},
		{RatePct: "5.50", BasisAmount: 3333, CalculatedTax: 183},
	}
	parts := splitByRate(10001, breakdown)
	if len(parts) != 2 || parts[0].VATRate != "20.00" || parts[1].VATRate != "5.50" {
		t.Fatalf("ventilation inattendue : %+v", parts)
	}
	if parts[0].Amount != "77.35" || parts[1].Amount != "22.66" {
		t.Errorf("parts = %s + %s, attendu 77.35 + 22.66 (somme 100.01)", parts[0].Amount, parts[1].Amount)
	}
	if one := splitByRate(59000, []facturx.VATSubtotal{{RatePct: "0.00", BasisAmount: 59000}}); one[0].Amount != "590.00" {
		t.Errorf("taux unique : %+v", one)
	}
}

func TestSafeName(t *testing.T) {
	if got := safeName(`2026-08-30 - Fournisseur: B - B/12. `); got != "2026-08-30 - Fournisseur_ B - B_12" {
		t.Errorf("safeName = %q", got)
	}
}
