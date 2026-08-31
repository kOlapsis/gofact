package facturx

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"time"
)

// Espaces de noms CII (UN/CEFACT D16B).
const (
	nsRSM = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	nsRAM = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	nsUDT = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"
	nsQDT = "urn:un:unece:uncefact:data:standard:QualifiedDataType:100"

	// guidelineEN16931 identifie le profil de conformité Factur-X (BT-24). DOIT rester la
	// valeur EN 16931 « nue » : le schématron Factur-X (FX-SCH-A-000026, liste cl id=1) et
	// le socle EN 16931 n'acceptent que cette valeur. Le profil de routage réforme FR
	// (« …france:billing:Factur-X:1.0 ») relève de la couche TRANSPORT Peppol, assignée par
	// la PDP — l'injecter ici casse Factur-X/EN16931. La réforme FR s'exprime via BT-23
	// (mode de facturation) + le contenu, par-dessus un socle EN 16931 valide.
	guidelineEN16931 = "urn:cen.eu:en16931:2017"

	// processBillingFR est le « mode de facturation » (BT-23) par défaut. La réforme FR
	// l'exige (PEPPOL-EN16931-R001) et le contraint (BR-FR-08, schématron Flux 2) à la
	// liste : B1, S1, M1, B2, S2, M2, B4, S4, M4, S5, S6, B7, S7 — où le préfixe code la
	// nature (B = Biens, S = Services, M = Mixte) et le chiffre le cas. Le défaut vise le
	// cas le plus courant : « S1 » (services, facture domestique standard).
	// Surchargeable par facture via le sidecar (`business_process`).
	processBillingFR = "S1"
)

// DocTypeCode est le code type de document (UNTDID 1001).
type DocTypeCode string

const (
	DocInvoice    DocTypeCode = "380" // facture
	DocCreditNote DocTypeCode = "381" // avoir
)

// Party est une partie de la transaction (vendeur ou acheteur).
type Party struct {
	Name        string
	VATNumber   string // BT-31 n° TVA intracom (ex. FR12345678901) ; vide si non assujetti (293 B)
	TaxID       string // BT-32 identifiant fiscal (schemeID "FC") ; requis si exonéré sans n° TVA
	SIREN       string // identifiant légal FR (BT-30/BT-47)
	Email       string // adresse électronique e-mail (BT-34/BT-49, scheme EM) ; repli si EAddr vide
	EAddr       string // BT-34/BT-49 adresse électronique de routage (PDP) ; prioritaire sur Email
	EAddrScheme string // schéma de l'adresse de routage (ex. "0225" SIRET FR) ; défaut "EM"
	Address     string
	PostalCode  string
	City        string
	CountryCode string // ISO 3166-1 alpha-2 (BT-40 / BT-55)
}

// Line est une ligne de facture (montants en centimes int64, jamais de float).
type Line struct {
	ID          string // BT-126 (numéro de ligne)
	Name        string // BT-153 (désignation)
	UnitCode    string // UN/ECE Rec 20 : "DAY" (TJM), "C62" (unité/forfait)
	Quantity    string // quantité formatée (ex. "9.00")
	NetPrice    int64  // BT-146 prix unitaire net, centimes
	LineTotal   int64  // BT-131 montant net de ligne, centimes
	VATCategory string // BT-151 : "S" standard, "E" exonéré
	VATRatePct  string // BT-152 : taux en % (ex. "20.00", "0.00")
}

// VATSubtotal est un sous-total de TVA par taux/catégorie.
type VATSubtotal struct {
	CategoryCode        string // "S", "E"…
	RatePct             string // "20.00", "0.00"
	BasisAmount         int64  // BT-116, centimes
	CalculatedTax       int64  // BT-117, centimes
	ExemptionReason     string // BT-120, requis si exonéré (ex. franchise 293 B)
	ExemptionReasonCode string // BT-121, code VATEX (ex. VATEX-FR-FRANCHISE)
}

// Note est une mention libre du document (BG-1) : contenu (BT-22) + code sujet
// (BT-21, UNTDID 4451). Les mentions légales françaises utilisent PMD (pénalités
// de retard), PMT (frais de recouvrement) et AAB (escompte).
type Note struct {
	Content     string
	SubjectCode string
}

