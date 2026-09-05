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

// noteCodes renvoie les codes sujet des notes, dans l'ordre d'émission.
func noteCodes(inv Invoice) []string {
	out := make([]string, 0, len(inv.Notes))
	for _, n := range inv.Notes {
		out = append(out, n.SubjectCode)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Une note de contexte s'AJOUTE aux mentions légales, elle ne les remplace pas.
// C'est le défaut qui a fait rejeter la facture 2026015 (BR-FR-05).
func TestNotesWithoutCodeKeepLegalMentions(t *testing.T) {
	setSellerEnv(t)
	spec := franchiseSpec()
	spec.Notes = []NoteSpec{{Content: "Prestation réalisée dans le cadre du projet Atlas."}}
	inv, err := spec.ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}
	if got := noteCodes(inv); !equalStrings(got, []string{"PMD", "PMT", "AAB", "AAI"}) {
		t.Fatalf("codes des notes = %v, want [PMD PMT AAB AAI]", got)
	}
	if inv.Notes[3].Content != "Prestation réalisée dans le cadre du projet Atlas." {
		t.Errorf("la note de contexte doit être conservée telle quelle : %q", inv.Notes[3].Content)
	}
	if err := inv.Validate(); err != nil {
		t.Errorf("la facture fusionnée doit être conforme : %v", err)
	}
	xml, err := BuildCII(inv)
	if err != nil {
		t.Fatalf("BuildCII: %v", err)
	}
	for _, code := range []string{"PMD", "PMT", "AAB", "AAI"} {
		if !strings.Contains(string(xml), "<ram:SubjectCode>"+code+"</ram:SubjectCode>") {
			t.Errorf("le CII doit porter une note %s", code)
		}
	}
}

// Une note du spec qui porte un code légal REMPLACE la mention par défaut du
// même code — personnalisation possible, duplication impossible.
func TestNotesWithLegalCodeReplaceDefault(t *testing.T) {
	setSellerEnv(t)
	spec := franchiseSpec()
	spec.Notes = []NoteSpec{
		{SubjectCode: "pmd", Content: "Pénalités de retard : 10 % l'an."},
		{SubjectCode: "ABL", Content: "Livraison sur site."},
	}
	inv, err := spec.ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}
	if got := noteCodes(inv); !equalStrings(got, []string{"PMD", "PMT", "AAB", "ABL"}) {
		t.Fatalf("codes des notes = %v, want [PMD PMT AAB ABL]", got)
	}
	if inv.Notes[0].Content != "Pénalités de retard : 10 % l'an." {
		t.Errorf("la mention PMD doit être celle du spec : %q", inv.Notes[0].Content)
	}
	if inv.Notes[1] != defaultNotes[1] || inv.Notes[2] != defaultNotes[2] {
		t.Error("PMT et AAB doivent rester les mentions par défaut")
	}
}

// Sans notes dans le spec, les trois mentions par défaut, et rien d'autre.
func TestNotesDefaultWhenSpecHasNone(t *testing.T) {
	setSellerEnv(t)
	inv, err := franchiseSpec().ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}
	if got := noteCodes(inv); !equalStrings(got, []string{"PMD", "PMT", "AAB"}) {
		t.Errorf("codes des notes = %v, want [PMD PMT AAB]", got)
	}
	// Les notes vides sont ignorées, elles ne créent pas de AAI fantôme.
	spec := franchiseSpec()
	spec.Notes = []NoteSpec{{Content: "   "}}
	inv, _ = spec.ToInvoice()
	if len(inv.Notes) != 3 {
		t.Errorf("une note vide ne doit rien ajouter : %v", noteCodes(inv))
	}
}

// Aucune spec ne peut produire un document privé d'une mention légale : les
// combinaisons de notes possibles passent toutes la règle BR-FR-05.
func TestNotesNeverDropLegalMentions(t *testing.T) {
	setSellerEnv(t)
	cases := [][]NoteSpec{
		nil,
		{{Content: "contexte"}},
		{{Content: "a"}, {Content: "b", SubjectCode: "AAI"}},
		{{Content: "x", SubjectCode: "PMD"}, {Content: "y", SubjectCode: "PMT"}, {Content: "z", SubjectCode: "AAB"}},
		{{Content: "x", SubjectCode: "PMD"}, {Content: "x2", SubjectCode: "PMD"}},
	}
	for i, notes := range cases {
		spec := franchiseSpec()
		spec.Notes = notes
		inv, err := spec.ToInvoice()
		if err != nil {
			t.Fatalf("cas %d: ToInvoice: %v", i, err)
		}
		if err := inv.Validate(); err != nil {
			t.Errorf("cas %d: %v", i, err)
		}
	}
}

