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

// Parse lit un fichier .env et renvoie ses clés sans toucher à l'environnement
// du processus. C'est la forme qui convient à un serveur qui sert plusieurs
// organisations : chacune garde son fichier, aucune ne fuit dans les autres.
// Un fichier absent n'est pas une erreur et renvoie une table vide.
func Parse(path string) (map[string]string, error) {
	vars := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return vars, nil
		}
		return nil, fmt.Errorf("dotenv: lecture %s: %w", path, err)
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
		vars[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return vars, sc.Err()
}

// Load lit un fichier .env et exporte ses clés dans l'environnement, sans jamais
// écraser une variable déjà définie. Un fichier absent n'est pas une erreur.
func Load(path string) error {
	vars, err := Parse(path)
	if err != nil {
		return err
	}
	for k, v := range vars {
		if _, set := os.LookupEnv(k); !set {
			_ = os.Setenv(k, v)
		}
	}
	return nil
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