// Invoice est le modèle métier d'une facture/avoir prêt à projeter en CII.
type Invoice struct {
	Number          string
	Type            DocTypeCode
	BusinessProcess string // BT-23 ; omis si vide (doit sinon suivre la liste FR, BR-FR-08)
	Notes           []Note // BG-1 mentions (légales FR…)
	IssueDate       time.Time
	DueDate         time.Time // BT-9 échéance (requise si DuePayable > 0)
	DeliveryDate    time.Time // BT-72 date de livraison/prestation (optionnel)
	BuyerReference  string    // BT-10 référence acheteur ; vide si absent
	Currency        string    // ISO 4217 (BT-5)
	PaymentMeans    string    // BT-81 code UNTDID 4461 (ex. "30" virement) ; vide si absent
	PayeeIBAN       string    // BT-84 IBAN du compte de règlement ; vide si absent
	Seller          Party
	Buyer           Party
	Lines           []Line
	VATBreakdown    []VATSubtotal
	LineTotal       int64 // BT-106, somme des montants nets de ligne
	TaxBasisTotal   int64 // BT-109, base imposable totale
	TaxTotal        int64 // BT-110, TVA dans la devise du document
	VATEur          int64 // BT-111, TVA en devise de comptabilisation (EUR) si devise ≠ EUR
	GrandTotal      int64 // BT-112
	DuePayable      int64 // BT-115
}

// BuildCII sérialise une facture/avoir en XML CII profil EN 16931.
func BuildCII(inv Invoice) ([]byte, error) {
	// BT-23 (processus métier) : requis par le profil réforme FR (PEPPOL-EN16931-R001).
	// Défaut = processus Peppol Billing ; surchargeable via le modèle si besoin.
	bp := inv.BusinessProcess
	if bp == "" {
		bp = processBillingFR
	}
	ctx := ciiContext{
		BusinessProcess: &ciiID{Value: bp},
		Guideline:       ciiID{Value: guidelineEN16931},
	}

	doc := ciiInvoice{
		XMLNSRsm:    nsRSM,
		XMLNSRam:    nsRAM,
		XMLNSUdt:    nsUDT,
		XMLNSQdt:    nsQDT,
		Context:     ctx,
		Document:    buildDocument(inv),
		Transaction: buildTransaction(inv),
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("facturx: encodage CII: %w", err)
	}
	return buf.Bytes(), nil
}

func buildDocument(inv Invoice) ciiDocument {
	notes := make([]ciiNote, 0, len(inv.Notes))
	for _, n := range inv.Notes {
		notes = append(notes, ciiNote{Content: n.Content, SubjectCode: n.SubjectCode})
	}
	return ciiDocument{
		ID:       inv.Number,
		TypeCode: string(inv.Type),
		IssueDateTime: ciiIssueDate{
			DateTimeString: ciiDateString{Format: "102", Value: inv.IssueDate.Format("20060102")},
		},
		Notes: notes,
	}
}

