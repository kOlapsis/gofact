// Package pdp abstrait la plateforme de dématérialisation partenaire (PDP) sur
// laquelle les factures sont déposées. L'interface est volontairement minimale
// — déposer, suivre — et ne s'élargira qu'à l'arrivée d'un second fournisseur
// réel : généraliser sur un seul cas produit des abstractions fausses.
package pdp

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Event est un statut du cycle de vie d'une facture déposée.
type Event struct {
	CreatedAt  string `json:"created_at"`
	StatusCode string `json:"status_code"`
	StatusText string `json:"status_text"`
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
