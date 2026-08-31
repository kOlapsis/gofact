package workspace

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kolapsis/gofact/internal/facturx"
)

// Historique client : la première source de découverte, et la plus fiable — un
// client déjà facturé a des coordonnées validées par l'usage, adresse de
// routage Peppol comprise. Zéro réseau : tout vient des sidecars JSON déjà
// présents dans le dossier.

// Client est un acheteur reconstitué depuis l'historique du dossier.
type Client struct {
	facturx.PartySpec
	LastInvoice string `json:"last_invoice"`        // numéro de la facture la plus récente
	LastIssued  string `json:"last_issued"`         // date ISO de cette facture
	Invoices    int    `json:"invoices"`            // nombre de factures retrouvées
	LastRef     string `json:"buyer_ref,omitempty"` // dernière référence acheteur (devis)
}

// Clients reconstruit la liste des acheteurs depuis les sidecars JSON du
// dossier, du plus récemment facturé au plus ancien. Seuls les fichiers dont le
// contenu est une spec gofact valide sont retenus : le dossier peut contenir
// n'importe quoi d'autre, on ne lit rien au-delà de ce format.
func (o *Org) Clients() ([]Client, error) {
	entries, err := os.ReadDir(o.Path)
	if err != nil {
		return nil, err
	}

	byKey := map[string]*Client{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == RegistryFile {
			continue
		}
		spec, err := facturx.LoadSpec(filepath.Join(o.Path, e.Name()))
		if err != nil || strings.TrimSpace(spec.Buyer.Name) == "" || spec.Number == "" {
			continue // pas un sidecar de facture
		}
		key := clientKey(spec.Buyer)
		c, ok := byKey[key]
		if !ok {
			c = &Client{}
			byKey[key] = c
		}
		c.Invoices++
		// La facture la plus récente fait foi pour les coordonnées.
		if spec.IssueDate >= c.LastIssued {
			c.PartySpec = spec.Buyer
			c.LastIssued = spec.IssueDate
			c.LastInvoice = spec.Number
			c.LastRef = spec.BuyerRef
		}
	}

	out := make([]Client, 0, len(byKey))
	for _, c := range byKey {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastIssued > out[j].LastIssued })
	return out, nil
}

// FindClient cherche un client dans l'historique par nom (insensible à la casse
// et aux accents près — recherche par sous-chaîne) ou par SIREN/SIRET exact.
func (o *Org) FindClient(query string) ([]Client, error) {
	all, err := o.Clients()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	digits := strings.ReplaceAll(q, " ", "")
	var out []Client
	for _, c := range all {
		if strings.Contains(strings.ToLower(c.Name), q) ||
			(digits != "" && (c.SIREN == digits || c.SIRET == digits)) {
			out = append(out, c)
		}
	}
	return out, nil
}

// clientKey identifie un client : le SIREN quand il existe (stable à travers les
// renommages), sinon le nom normalisé.
func clientKey(p facturx.PartySpec) string {
	if s := strings.ReplaceAll(strings.TrimSpace(p.SIREN), " ", ""); s != "" {
		return "siren:" + s
	}
	if s := strings.ReplaceAll(strings.TrimSpace(p.SIRET), " ", ""); len(s) >= 9 {
		return "siren:" + s[:9]
	}
	return "name:" + strings.ToLower(strings.TrimSpace(p.Name))
}
