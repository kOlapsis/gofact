// Package pdp abstrait la plateforme de dématérialisation partenaire (PDP) sur
// laquelle les factures sont déposées et reçues. L'interface ne couvre que ce
// dont gofact se sert : déposer, suivre, signaler un encaissement, recevoir.
package pdp

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Event est un statut du cycle de vie d'une facture déposée.
type Event struct {
	CreatedAt  string   `json:"created_at"`
	StatusCode string   `json:"status_code"`
	StatusText string   `json:"status_text"`
	Reasons    []string `json:"reasons,omitempty"` // motifs détaillés d'un rejet, un par ligne
}

// IsRejection dit si un statut marque le rejet de la facture — par la
// plateforme (fr:213 « Rejetée »), par le concentrateur (ppf:rejected) ou par
// le destinataire (fr:210 « Refusée »). Un code ou un libellé suffit : les
// fournisseurs ne partagent que le vocabulaire du cycle de vie FR.
func IsRejection(e Event) bool {
	code := strings.ToLower(strings.TrimSpace(e.StatusCode))
	switch {
	case code == "fr:213", code == "fr:210",
		strings.HasSuffix(code, ":rejected"), strings.HasSuffix(code, ":refused"):
		return true
	}
	text := strings.ToLower(e.StatusText)
	return strings.Contains(text, "rejet") || strings.Contains(text, "reject") || strings.Contains(text, "refus")
}

// Rejection résume un cycle de vie : la facture a-t-elle été rejetée, et pour
// quels motifs — ceux portés par les événements de rejet, dédoublonnés, dans
// l'ordre d'apparition. Un rejet sans motif détaillé renvoie le libellé du statut.
func Rejection(events []Event) (rejected bool, reasons []string) {
	seen := map[string]bool{}
	for _, e := range events {
		if !IsRejection(e) {
			continue
		}
		rejected = true
		lines := e.Reasons
		if len(lines) == 0 {
			lines = []string{strings.TrimSpace(e.StatusCode + " " + e.StatusText)}
		}
		for _, r := range lines {
			r = strings.TrimSpace(r)
			if r == "" || seen[r] {
				continue
			}
			seen[r] = true
			reasons = append(reasons, r)
		}
	}
	return rejected, reasons
}

// Receipt est l'accusé d'un dépôt.
type Receipt struct {
	Provider  string  `json:"provider"`
	Reference string  `json:"reference"` // identifiant chez le fournisseur
	Events    []Event `json:"events"`
}

// Provider est un fournisseur PDP.
type Provider interface {
	// Name est l'identifiant stable du fournisseur (ex. "superpdp").
	Name() string
	// Send dépose un PDF Factur-X et renvoie l'accusé.
	Send(ctx context.Context, pdfPath string) (Receipt, error)
	// Status renvoie le cycle de vie d'un dépôt antérieur.
	Status(ctx context.Context, reference string) ([]Event, error)
	// ReportPayment signale l'encaissement d'une facture émise.
	ReportPayment(ctx context.Context, reference string, p Payment) error
	// Received liste les factures reçues après le curseur afterID, dans l'ordre d'arrivée.
	Received(ctx context.Context, afterID int64) ([]Incoming, error)
	// Download renvoie le PDF Factur-X lisible d'une facture, émise ou reçue.
	Download(ctx context.Context, reference string) ([]byte, error)
}

// Payment décrit un encaissement ; sans ventilation, la plateforme retient le total de la facture à la date du signalement.
type Payment struct {
	Date     string // ISO YYYY-MM-DD
	Currency string
	Parts    []PaymentPart
}

// PaymentPart est la part d'un encaissement soumise à un taux de TVA.
type PaymentPart struct {
	Amount  string // décimal, ex. "1200.00"
	VATRate string // pourcentage, ex. "20.00"
}

// Incoming est une facture reçue par la plateforme.
type Incoming struct {
	Provider   string `json:"provider"`
	Reference  string `json:"reference"`
	Cursor     int64  `json:"cursor"`
	ReceivedAt string `json:"received_at"`
	Number     string `json:"number"`
	IssueDate  string `json:"issue_date"`
	Seller     string `json:"seller"`
	SellerID   string `json:"seller_id,omitempty"`
	TotalHT    string `json:"total_ht"`
	TotalVAT   string `json:"total_vat"`
	TotalTTC   string `json:"total_ttc"`
	Currency   string `json:"currency"`
}

// Factory construit un fournisseur depuis une source de configuration (le .env
// d'une organisation, ou l'environnement du processus). lookup renvoie "" pour
// une clé absente. Elle renvoie une erreur EXPLICABLE si la configuration est
// incomplète — le message sera relayé tel quel à l'utilisateur.
type Factory func(lookup func(string) string) (Provider, error)

var factories = map[string]Factory{}

// RegisterProvider inscrit un fournisseur au registre. À appeler depuis un init().
func RegisterProvider(name string, f Factory) { factories[name] = f }

// EnvProvider est la variable qui sélectionne le fournisseur d'une organisation.
const EnvProvider = "GOFACT_PDP"

// Open construit le fournisseur sélectionné par la configuration. Un seul
// fournisseur existant, il est le défaut ; la variable GOFACT_PDP tranche
// lorsqu'il y en aura plusieurs.
func Open(lookup func(string) string) (Provider, error) {
	name := strings.TrimSpace(lookup(EnvProvider))
	if name == "" {
		name = "superpdp"
	}
	f, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("pdp: fournisseur %q inconnu (disponibles : %s)", name, strings.Join(Names(), ", "))
	}
	return f(lookup)
}

// Names liste les fournisseurs enregistrés.
func Names() []string {
	out := make([]string, 0, len(factories))
	for n := range factories {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
