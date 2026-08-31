// Package workspace gère les dossiers d'organisation : une entité émettrice =
// un dossier autonome, qui porte son identité (.env), son registre de
// numérotation (numerotation.json), son modèle de facture figé et ses factures.
//
// Le paquet est conçu pour un processus long servant plusieurs organisations
// (le serveur MCP) : rien ne passe par l'environnement global du processus, et
// l'attribution des numéros — une obligation légale de séquence continue — est
// verrouillée contre les accès concurrents.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kolapsis/gofact/internal/dotenv"
	"github.com/kolapsis/gofact/internal/facturx"
)

// RegistryFile est le registre de numérotation, source de vérité du dossier.
const RegistryFile = "numerotation.json"

// EnvFile porte l'identité de l'entité émettrice (GOFACT_SELLER_*, IBAN…) et,
// le cas échéant, ses identifiants PDP. Il ne quitte jamais le dossier.
const EnvFile = ".env"

// secretPrefixes désigne les clés dont la valeur ne doit JAMAIS sortir du
// dossier — ni dans une sortie d'outil MCP, ni dans un affichage CLI.
var secretKeywords = []string{"SECRET", "TOKEN", "PASSWORD", "_KEY"}

// Org est un dossier d'organisation ouvert.
type Org struct {
	Path string
	env  map[string]string // contenu du .env du dossier, secrets compris
}

// Open ouvre un dossier d'organisation existant. Le dossier doit contenir un
// registre de numérotation — c'est ce qui le distingue d'un dossier quelconque,
// et ce qui évite de lire des fichiers au hasard sur le disque.
func Open(path string) (*Org, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(abs, RegistryFile)); err != nil {
		return nil, fmt.Errorf("workspace: %s n'est pas un dossier d'organisation (pas de %s)", abs, RegistryFile)
	}
	env, err := dotenv.Parse(filepath.Join(abs, EnvFile))
	if err != nil {
		return nil, err
	}
	return &Org{Path: abs, env: env}, nil
}

// Name est le nom de l'entité émettrice : celui du vendeur configuré, sinon le
// nom du dossier.
func (o *Org) Name() string {
	if n := strings.TrimSpace(o.env["GOFACT_SELLER_NAME"]); n != "" {
		return n
	}
	return filepath.Base(o.Path)
}

// Config résout les défauts de facturation de l'organisation : d'abord le .env
// du dossier, puis l'environnement du processus en repli. Aucune écriture dans
// l'environnement global — deux organisations ne se contaminent jamais.
func (o *Org) Config() facturx.Config {
	return facturx.ConfigFrom(o.Lookup)
}

// Lookup renvoie une variable de l'organisation, avec l'environnement du
// processus en repli. C'est la source de configuration unique du dossier.
func (o *Org) Lookup(key string) string {
	if v, ok := o.env[key]; ok {
		return v
	}
	return os.Getenv(key)
}

// Identity décrit l'organisation SANS exposer de secret : c'est la forme
// destinée aux sorties d'outils (MCP, CLI).
type Identity struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	SIRET      string `json:"siret,omitempty"`
	SIREN      string `json:"siren,omitempty"`
	Email      string `json:"email,omitempty"`
	Address    string `json:"address,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	City       string `json:"city,omitempty"`
	Country    string `json:"country,omitempty"`
	HasIBAN    bool   `json:"has_iban"`
	HasPDP     bool   `json:"has_pdp"`
}

// Identity projette la configuration en une fiche sans secret. L'IBAN n'est pas
// un secret au sens strict (il figure sur chaque facture) mais il n'a rien à
// faire dans une fiche descriptive : on n'en publie que la présence.
func (o *Org) Identity() Identity {
	cfg := o.Config()
	s := cfg.Seller
	return Identity{
		Name:       o.Name(),
		Path:       o.Path,
		SIRET:      s.SIRET,
		SIREN:      s.SIREN,
		Email:      s.Email,
		Address:    s.Address,
		PostalCode: s.PostalCode,
		City:       s.City,
		Country:    s.Country,
		HasIBAN:    cfg.IBAN != "",
		HasPDP:     o.Lookup("SUPERPDP_CLIENT_ID") != "" && o.Lookup("SUPERPDP_CLIENT_SECRET") != "",
	}
}

// IsSecretKey dit si une clé de configuration désigne un secret. Toute couche
// d'affichage doit s'y référer avant d'imprimer une valeur.
func IsSecretKey(key string) bool {
	up := strings.ToUpper(key)
	for _, kw := range secretKeywords {
		if strings.Contains(up, kw) {
			return true
		}
	}
	return false
}
