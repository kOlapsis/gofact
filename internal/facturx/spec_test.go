package facturx

import (
	"strings"
	"testing"
)

// Identité de test — entièrement fictive : gofact ne code aucune identité en dur,
// elle vient de l'environnement (GOFACT_SELLER_*) ou du bloc "seller" du JSON.
const (
	testSellerSIRET = "12345678900014"
	testSellerSIREN = "123456789"
	testIBAN        = "FR7630001007941234567890185"
)

// setSellerEnv configure le vendeur par défaut pour la durée du test.
func setSellerEnv(t *testing.T) {
	t.Helper()
	t.Setenv(envSellerName, "Studio Exemple")
	t.Setenv(envSellerSIRET, testSellerSIRET)
	t.Setenv(envSellerEmail, "contact@exemple.test")
	t.Setenv(envSellerAddress, "1 rue de l'Exemple")
	t.Setenv(envSellerPostalCode, "33000")
	t.Setenv(envSellerCity, "Bordeaux")
	t.Setenv(envPayeeIBAN, testIBAN)
}

// franchiseSpec est une facture franchise 293 B minimale.
func franchiseSpec() Spec {
	return Spec{
		Number:    "2026011",
		IssueDate: "2026-06-29",
		Buyer: PartySpec{
			Name: "ACME SAS", SIRET: "55208131700015",
			Address: "1 rue de la Paix", PostalCode: "75002", City: "Paris",
		},
		Lines: []LineSpec{
			{Name: "Prestation", Unit: "day", Quantity: "2.00", UnitPrice: 60000, Amount: 120000},
		},
	}
}

func TestToInvoiceFranchiseDefaults(t *testing.T) {
	setSellerEnv(t)
	inv, err := franchiseSpec().ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}

	if inv.Type != DocInvoice {
		t.Errorf("Type = %q, want 380", inv.Type)
	}
	// Échéance « À réception » ⇒ = émission ; livraison ⇒ = émission.
	if !inv.DueDate.Equal(inv.IssueDate) {
		t.Errorf("DueDate %v != IssueDate %v", inv.DueDate, inv.IssueDate)
	}
	if inv.DeliveryDate.IsZero() {
		t.Error("DeliveryDate ne doit pas être nulle (défaut = émission)")
	}
	// Totaux exonérés.
	if inv.LineTotal != 120000 || inv.TaxTotal != 0 || inv.GrandTotal != 120000 {
		t.Errorf("totaux = HT %d / TVA %d / TTC %d", inv.LineTotal, inv.TaxTotal, inv.GrandTotal)
	}
	// Vendeur : franchise ⇒ pas de n° TVA mais BT-32 (FC) = SIREN.
	if inv.Seller.VATNumber != "" {
		t.Errorf("vendeur ne doit pas avoir de n° TVA, a %q", inv.Seller.VATNumber)
	}
	if inv.Seller.TaxID != testSellerSIREN {
		t.Errorf("Seller.TaxID = %q, want %q", inv.Seller.TaxID, testSellerSIREN)
	}
	// Adresse de routage (BT-34) dérivée du SIREN à défaut d'être déclarée.
	if inv.Seller.EAddr != testSellerSIREN || inv.Seller.EAddrScheme != defaultEAddrScheme {
		t.Errorf("routage vendeur = %q/%q, want %q/%q",
			inv.Seller.EAddr, inv.Seller.EAddrScheme, testSellerSIREN, defaultEAddrScheme)
	}
	// IBAN de règlement issu de l'environnement.
	if inv.PayeeIBAN != testIBAN {
		t.Errorf("IBAN = %q, want %q", inv.PayeeIBAN, testIBAN)
	}
	// Sous-total TVA exonéré avec code VATEX.
	if len(inv.VATBreakdown) != 1 || inv.VATBreakdown[0].CategoryCode != "E" ||
		inv.VATBreakdown[0].ExemptionReasonCode != fallbackVATExCode {
		t.Errorf("breakdown TVA inattendu: %+v", inv.VATBreakdown)
	}
	// Mentions légales FR par défaut.
	if len(inv.Notes) != 3 {
		t.Errorf("Notes = %d, want 3 (PMD/PMT/AAB)", len(inv.Notes))
	}
}

func TestBuildCIIFranchiseShape(t *testing.T) {
	setSellerEnv(t)
	inv, err := franchiseSpec().ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}
	xml, err := BuildCII(inv)
	if err != nil {
		t.Fatalf("BuildCII: %v", err)
	}
	s := string(xml)

	wants := []string{
		`urn:cen.eu:en16931:2017`,                // profil EN 16931
		`schemeID="FC">` + testSellerSIREN,       // BT-32 identifiant fiscal
		`<ram:CategoryCode>E</ram:CategoryCode>`, // catégorie exonérée
		`VATEX-FR-FRANCHISE`,                     // code exonération
		`<ram:SubjectCode>PMD</ram:SubjectCode>`, // mention pénalités
		`<ram:ActualDeliverySupplyChainEvent>`,   // date de livraison présente
	}
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Errorf("XML CII ne contient pas %q", w)
		}
	}
	// BT-23 (mode de facturation) émis par défaut : « S1 » (services), requis et
	// contraint par la réforme FR (BR-FR-08).
	if !strings.Contains(s, "<ram:ID>S1</ram:ID>") {
		t.Error("BT-23 (mode de facturation « S1 ») doit être émis par défaut")
	}
	// Aucun élément vide (PEPPOL-EN16931-R008).
	if strings.Contains(s, "></ram:ApplicableHeaderTradeDelivery>") {
		t.Error("ApplicableHeaderTradeDelivery ne doit pas être vide")
	}
}

func TestRequiredFields(t *testing.T) {
	setSellerEnv(t)
	if _, err := (Spec{IssueDate: "2026-06-29"}).ToInvoice(); err == nil {
		t.Error("attendu une erreur pour numéro manquant")
	}
	noLines := franchiseSpec()
	noLines.Lines = nil
	if _, err := noLines.ToInvoice(); err == nil {
		t.Error("attendu une erreur pour lignes manquantes")
	}
}

// Sans identité configurée ni bloc "seller", la génération doit échouer plutôt
// que de retomber sur une identité codée en dur.
func TestSellerRequired(t *testing.T) {
	for _, k := range []string{envSellerName, envSellerSIREN, envSellerSIRET} {
		t.Setenv(k, "")
	}
	if _, err := franchiseSpec().ToInvoice(); err == nil {
		t.Fatal("attendu une erreur pour vendeur non configuré")
	}

	// Le bloc "seller" du JSON supplée l'environnement.
	spec := franchiseSpec()
	spec.Seller = &PartySpec{Name: "Studio Exemple", SIRET: testSellerSIRET}
	inv, err := spec.ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice avec seller explicite: %v", err)
	}
	if inv.Seller.Name != "Studio Exemple" || inv.Seller.SIREN != testSellerSIREN {
		t.Errorf("vendeur = %+v", inv.Seller)
	}
}