func buildTransaction(inv Invoice) ciiTransaction {
	lines := make([]ciiLineItem, 0, len(inv.Lines))
	for _, l := range inv.Lines {
		lines = append(lines, ciiLineItem{
			DocLine: ciiDocLine{LineID: l.ID},
			Product: ciiProduct{Name: l.Name},
			Agreement: ciiLineAgreement{
				NetPrice: ciiNetPrice{ChargeAmount: formatAmount(l.NetPrice)},
			},
			Delivery: ciiLineDelivery{
				BilledQuantity: ciiQuantity{UnitCode: l.UnitCode, Value: l.Quantity},
			},
			Settlement: ciiLineSettlement{
				Tax: ciiLineTax{TypeCode: "VAT", CategoryCode: l.VATCategory, RatePercent: l.VATRatePct},
				Summation: ciiLineSummation{
					LineTotalAmount: formatAmount(l.LineTotal),
				},
			},
		})
	}

	taxes := make([]ciiHeaderTax, 0, len(inv.VATBreakdown))
	for _, v := range inv.VATBreakdown {
		t := ciiHeaderTax{
			CalculatedAmount:    formatAmount(v.CalculatedTax),
			TypeCode:            "VAT",
			ExemptionReason:     v.ExemptionReason,
			BasisAmount:         formatAmount(v.BasisAmount),
			CategoryCode:        v.CategoryCode,
			ExemptionReasonCode: v.ExemptionReasonCode,
			RatePercent:         v.RatePct,
		}
		taxes = append(taxes, t)
	}

	var delivery ciiHeaderDelivery
	if !inv.DeliveryDate.IsZero() {
		delivery.Event = &ciiDeliveryEvent{
			OccurrenceDateTime: ciiOccurrence{
				DateTimeString: ciiDateString{Format: "102", Value: inv.DeliveryDate.Format("20060102")},
			},
		}
	}

	settlement := ciiHeaderSettlement{
		Currency: inv.Currency,
		Taxes:    taxes,
		Summation: ciiHeaderSummation{
			LineTotalAmount:     formatAmount(inv.LineTotal),
			TaxBasisTotalAmount: formatAmount(inv.TaxBasisTotal),
			TaxTotalAmount:      []ciiAmountCur{{CurrencyID: inv.Currency, Value: formatAmount(inv.TaxTotal)}},
			GrandTotalAmount:    formatAmount(inv.GrandTotal),
			DuePayableAmount:    formatAmount(inv.DuePayable),
		},
	}
	// Document en devise étrangère : déclarer la devise de comptabilisation (BT-6=EUR)
	// et porter la contre-valeur TVA en EUR (BT-111). EN 16931 BR-53 l'exige dès que
	// la devise de TVA diffère de la devise de facture ; sans elle, BR-53 échoue.
	if inv.Currency != "EUR" {
		settlement.TaxCurrency = "EUR"
		settlement.Summation.TaxTotalAmount = append(settlement.Summation.TaxTotalAmount,
			ciiAmountCur{CurrencyID: "EUR", Value: formatAmount(inv.VATEur)})
	}
	if inv.PaymentMeans != "" {
		pm := &ciiPaymentMeans{TypeCode: inv.PaymentMeans}
		if inv.PayeeIBAN != "" {
			pm.PayeeAccount = &ciiPayeeAccount{IBANID: inv.PayeeIBAN}
		}
		settlement.PaymentMeans = pm
	}
	if !inv.DueDate.IsZero() {
		settlement.PaymentTerms = &ciiPaymentTerms{
			DueDate: ciiDueDate{
				DateTimeString: ciiDateString{Format: "102", Value: inv.DueDate.Format("20060102")},
			},
		}
	}

	return ciiTransaction{
		Lines: lines,
		Agreement: ciiHeaderAgreement{
			BuyerReference: inv.BuyerReference,
			Seller:         buildParty(inv.Seller),
			Buyer:          buildParty(inv.Buyer),
		},
		Delivery:   delivery,
		Settlement: settlement,
	}
}

func buildParty(p Party) ciiParty {
	party := ciiParty{
		Name: p.Name,
		Address: ciiAddress{
			PostcodeCode: p.PostalCode,
			LineOne:      p.Address,
			CityName:     p.City,
			CountryID:    p.CountryCode,
		},
	}
	if p.SIREN != "" {
		party.LegalOrg = &ciiLegalOrg{ID: ciiSchemeID{SchemeID: "0002", Value: p.SIREN}}
	}
	// BT-34/BT-49 adresse électronique : adresse de routage (PDP) prioritaire,
	// sinon l'e-mail (scheme EM).
	if p.EAddr != "" {
		scheme := p.EAddrScheme
		if scheme == "" {
			scheme = "EM"
		}
		party.URI = &ciiURIComm{ID: ciiSchemeID{SchemeID: scheme, Value: p.EAddr}}
	} else if p.Email != "" {
		party.URI = &ciiURIComm{ID: ciiSchemeID{SchemeID: "EM", Value: p.Email}}
	}
	// BT-31 (n° TVA, scheme VA) et/ou BT-32 (identifiant fiscal, scheme FC).
	// En franchise 293 B le vendeur n'a pas de n° TVA : BT-32 satisfait BR-E-02.
	if p.VATNumber != "" {
		party.TaxRegistration = append(party.TaxRegistration,
			ciiTaxRegistration{ID: ciiSchemeID{SchemeID: "VA", Value: p.VATNumber}})
	}
	if p.TaxID != "" {
		party.TaxRegistration = append(party.TaxRegistration,
			ciiTaxRegistration{ID: ciiSchemeID{SchemeID: "FC", Value: p.TaxID}})
	}
	return party
}

// formatAmount convertit des centimes int64 en décimal à 2 chiffres, sans float.
func formatAmount(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	s := fmt.Sprintf("%d.%02d", cents/100, cents%100)
	if neg {
		return "-" + s
	}
	return s
}

