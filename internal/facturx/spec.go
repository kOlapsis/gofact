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
	Number       string     `json:"number,omitempty"`           // ex. "2026011" (sans préfixe)
	Type         string     `json:"type,omitempty"`             // "invoice" (défaut) | "credit_note"
	IssueDate    string     `json:"issue_date,omitempty"`       // ISO "2026-06-29"
	DueDate      string     `json:"due_date,omitempty"`         // ISO ; vide ⇒ "À réception" ⇒ = émission
	DeliveryDate string     `json:"delivery_date,omitempty"`    // ISO ; vide ⇒ omis
	BuyerRef     string     `json:"buyer_reference,omitempty"`  // BT-10 ; optionnel
	BusinessProc string     `json:"business_process,omitempty"` // BT-23 ; vide ⇒ défaut profil réforme FR
	Currency     string     `json:"currency,omitempty"`         // défaut EUR
	Buyer        PartySpec  `json:"buyer"`
	Lines        []LineSpec `json:"lines"`
	VAT          *VATSpec   `json:"vat,omitempty"`    // défaut : franchise 293 B (exonéré)
	Seller       *PartySpec `json:"seller,omitempty"` // override ; défaut GOFACT_SELLER_*
	IBAN         string     `json:"iban,omitempty"`   // override ; défaut GOFACT_PAYEE_IBAN
	// Notes complémentaires (BG-1). Les mentions légales PMD, PMT et AAB sont
	// TOUJOURS présentes : une note du spec qui porte l'un de ces codes remplace
	// la mention par défaut, une note sans code est une information générale (AAI).
	Notes []NoteSpec `json:"notes,omitempty" jsonschema:"notes complémentaires (BG-1) ; les mentions légales PMD/PMT/AAB sont toujours ajoutées automatiquement — une note portant l'un de ces codes remplace la mention par défaut du même code, une note sans subject_code est une information générale (AAI)"`
	// Objet de la facture, inscrit au registre comme intitulé du projet. À défaut,
	// le libellé de la première ligne sert de dernier recours.
	Title string `json:"title,omitempty" jsonschema:"objet de la facture en quelques mots (ex. « Refonte du site vitrine ») ; inscrit au registre comme intitulé du projet"`
}

// NoteSpec est une mention libre (BT-22) avec son code sujet (BT-21).
type NoteSpec struct {
	Content     string `json:"content" jsonschema:"texte de la note"`
	SubjectCode string `json:"subject_code,omitempty" jsonschema:"code sujet UNTDID 4451 ; vide ⇒ AAI (information générale) ; PMD, PMT ou AAB pour personnaliser la mention légale correspondante"`
}

