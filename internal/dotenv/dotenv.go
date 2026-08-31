// Package dotenv charge des variables d'environnement depuis un fichier .env.
//
// gofact ne code en dur aucune identité ni aucun secret : le vendeur, l'IBAN de
// règlement et les identifiants PDP viennent tous de l'environnement, qu'il soit
// renseigné par le shell ou par un fichier .env (jamais versionné).
package dotenv

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Load lit un fichier .env et exporte ses clés dans l'environnement, sans jamais
// écraser une variable déjà définie. Un fichier absent n'est pas une erreur.
func Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("dotenv: lecture %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if _, set := os.LookupEnv(k); !set {
			_ = os.Setenv(k, v)
		}
	}
	return sc.Err()
}

// LoadDefault charge le .env explicite s'il est fourni, sinon les emplacements
// conventionnels : ./.env puis $XDG_CONFIG_HOME/gofact/.env (~/.config par
// défaut). Le premier chargé gagne — Load n'écrase jamais une clé déjà posée.
func LoadDefault(explicit string) error {
	if explicit != "" {
		return Load(explicit)
	}
	for _, p := range defaultPaths() {
		if err := Load(p); err != nil {
			return err
		}
	}
	return nil
}

func defaultPaths() []string {
	paths := []string{".env"}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, ".config")
		}
	}
	if dir != "" {
		paths = append(paths, filepath.Join(dir, "gofact", ".env"))
	}
	return paths
}
