package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kolapsis/gofact/internal/dotenv"
)

// Correction de l'identité d'une organisation déjà créée.
//
// Sans cette opération, une identité incomplète est une impasse : l'utilisateur
// ne s'en aperçoit qu'à la dernière étape, quand une règle EN 16931 refuse la
// facture (BR-50 sur l'IBAN, par exemple), et `Init` refuse à juste titre
// d'écraser un dossier qui porte déjà un registre de numérotation.

// IsIdentityKey dit si une clé de configuration fait partie de l'identité
// modifiable de l'organisation. Tout le reste — identifiants de plateforme,
// réglages ajoutés à la main — est hors de portée de cette opération.
func IsIdentityKey(key string) bool {
	for _, k := range identityKeys {
		if k == key {
			return true
		}
	}
	return false
}

// UpdateIdentity réécrit le .env du dossier en n'y modifiant que les clés
// fournies. Le fichier appartient à l'utilisateur : ses commentaires, l'ordre
// de ses lignes et les clés absentes de changes — identifiants de plateforme
// compris — sont conservés tels quels. Une valeur vide supprime la clé, ce qui
// est le seul moyen de revenir à la franchise de TVA après avoir renseigné un
// numéro intracommunautaire.
func (o *Org) UpdateIdentity(changes map[string]string) error {
	if len(changes) == 0 {
		return fmt.Errorf("workspace: aucune modification demandée")
	}
	for k := range changes {
		if !IsIdentityKey(k) {
			return fmt.Errorf("workspace: %s n'est pas une clé d'identité modifiable", k)
		}
	}

	path := filepath.Join(o.Path, EnvFile)
	var lines []string
	if raw, err := os.ReadFile(path); err == nil {
		if trimmed := strings.TrimRight(string(raw), "\n"); trimmed != "" {
			lines = strings.Split(trimmed, "\n")
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	seen := make(map[string]bool, len(changes))
	out := make([]string, 0, len(lines)+len(changes))
	for _, line := range lines {
		key := envKeyOf(line)
		value, ok := changes[key]
		if key == "" || !ok {
			out = append(out, line)
			continue
		}
		seen[key] = true
		if strings.TrimSpace(value) == "" {
			continue // clé vidée : la ligne disparaît
		}
		out = append(out, fmt.Sprintf("%s=%q", key, value))
	}

	// Les clés qui n'existaient pas encore, dans l'ordre canonique du fichier.
	for _, k := range identityKeys {
		value, ok := changes[k]
		if !ok || seen[k] || strings.TrimSpace(value) == "" {
			continue
		}
		if len(out) == 0 {
			out = append(out, "# Identité de l'entité émettrice — propre à ce dossier, jamais versionnée.")
		}
		out = append(out, fmt.Sprintf("%s=%q", k, value))
	}

	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	env, err := dotenv.Parse(path)
	if err != nil {
		return err
	}
	o.env = env
	return nil
}

// envKeyOf extrait le nom de variable d'une ligne de .env, ou "" si la ligne
// est un commentaire, une ligne vide ou autre chose qu'une affectation.
func envKeyOf(line string) string {
	s := strings.TrimSpace(line)
	if s == "" || strings.HasPrefix(s, "#") {
		return ""
	}
	s = strings.TrimPrefix(s, "export ")
	i := strings.IndexByte(s, '=')
	if i <= 0 {
		return ""
	}
	return strings.TrimSpace(s[:i])
}