// ── Modèle XML (ordre des champs = ordre des séquences XSD CII) ────────────

type ciiInvoice struct {
	XMLName     xml.Name       `xml:"rsm:CrossIndustryInvoice"`
	XMLNSRsm    string         `xml:"xmlns:rsm,attr"`
	XMLNSRam    string         `xml:"xmlns:ram,attr"`
	XMLNSUdt    string         `xml:"xmlns:udt,attr"`
	XMLNSQdt    string         `xml:"xmlns:qdt,attr"`
	Context     ciiContext     `xml:"rsm:ExchangedDocumentContext"`
	Document    ciiDocument    `xml:"rsm:ExchangedDocument"`
	Transaction ciiTransaction `xml:"rsm:SupplyChainTradeTransaction"`
}

type ciiContext struct {
	BusinessProcess *ciiID `xml:"ram:BusinessProcessSpecifiedDocumentContextParameter>ram:ID,omitempty"`
	Guideline       ciiID  `xml:"ram:GuidelineSpecifiedDocumentContextParameter>ram:ID"`
}

type ciiID struct {
	Value string `xml:",chardata"`
}

type ciiDocument struct {
	ID            string       `xml:"ram:ID"`
	TypeCode      string       `xml:"ram:TypeCode"`
	IssueDateTime ciiIssueDate `xml:"ram:IssueDateTime"`
	Notes         []ciiNote    `xml:"ram:IncludedNote,omitempty"`
}

type ciiNote struct {
	Content     string `xml:"ram:Content"`
	SubjectCode string `xml:"ram:SubjectCode,omitempty"`
}

type ciiIssueDate struct {
	DateTimeString ciiDateString `xml:"udt:DateTimeString"`
}

type ciiDateString struct {
	Format string `xml:"format,attr"`
	Value  string `xml:",chardata"`
}

type ciiTransaction struct {
	Lines      []ciiLineItem       `xml:"ram:IncludedSupplyChainTradeLineItem"`
	Agreement  ciiHeaderAgreement  `xml:"ram:ApplicableHeaderTradeAgreement"`
	Delivery   ciiHeaderDelivery   `xml:"ram:ApplicableHeaderTradeDelivery"`
	Settlement ciiHeaderSettlement `xml:"ram:ApplicableHeaderTradeSettlement"`
}

type ciiLineItem struct {
	DocLine    ciiDocLine        `xml:"ram:AssociatedDocumentLineDocument"`
	Product    ciiProduct        `xml:"ram:SpecifiedTradeProduct"`
	Agreement  ciiLineAgreement  `xml:"ram:SpecifiedLineTradeAgreement"`
	Delivery   ciiLineDelivery   `xml:"ram:SpecifiedLineTradeDelivery"`
	Settlement ciiLineSettlement `xml:"ram:SpecifiedLineTradeSettlement"`
}

type ciiDocLine struct {
	LineID string `xml:"ram:LineID"`
}

type ciiProduct struct {
	Name string `xml:"ram:Name"`
}

type ciiLineAgreement struct {
	NetPrice ciiNetPrice `xml:"ram:NetPriceProductTradePrice"`
}

type ciiNetPrice struct {
	ChargeAmount string `xml:"ram:ChargeAmount"`
}

type ciiLineDelivery struct {
	BilledQuantity ciiQuantity `xml:"ram:BilledQuantity"`
}

type ciiQuantity struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

type ciiLineSettlement struct {
	Tax       ciiLineTax       `xml:"ram:ApplicableTradeTax"`
	Summation ciiLineSummation `xml:"ram:SpecifiedTradeSettlementLineMonetarySummation"`
}

type ciiLineTax struct {
	TypeCode     string `xml:"ram:TypeCode"`
	CategoryCode string `xml:"ram:CategoryCode"`
	RatePercent  string `xml:"ram:RateApplicablePercent"`
}

type ciiLineSummation struct {
	LineTotalAmount string `xml:"ram:LineTotalAmount"`
}

type ciiHeaderAgreement struct {
	BuyerReference string   `xml:"ram:BuyerReference,omitempty"`
	Seller         ciiParty `xml:"ram:SellerTradeParty"`
	Buyer          ciiParty `xml:"ram:BuyerTradeParty"`
}

