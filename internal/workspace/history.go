package workspace

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

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

// FindClient cherche un client dans l'historique par nom (recherche par
// sous-chaîne, insensible à la casse, aux accents et aux séparateurs) ou par
// SIREN/SIRET. Un SIREN retrouve aussi les clients dont l'historique ne connaît
// que le SIRET, dont il est le préfixe.
//
// La normalisation du nom n'est pas cosmétique : sans elle, chercher « NeoDTx »
// un client enregistré « Neo DTx » ne renvoie rien, et l'appelant se rabat sur
// l'annuaire public — où un homonyme peut être facturé à sa place.
func (o *Org) FindClient(query string) ([]Client, error) {
	all, err := o.Clients()
	if err != nil {
		return nil, err
	}
	q := foldKey(query)
	digits := digitsOnly(query)
	if q == "" && digits == "" {
		return nil, nil
	}
	var out []Client
	for _, c := range all {
		if (q != "" && strings.Contains(foldKey(c.Name), q)) || matchesID(c, digits) {
			out = append(out, c)
		}
	}
	return out, nil
}

// matchesID compare une suite de chiffres au SIREN et au SIRET du client. Le
// SIREN étant les 9 premiers chiffres du SIRET, une recherche par SIREN doit
// retrouver un client dont on ne connaît que l'établissement.
func matchesID(c Client, digits string) bool {
	if digits == "" {
		return false
	}
	for _, id := range []string{digitsOnly(c.SIREN), digitsOnly(c.SIRET)} {
		if id == "" {
			continue
		}
		if id == digits || (len(digits) == 9 && len(id) >= 9 && id[:9] == digits) {
			return true
		}
	}
	return false
}

// digitsOnly ne retient que les chiffres : « 945 226 215 » et « 945226215 »
// désignent le même SIREN.
func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// foldKey réduit une chaîne à sa forme comparable : minuscules, accents
// dépliés, séparateurs et ponctuation supprimés. « Neo DTx », « NeoDTx » et
// « néo-dtx » partagent ainsi la même clé.
func foldKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if repl, ok := foldRunes[r]; ok {
			b.WriteString(repl)
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
		// Espaces, tirets, apostrophes, points : ignorés.
	}
	return b.String()
}

// foldRunes déplie les lettres accentuées et les ligatures rencontrées dans les
// raisons sociales européennes. Table explicite plutôt que golang.org/x/text :
// la dépendance resterait indirecte et le jeu à couvrir est petit et stable.
var foldRunes = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a",
	'ç': "c", 'ć': "c", 'č': "c",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ę': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i",
	'ñ': "n", 'ń': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u",
	'ý': "y", 'ÿ': "y",
	'š': "s", 'ś': "s", 'ż': "z", 'ź': "z", 'ž': "z", 'ł': "l",
	'æ': "ae", 'œ': "oe", 'ß': "ss",
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