// PartySpec décrit une partie. SIRET (14) prime pour dériver le SIREN (9) si SIREN absent.
type PartySpec struct {
	Name        string `json:"name"`
	SIREN       string `json:"siren,omitempty"`
	SIRET       string `json:"siret,omitempty"`
	VATNumber   string `json:"vat_number,omitempty"`
	Email       string `json:"email,omitempty"`
	EAddr       string `json:"electronic_address,omitempty"`        // BT-34/BT-49 adresse de routage PDP
	EAddrSchema string `json:"electronic_address_scheme,omitempty"` // ex. "0225" (SIRET FR)
	Address     string `json:"address,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
	City        string `json:"city,omitempty"`
	Country     string `json:"country_code,omitempty"`
}

// LineSpec est une ligne de prestation. Quantity est une chaîne décimale ("9.00").
type LineSpec struct {
	Name      string `json:"name"`
	Unit      string `json:"unit,omitempty"`                // "day" | "unit" (défaut)
	Quantity  string `json:"quantity,omitempty"`            // défaut "1.00"
	UnitPrice int64  `json:"unit_price_ht_cents,omitempty"` // centimes
	Amount    int64  `json:"amount_ht_cents,omitempty"`     // centimes ; 0 ⇒ = UnitPrice
}

// VATSpec porte le régime de TVA. Par défaut exonéré (franchise 293 B).
type VATSpec struct {
	Exempt        bool   `json:"exempt,omitempty"`
	Mention       string `json:"mention,omitempty"`
	ExemptionCode string `json:"exemption_code,omitempty"`
	RatePct       string `json:"rate_pct,omitempty"` // requis si Exempt == false (ex. "20.00")
}

// defaultNotes sont les mentions légales obligatoires en France (BR-FR-05) :
// pénalités de retard (PMD), frais de recouvrement (PMT), escompte (AAB).
// Reprend la formulation du template de facture.
var defaultNotes = []Note{
	{SubjectCode: "PMD", Content: "En cas de retard de paiement, application de pénalités au taux annuel de trois fois le taux d'intérêt légal en vigueur (art. L441-10 du Code de commerce), exigibles dès le jour suivant la date d'échéance et sans rappel préalable."},
	{SubjectCode: "PMT", Content: "Indemnité forfaitaire pour frais de recouvrement de 40 € (art. D441-5 du Code de commerce), applicable à chaque facture en retard, sans préjudice d'une indemnisation complémentaire sur justification."},
	{SubjectCode: "AAB", Content: "Pas d'escompte pour paiement anticipé."},
}

// legalNoteCodes sont les codes sujet (BT-21) des trois mentions obligatoires,
// dans l'ordre où elles sont émises.
var legalNoteCodes = []string{"PMD", "PMT", "AAB"}

// legalNoteLabels nomme chaque mention pour les messages d'erreur.
var legalNoteLabels = map[string]string{
	"PMD": "pénalités de retard",
	"PMT": "frais de recouvrement",
	"AAB": "escompte",
}

// generalNoteCode est le code sujet attribué à une note du spec qui n'en déclare
// pas : AAI (UNTDID 4451, « information générale »). Sans lui, un paragraphe de
// contexte partirait sans code et pourrait passer pour une mention légale.
const generalNoteCode = "AAI"

func isLegalNoteCode(code string) bool {
	for _, c := range legalNoteCodes {
		if c == code {
			return true
		}
	}
	return false
}

// mergeNotes compose les mentions du document : les trois mentions légales
// d'abord, dans l'ordre PMD, PMT, AAB — chacune remplacée par la note du spec
// qui porte le même code, s'il y en a une — puis les autres notes du spec.
//
// C'est délibérément une fusion, pas un remplacement : une note de contexte
// ajoutée à la facture ne doit jamais faire disparaître une mention légale.
// C'est exactement ce qui a fait rejeter une facture par une PDP (BR-FR-05)
// alors que le validateur EN 16931 la disait conforme.
func mergeNotes(user []NoteSpec) []Note {
	replacements := map[string]Note{}
	var extra []Note
	for _, n := range user {
		content := strings.TrimSpace(n.Content)
		if content == "" {
			continue
		}
		code := strings.ToUpper(strings.TrimSpace(n.SubjectCode))
		if isLegalNoteCode(code) {
			replacements[code] = Note{Content: content, SubjectCode: code}
			continue
		}
		if code == "" {
			code = generalNoteCode
		}
		extra = append(extra, Note{Content: content, SubjectCode: code})
	}
	out := make([]Note, 0, len(defaultNotes)+len(extra))
	for _, d := range defaultNotes {
		if r, ok := replacements[d.SubjectCode]; ok {
			out = append(out, r)
			continue
		}
		out = append(out, d)
	}
	return append(out, extra...)
}

// Schémas d'adresse électronique (EAS) rencontrés dans le routage FR. Une PDP
// française route sur l'annuaire : 0225 (SIREN, réforme FR) et 0002 (SIRENE) y
// figurent, un e-mail (EM) jamais — une facture adressée en EM est indélivrable.
const (
	eaddrSchemeEmail  = "EM"
	eaddrSchemeSIRENE = "0002"
)

// ResolveRouting détermine l'adresse électronique (BT-34 vendeur, BT-49
// acheteur) d'une partie, dans cet ordre :
//
//  1. l'adresse explicite (electronic_address), avec son schéma — ou, s'il
//     manque, EM pour un e-mail et 0225 pour un identifiant ;
//  2. sinon le SIREN, dérivé du SIRET au besoin, en 0225 ;
//  3. sinon l'e-mail en EM, uniquement quand aucun identifiant légal n'est connu.
//
// Renvoie l'adresse et son schéma, vides si rien n'est exploitable. C'est la
// même résolution pour les deux parties, et c'est elle que create_invoice
// inscrit dans le sidecar : le fichier reflète le XML.
func ResolveRouting(p PartySpec) (addr, scheme string) {
	addr = strings.ReplaceAll(strings.TrimSpace(p.EAddr), " ", "")
	scheme = strings.TrimSpace(p.EAddrSchema)
	if addr != "" {
		if scheme == "" {
			scheme = defaultEAddrScheme
			if strings.Contains(addr, "@") {
				scheme = eaddrSchemeEmail
			}
		}
		return addr, scheme
	}
	if siren := siren9(p.SIREN, p.SIRET); siren != "" {
		return siren, defaultEAddrScheme
	}
	if email := strings.TrimSpace(p.Email); email != "" {
		return email, eaddrSchemeEmail
	}
	return "", ""
}

// IsRoutableScheme dit si un schéma d'adresse est routable par une PDP
// française — c'est-à-dire présent dans l'annuaire : 0225 ou 0002.
func IsRoutableScheme(scheme string) bool {
	scheme = strings.TrimSpace(scheme)
	return scheme == defaultEAddrScheme || scheme == eaddrSchemeSIRENE
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

// MarshalSpec sérialise une spec en JSON indenté — la forme sidecar écrite à
// côté du HTML de chaque facture.
func MarshalSpec(s Spec) ([]byte, error) {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("facturx: sérialisation spec: %w", err)
	}
	return append(raw, '\n'), nil
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
	eaddr, scheme := ResolveRouting(p)
	return Party{
		Name:        p.Name,
		VATNumber:   strings.ReplaceAll(strings.TrimSpace(p.VATNumber), " ", ""),
		SIREN:       siren9(p.SIREN, p.SIRET),
		Email:       strings.TrimSpace(p.Email),
		EAddr:       eaddr,
		EAddrScheme: scheme,
		Address:     p.Address,
		PostalCode:  p.PostalCode,
		City:        p.City,
		CountryCode: country,
	}
}

// sellerParty résout le vendeur : bloc "seller" du JSON s'il est fourni, sinon
// l'identité de la configuration. Aucune identité n'est codée en dur.
func sellerParty(s *PartySpec, def PartySpec) (Party, error) {
	spec := s
	if spec == nil || strings.TrimSpace(spec.Name) == "" {
		if def.Name == "" {
			return Party{}, fmt.Errorf(
				"facturx: vendeur non configuré — renseignez %s (et les autres GOFACT_SELLER_*) "+
					"dans l'environnement ou un fichier .env, ou fournissez un bloc \"seller\" dans le JSON",
				envSellerName)
		}
		spec = &def
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

// ToInvoice projette la spec vers le modèle CII avec les défauts lus dans
// l'environnement du processus — le comportement historique du CLI.
func (s Spec) ToInvoice() (Invoice, error) {
	return s.ToInvoiceWith(ConfigFromEnv())
}

// ToInvoiceWith projette la spec (avec les défauts de cfg) vers le modèle CII,
// calcule les totaux et le sous-total de TVA. Les montants sont en centimes.
// C'est le point d'entrée pour un processus servant plusieurs organisations :
// chaque appel reçoit la configuration de l'entité émettrice concernée.
func (s Spec) ToInvoiceWith(cfg Config) (Invoice, error) {
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
			sub.ExemptionReason = cfg.VATExemptMention
		}
		sub.ExemptionReasonCode = vat.ExemptionCode
		if sub.ExemptionReasonCode == "" {
			sub.ExemptionReasonCode = cfg.VATExemptReasonCode
		}
	}

	grand := lineTotal + taxTotal

	seller, err := sellerParty(s.Seller, cfg.Seller)
	if err != nil {
		return Invoice{}, err
	}

	iban := strings.ReplaceAll(strings.TrimSpace(s.IBAN), " ", "")
	if iban == "" {
		iban = cfg.IBAN
	}

	inv := Invoice{
		Number:          strings.TrimSpace(s.Number),
		Type:            docType,
		Notes:           mergeNotes(s.Notes),
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
