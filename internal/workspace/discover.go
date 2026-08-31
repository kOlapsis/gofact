package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Découverte des dossiers d'organisation, par ordre de priorité :
//
//	1. chemin explicite (paramètre d'outil MCP, flag CLI) ;
//	2. registre utilisateur ~/.config/gofact/orgs.json ;
//	3. GOFACT_ORGS_DIRS (liste de chemins, séparateur du système) ;
//	4. GOFACT_INVOICES_DIR (une seule organisation — compatibilité avec le
//	   skill historique) ;
//	5. le répertoire courant, s'il est lui-même un dossier d'organisation.
//
// On n'explore jamais le disque au-delà de ces sources : un serveur qui
// fouillerait ~/ pour « trouver des factures » serait une faute.

// OrgsRegistryFile liste les dossiers d'organisation connus de l'utilisateur.
type orgsRegistry struct {
	Orgs []string `json:"orgs"`
}

func orgsRegistryPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "gofact", "orgs.json")
}

// Discover ouvre les organisations accessibles. explicit, s'il est non vide,
// court-circuite tout le reste.
func Discover(explicit string) ([]*Org, error) {
	if explicit != "" {
		org, err := Open(explicit)
		if err != nil {
			return nil, err
		}
		return []*Org{org}, nil
	}

	var paths []string
	seen := map[string]bool{}
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}

	if reg := orgsRegistryPath(); reg != "" {
		if raw, err := os.ReadFile(reg); err == nil {
			var r orgsRegistry
			if err := json.Unmarshal(raw, &r); err != nil {
				return nil, fmt.Errorf("workspace: %s illisible: %w", reg, err)
			}
			for _, p := range r.Orgs {
				add(p)
			}
		}
	}
	for _, p := range filepath.SplitList(os.Getenv("GOFACT_ORGS_DIRS")) {
		add(p)
	}
	add(os.Getenv("GOFACT_INVOICES_DIR"))
	if cwd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(cwd, RegistryFile)); err == nil {
			add(cwd)
		}
	}

	var orgs []*Org
	for _, p := range paths {
		org, err := Open(p)
		if err != nil {
			continue // un chemin listé mais invalide n'empêche pas les autres
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}

// Register inscrit un dossier dans le registre utilisateur, pour qu'il soit
// découvert dans les sessions suivantes.
func Register(path string) error {
	reg := orgsRegistryPath()
	if reg == "" {
		return fmt.Errorf("workspace: répertoire de configuration introuvable")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	var r orgsRegistry
	if raw, err := os.ReadFile(reg); err == nil {
		_ = json.Unmarshal(raw, &r)
	}
	for _, p := range r.Orgs {
		if p == abs {
			return nil
		}
	}
	r.Orgs = append(r.Orgs, abs)
	if err := os.MkdirAll(filepath.Dir(reg), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := reg + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, reg)
}

// Init crée un nouveau dossier d'organisation : registre vierge et .env
// d'identité. Refuse d'écraser un dossier déjà initialisé.
func Init(path string, identity map[string]string) (*Org, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(abs, RegistryFile)); err == nil {
		return nil, fmt.Errorf("workspace: %s est déjà un dossier d'organisation", abs)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}

	seed := map[string]any{
		"_doc": "Registre de numérotation gofact. Source de vérité : numérotation continue, " +
			"sans trou, jamais réutilisée (obligation légale). Ne pas éditer les compteurs à la main.",
		"compteurs": map[string]int{},
		"factures":  []any{},
	}
	raw, err := json.MarshalIndent(seed, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(abs, RegistryFile), raw, 0o644); err != nil {
		return nil, err
	}

	// .env : uniquement les clés fournies — on n'invente rien, et jamais de
	// valeur d'exemple qui pourrait finir sur une vraie facture.
	if len(identity) > 0 {
		var b strings.Builder
		b.WriteString("# Identité de l'entité émettrice — propre à ce dossier, jamais versionnée.\n")
		for _, k := range identityKeys {
			if v, ok := identity[k]; ok && strings.TrimSpace(v) != "" {
				fmt.Fprintf(&b, "%s=%q\n", k, v)
			}
		}
		if err := os.WriteFile(filepath.Join(abs, EnvFile), []byte(b.String()), 0o600); err != nil {
			return nil, err
		}
	}

	if err := Register(abs); err != nil {
		return nil, err
	}
	return Open(abs)
}

// identityKeys est l'ordre d'écriture des clés d'identité dans le .env.
var identityKeys = []string{
	"GOFACT_SELLER_NAME",
	"GOFACT_SELLER_SIRET",
	"GOFACT_SELLER_SIREN",
	"GOFACT_SELLER_VAT_NUMBER",
	"GOFACT_SELLER_EMAIL",
	"GOFACT_SELLER_ADDRESS",
	"GOFACT_SELLER_POSTAL_CODE",
	"GOFACT_SELLER_CITY",
	"GOFACT_SELLER_COUNTRY",
	"GOFACT_PAYEE_IBAN",
}