type ciiParty struct {
	Name            string               `xml:"ram:Name"`
	LegalOrg        *ciiLegalOrg         `xml:"ram:SpecifiedLegalOrganization,omitempty"`
	Address         ciiAddress           `xml:"ram:PostalTradeAddress"`
	URI             *ciiURIComm          `xml:"ram:URIUniversalCommunication,omitempty"`
	TaxRegistration []ciiTaxRegistration `xml:"ram:SpecifiedTaxRegistration,omitempty"`
}

type ciiURIComm struct {
	ID ciiSchemeID `xml:"ram:URIID"`
}

type ciiLegalOrg struct {
	ID ciiSchemeID `xml:"ram:ID"`
}

type ciiAddress struct {
	PostcodeCode string `xml:"ram:PostcodeCode,omitempty"`
	LineOne      string `xml:"ram:LineOne,omitempty"`
	CityName     string `xml:"ram:CityName,omitempty"`
	CountryID    string `xml:"ram:CountryID"`
}

type ciiTaxRegistration struct {
	ID ciiSchemeID `xml:"ram:ID"`
}

type ciiSchemeID struct {
	SchemeID string `xml:"schemeID,attr"`
	Value    string `xml:",chardata"`
}

type ciiHeaderDelivery struct {
	Event *ciiDeliveryEvent `xml:"ram:ActualDeliverySupplyChainEvent,omitempty"`
}

type ciiDeliveryEvent struct {
	OccurrenceDateTime ciiOccurrence `xml:"ram:OccurrenceDateTime"`
}

type ciiOccurrence struct {
	DateTimeString ciiDateString `xml:"udt:DateTimeString"`
}

type ciiHeaderSettlement struct {
	// TaxCurrency (BT-6) précède InvoiceCurrencyCode (BT-5) dans la séquence XSD ;
	// présent uniquement quand la devise de comptabilisation diffère (devise ≠ EUR).
	TaxCurrency  string             `xml:"ram:TaxCurrencyCode,omitempty"`
	Currency     string             `xml:"ram:InvoiceCurrencyCode"`
	PaymentMeans *ciiPaymentMeans   `xml:"ram:SpecifiedTradeSettlementPaymentMeans,omitempty"`
	Taxes        []ciiHeaderTax     `xml:"ram:ApplicableTradeTax"`
	PaymentTerms *ciiPaymentTerms   `xml:"ram:SpecifiedTradePaymentTerms,omitempty"`
	Summation    ciiHeaderSummation `xml:"ram:SpecifiedTradeSettlementHeaderMonetarySummation"`
}

type ciiPaymentMeans struct {
	TypeCode     string           `xml:"ram:TypeCode"`
	PayeeAccount *ciiPayeeAccount `xml:"ram:PayeePartyCreditorFinancialAccount,omitempty"`
}

type ciiPayeeAccount struct {
	IBANID string `xml:"ram:IBANID"`
}

type ciiPaymentTerms struct {
	DueDate ciiDueDate `xml:"ram:DueDateDateTime"`
}

type ciiDueDate struct {
	DateTimeString ciiDateString `xml:"udt:DateTimeString"`
}

type ciiHeaderTax struct {
	CalculatedAmount    string `xml:"ram:CalculatedAmount"`
	TypeCode            string `xml:"ram:TypeCode"`
	ExemptionReason     string `xml:"ram:ExemptionReason,omitempty"`
	BasisAmount         string `xml:"ram:BasisAmount"`
	CategoryCode        string `xml:"ram:CategoryCode"`
	ExemptionReasonCode string `xml:"ram:ExemptionReasonCode,omitempty"`
	RatePercent         string `xml:"ram:RateApplicablePercent"`
}

type ciiHeaderSummation struct {
	LineTotalAmount     string `xml:"ram:LineTotalAmount"`
	TaxBasisTotalAmount string `xml:"ram:TaxBasisTotalAmount"`
	// 0..2 TaxTotalAmount : BT-110 (devise du document) puis, si devise ≠ EUR,
	// BT-111 (contre-valeur TVA en devise de comptabilisation EUR — EN 16931 BR-53).
	TaxTotalAmount   []ciiAmountCur `xml:"ram:TaxTotalAmount"`
	GrandTotalAmount string         `xml:"ram:GrandTotalAmount"`
	DuePayableAmount string         `xml:"ram:DuePayableAmount"`
}

type ciiAmountCur struct {
	CurrencyID string `xml:"currencyID,attr"`
	Value      string `xml:",chardata"`
}
