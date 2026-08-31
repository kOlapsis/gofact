package facturx

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Spec est le contrat JSON émis par le skill creer-facture à côté du HTML.
// Tous les montants sont en CENTIMES (int64) — jamais de float. Le vendeur,
// l'IBAN de règlement et le régime de TVA viennent de l'environnement
// (variables GOFACT_*, cf. seller.go) : le JSON ne porte que ce qui varie d'une
// facture à l'autre, et peut surcharger n'importe lequel de ces défauts.
type Spec struct {
	Number       string     `json:"number"`           // ex. "2026011" (sans préfixe)
	Type         string     `json:"type"`             // "invoice" (défaut) | "credit_note"
	IssueDate    string     `json:"issue_date"`       // ISO "2026-06-29"
	DueDate      string     `json:"due_date"`         // ISO ; vide ⇒ "À réception" ⇒ = émission
	DeliveryDate string     `json:"delivery_date"`    // ISO ; vide ⇒ omis
	BuyerRef     string     `json:"buyer_reference"`  // BT-10 ; optionnel
	BusinessProc string     `json:"business_process"` // BT-23 ; vide ⇒ défaut profil réforme FR
	Currency     string     `json:"currency"`         // défaut EUR
	Buyer        PartySpec  `json:"buyer"`
	Lines        []LineSpec `json:"lines"`
	VAT          *VATSpec   `json:"vat"`    // défaut : franchise 293 B (exonéré)
	Seller       *PartySpec `json:"seller"` // override ; défaut GOFACT_SELLER_*
	IBAN         string     `json:"iban"`   // override ; défaut GOFACT_PAYEE_IBAN
	Notes        []NoteSpec `json:"notes"`  // override ; défaut : mentions légales FR
}

// NoteSpec est une mention libre (BT-22) avec son code sujet (BT-21).
type NoteSpec struct {
	Content     string `json:"content"`
	SubjectCode string `json:"subject_code"`
}

// PartySpec décrit une partie. SIRET (14) prime pour dériver le SIREN (9) si SIREN absent.
type PartySpec struct {
	Name        string `json:"name"`
	SIREN       string `json:"siren"`
	SIRET       string `json:"siret"`
	VATNumber   string `json:"vat_number"`
	Email       string `json:"email"`
	EAddr       string `json:"electronic_address"`        // BT-34/BT-49 adresse de routage PDP
	EAddrSchema string `json:"electronic_address_scheme"` // ex. "0225" (SIRET FR)
	Address     string `json:"address"`
	PostalCode  string `json:"postal_code"`
	City        string `json:"city"`
	Country     string `json:"country_code"`
}

// LineSpec est une ligne de prestation. Quantity est une chaîne décimale ("9.00").
type LineSpec struct {
	Name      string `json:"name"`
	Unit      string `json:"unit"`                // "day" | "unit" (défaut)
	Quantity  string `json:"quantity"`            // défaut "1.00"
	UnitPrice int64  `json:"unit_price_ht_cents"` // centimes
	Amount    int64  `json:"amount_ht_cents"`     // centimes ; 0 ⇒ = UnitPrice
}

// VATSpec porte le régime de TVA. Par défaut exonéré (franchise 293 B).
type VATSpec struct {
	Exempt        bool   `json:"exempt"`
	Mention       string `json:"mention"`
	ExemptionCode string `json:"exemption_code"`
	RatePct       string `json:"rate_pct"` // requis si Exempt == false (ex. "20.00")
}

// defaultNotes sont les mentions légales obligatoires en France (BR-FR-05) :
// pénalités de retard (PMD), frais de recouvrement (PMT), escompte (AAB).
// Reprend la formulation du template de facture.
var defaultNotes = []Note{
	{SubjectCode: "PMD", Content: "En cas de retard de paiement, application de pénalités au taux annuel de trois fois le taux d'intérêt légal en vigueur (art. L441-10 du Code de commerce), exigibles dès le jour suivant la date d'échéance et sans rappel préalable."},
	{SubjectCode: "PMT", Content: "Indemnité forfaitaire pour frais de recouvrement de 40 € (art. D441-5 du Code de commerce), applicable à chaque facture en retard, sans préjudice d'une indemnisation complémentaire sur justification."},
	{SubjectCode: "AAB", Content: "Pas d'escompte pour paiement anticipé."},
}