// BT-49 : sans adresse électronique déclarée, l'acheteur est routé sur son
// SIREN (scheme 0225), comme le vendeur — jamais sur son e-mail, que
// l'annuaire ne connaît pas.
func TestBuyerRoutingDefaultsToSIREN(t *testing.T) {
	setSellerEnv(t)
	spec := franchiseSpec()
	spec.Buyer.Email = "compta@acme.example" // SIRET 55208131700015, pas d'adresse déclarée
	inv, err := spec.ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}
	if inv.Buyer.EAddr != "552081317" || inv.Buyer.EAddrScheme != "0225" {
		t.Fatalf("routage acheteur = %q/%q, want 552081317/0225", inv.Buyer.EAddr, inv.Buyer.EAddrScheme)
	}
	xml, err := BuildCII(inv)
	if err != nil {
		t.Fatalf("BuildCII: %v", err)
	}
	buyer := buyerBlock(t, string(xml))
	if !strings.Contains(buyer, `schemeID="0225">552081317<`) {
		t.Errorf("BuyerTradeParty doit porter URIID 0225:552081317 :\n%s", buyer)
	}
	if strings.Contains(buyer, `schemeID="EM"`) {
		t.Errorf("BuyerTradeParty ne doit pas être routé sur l'e-mail :\n%s", buyer)
	}
}

// Sans aucun identifiant légal, l'e-mail reste le dernier repli (scheme EM).
func TestBuyerRoutingFallsBackToEmailWithoutLegalID(t *testing.T) {
	setSellerEnv(t)
	spec := franchiseSpec()
	spec.Buyer.SIRET, spec.Buyer.SIREN = "", ""
	spec.Buyer.Email = "compta@acme.example"
	inv, err := spec.ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}
	if inv.Buyer.EAddr != "compta@acme.example" || inv.Buyer.EAddrScheme != "EM" {
		t.Errorf("routage acheteur = %q/%q, want compta@acme.example/EM", inv.Buyer.EAddr, inv.Buyer.EAddrScheme)
	}
}

// Une adresse déclarée l'emporte toujours, avec son schéma.
func TestBuyerRoutingExplicitAddressWins(t *testing.T) {
	setSellerEnv(t)
	spec := franchiseSpec()
	spec.Buyer.EAddr, spec.Buyer.EAddrSchema = "552081317", "0002"
	inv, err := spec.ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}
	if inv.Buyer.EAddr != "552081317" || inv.Buyer.EAddrScheme != "0002" {
		t.Errorf("routage acheteur = %q/%q, want 552081317/0002", inv.Buyer.EAddr, inv.Buyer.EAddrScheme)
	}
	// Adresse déclarée sans schéma : un identifiant est en 0225, un e-mail en EM.
	if a, s := ResolveRouting(PartySpec{EAddr: "552081317"}); a != "552081317" || s != "0225" {
		t.Errorf("identifiant sans schéma = %q/%q, want 552081317/0225", a, s)
	}
	if a, s := ResolveRouting(PartySpec{EAddr: "x@y.example", SIRET: "55208131700015"}); a != "x@y.example" || s != "EM" {
		t.Errorf("e-mail déclaré sans schéma = %q/%q, want x@y.example/EM", a, s)
	}
	if a, s := ResolveRouting(PartySpec{Name: "Sans rien"}); a != "" || s != "" {
		t.Errorf("partie sans identifiant ni e-mail = %q/%q, want vide", a, s)
	}
}

// Régression : la spec de la facture 2026015 telle qu'envoyée la première fois
// — acheteur avec SIRET et e-mail mais sans adresse électronique, une note de
// contexte sans code. Rejetée par la PDP (BR-FR-05, BT-49 en EM). Elle doit
// désormais produire un CII avec les trois mentions légales, la note en AAI et
// un acheteur routé en 0225 sur son SIREN.
func TestRegressionInvoice2026015(t *testing.T) {
	setSellerEnv(t)
	spec := Spec{
		Number:    "2026015",
		IssueDate: "2026-09-05",
		Buyer: PartySpec{
			Name: "Client Exemple SAS", SIRET: "55208131700015", Email: "compta@client.example",
			Address: "1 rue de la Paix", PostalCode: "75002", City: "Paris",
		},
		Lines: []LineSpec{
			{Name: "Préparation du disque additionnel et migration des données", Unit: "day", Quantity: "1.00", UnitPrice: 60000},
		},
		Notes: []NoteSpec{{Content: "Intervention réalisée à distance le 4 septembre 2026, à la demande du client."}},
	}
	inv, err := spec.ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}
	if err := inv.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	xml, err := BuildCII(inv)
	if err != nil {
		t.Fatalf("BuildCII: %v", err)
	}
	s := string(xml)
	for _, w := range []string{
		`<ram:SubjectCode>PMD</ram:SubjectCode>`,
		`<ram:SubjectCode>PMT</ram:SubjectCode>`,
		`<ram:SubjectCode>AAB</ram:SubjectCode>`,
		`<ram:SubjectCode>AAI</ram:SubjectCode>`,
		`Intervention réalisée à distance`,
	} {
		if !strings.Contains(s, w) {
			t.Errorf("CII sans %q", w)
		}
	}
	buyer := buyerBlock(t, s)
	if !strings.Contains(buyer, `schemeID="0225">552081317<`) || strings.Contains(buyer, `schemeID="EM"`) {
		t.Errorf("BT-49 attendu 0225:552081317, sans EM :\n%s", buyer)
	}
}

// buyerBlock isole l'élément BuyerTradeParty du XML.
func buyerBlock(t *testing.T, xml string) string {
	t.Helper()
	start := strings.Index(xml, "<ram:BuyerTradeParty>")
	end := strings.Index(xml, "</ram:BuyerTradeParty>")
	if start < 0 || end < 0 {
		t.Fatal("BuyerTradeParty absent du CII")
	}
	return xml[start:end]
}