// LoadSpec lit et décode un fichier JSON de spécification de facture.
func LoadSpec(path string) (Spec, error) {
	var s Spec
	raw, err := os.ReadFile(path)
	if err != nil {
		return s, fmt.Errorf("facturx: lecture spec %q: %w", path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return s, fmt.Errorf("facturx: JSON spec invalide: %w", err)
	}
	return s, nil
}

// siren9 renvoie un SIREN (9 chiffres) à partir d'un SIREN ou d'un SIRET, en
// retirant les espaces. Vide si rien d'exploitable.
func siren9(siren, siret string) string {
	clean := func(s string) string { return strings.ReplaceAll(strings.TrimSpace(s), " ", "") }
	if s := clean(siren); s != "" {
		return s
	}
	if s := clean(siret); len(s) >= 9 {
		return s[:9]
	}
	return ""
}

func partyToCII(p PartySpec, fallbackCountry string) Party {
	country := strings.TrimSpace(p.Country)
	if country == "" {
		country = fallbackCountry
	}
	return Party{
		Name:        p.Name,
		VATNumber:   strings.ReplaceAll(strings.TrimSpace(p.VATNumber), " ", ""),
		SIREN:       siren9(p.SIREN, p.SIRET),
		Email:       strings.TrimSpace(p.Email),
		EAddr:       strings.TrimSpace(p.EAddr),
		EAddrScheme: strings.TrimSpace(p.EAddrSchema),
		Address:     p.Address,
		PostalCode:  p.PostalCode,
		City:        p.City,
		CountryCode: country,
	}
}

// sellerParty résout le vendeur : bloc "seller" du JSON s'il est fourni, sinon
// identité configurée dans l'environnement. Aucune identité n'est codée en dur.
func sellerParty(s *PartySpec) (Party, error) {
	spec := s
	if spec == nil || strings.TrimSpace(spec.Name) == "" {
		fromEnv := sellerFromEnv()
		if fromEnv.Name == "" {
			return Party{}, fmt.Errorf(
				"facturx: vendeur non configuré — renseignez %s (et les autres GOFACT_SELLER_*) "+
					"dans l'environnement ou un fichier .env, ou fournissez un bloc \"seller\" dans le JSON",
				envSellerName)
		}
		spec = &fromEnv
	}
	p := partyToCII(*spec, "FR")
	// Vendeur exonéré sans n° TVA : BT-32 (scheme FC) = SIREN (BR-E-02).
	if p.VATNumber == "" && p.TaxID == "" {
		p.TaxID = p.SIREN
	}
	return p, nil
}

func parseDate(iso string) (time.Time, error) {
	iso = strings.TrimSpace(iso)
	if iso == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", iso)
}

// parseRateBP convertit un taux "20.00" en points de base (2000) sans float.
func parseRateBP(rate string) (int64, error) {
	rate = strings.TrimSpace(rate)
	if rate == "" {
		return 0, nil
	}
	parts := strings.SplitN(rate, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("taux TVA %q invalide: %w", rate, err)
	}
	frac := int64(0)
	if len(parts) == 2 {
		f := (parts[1] + "00")[:2]
		frac, err = strconv.ParseInt(f, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("taux TVA %q invalide: %w", rate, err)
		}
	}
	return whole*100 + frac, nil
}

func unitCode(unit string) string {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "day", "jour", "j", "tjm":
		return "DAY"
	default:
		return "C62" // unité / forfait
	}
}

// ToInvoice projette la spec (avec ses valeurs par défaut) vers le modèle CII,
// calcule les totaux et le sous-total de TVA. Les montants sont en centimes.
func (s Spec) ToInvoice() (Invoice, error) {
	docType := DocInvoice
	if strings.EqualFold(strings.TrimSpace(s.Type), "credit_note") || s.Type == string(DocCreditNote) {
		docType = DocCreditNote
	}

	issue, err := parseDate(s.IssueDate)
	if err != nil {
		return Invoice{}, fmt.Errorf("facturx: date d'émission: %w", err)
	}
	if issue.IsZero() {
		return Invoice{}, fmt.Errorf("facturx: date d'émission (issue_date) requise")
	}
	due, err := parseDate(s.DueDate)
	if err != nil {
		return Invoice{}, fmt.Errorf("facturx: date d'échéance: %w", err)
	}
	if due.IsZero() {
		due = issue // « À réception »
	}
	delivery, err := parseDate(s.DeliveryDate)
	if err != nil {
		return Invoice{}, fmt.Errorf("facturx: date de livraison: %w", err)
	}
	if delivery.IsZero() {
		delivery = issue // BT-72 : date de réalisation de la prestation = émission par défaut
	}

	currency := strings.TrimSpace(s.Currency)
	if currency == "" {
		currency = "EUR"
	}

	// Régime TVA : exonéré par défaut (franchise 293 B).
	vat := VATSpec{Exempt: true}
	if s.VAT != nil {
		vat = *s.VAT
	}
	category := "E"
	ratePct := "0.00"
	var rateBP int64
	if !vat.Exempt {
		category = "S"
		rateBP, err = parseRateBP(vat.RatePct)
		if err != nil {
			return Invoice{}, fmt.Errorf("facturx: %w", err)
		}
		ratePct = fmt.Sprintf("%d.%02d", rateBP/100, rateBP%100)
	}

	if len(s.Lines) == 0 {
		return Invoice{}, fmt.Errorf("facturx: au moins une ligne requise")
	}

	lines := make([]Line, 0, len(s.Lines))
	var lineTotal int64
	for i, l := range s.Lines {
		amount := l.Amount
		unitPrice := l.UnitPrice
		if amount == 0 {
			amount = unitPrice
		}
		if unitPrice == 0 {
			unitPrice = amount
		}
		qty := strings.TrimSpace(l.Quantity)
		if qty == "" {
			qty = "1.00"
		}
		lines = append(lines, Line{
			ID:          strconv.Itoa(i + 1),
			Name:        l.Name,
			UnitCode:    unitCode(l.Unit),
			Quantity:    qty,
			NetPrice:    unitPrice,
			LineTotal:   amount,
			VATCategory: category,
			VATRatePct:  ratePct,
		})
		lineTotal += amount
	}

	// TVA calculée : 0 si exonéré, sinon base × taux (arrondi au centime).
	taxTotal := int64(0)
	if !vat.Exempt {
		taxTotal = (lineTotal*rateBP + 5000) / 10000
	}

	sub := VATSubtotal{
		CategoryCode:  category,
		RatePct:       ratePct,
		BasisAmount:   lineTotal,
		CalculatedTax: taxTotal,
	}
	if vat.Exempt {
		sub.ExemptionReason = vat.Mention
		if sub.ExemptionReason == "" {
			sub.ExemptionReason = envOr(envVATExemptMention, fallbackVATMention)
		}
		sub.ExemptionReasonCode = vat.ExemptionCode
		if sub.ExemptionReasonCode == "" {
			sub.ExemptionReasonCode = envOr(envVATExemptCode, fallbackVATExCode)
		}
	}

	grand := lineTotal + taxTotal

	seller, err := sellerParty(s.Seller)
	if err != nil {
		return Invoice{}, err
	}

	iban := strings.ReplaceAll(strings.TrimSpace(s.IBAN), " ", "")
	if iban == "" {
		iban = defaultIBAN()
	}

	notes := defaultNotes
	if len(s.Notes) > 0 {
		notes = make([]Note, 0, len(s.Notes))
		for _, n := range s.Notes {
			notes = append(notes, Note{Content: n.Content, SubjectCode: n.SubjectCode})
		}
	}

	inv := Invoice{
		Number:          strings.TrimSpace(s.Number),
		Type:            docType,
		Notes:           notes,
		IssueDate:       issue,
		DueDate:         due,
		DeliveryDate:    delivery,
		BuyerReference:  strings.TrimSpace(s.BuyerRef),
		BusinessProcess: strings.TrimSpace(s.BusinessProc),
		Currency:        currency,
		PaymentMeans:    "30", // virement (UNTDID 4461)
		PayeeIBAN:       iban,
		Seller:          seller,
		Buyer:           partyToCII(s.Buyer, "FR"),
		Lines:           lines,
		VATBreakdown:    []VATSubtotal{sub},
		LineTotal:       lineTotal,
		TaxBasisTotal:   lineTotal,
		TaxTotal:        taxTotal,
		GrandTotal:      grand,
		DuePayable:      grand,
	}
	if inv.Number == "" {
		return Invoice{}, fmt.Errorf("facturx: numéro de facture (number) requis")
	}
	if inv.Buyer.Name == "" {
		return Invoice{}, fmt.Errorf("facturx: nom de l'acheteur (buyer.name) requis")
	}
	return inv, nil
}
