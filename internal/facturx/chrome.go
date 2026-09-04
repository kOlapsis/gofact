package facturx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Localisation du navigateur qui fait le rendu. C'est la seule dépendance
// externe de gofact, et elle est délibérément conservée : réimplémenter un
// moteur de mise en page CSS n'est pas envisageable, et Chrome est déjà présent
// sur la quasi-totalité des postes — sous une forme ou une autre, puisque Edge,
// Brave et Chromium partagent le même moteur (Blink/Skia) et produisent donc le
// même PDF.
//
// On ne télécharge jamais de navigateur : c'est plusieurs centaines de mégaoctets
// à l'insu de l'utilisateur. En cas d'absence, on explique quoi installer.

// envChrome permet de désigner explicitement l'exécutable à utiliser.
const envChrome = "GOFACT_CHROME"

// detectChrome renvoie le chemin d'un navigateur utilisable, ou "" si aucun n'est
// trouvé — auquel cas chromedp tentera sa propre détection.
func detectChrome() string {
	if p := strings.TrimSpace(os.Getenv(envChrome)); p != "" {
		return p
	}
	for _, c := range browserCandidates() {
		if usable(c) {
			return c
		}
	}
	// Dernier recours : le PATH, sous les noms les plus courants.
	for _, name := range browserCommands() {
		if p, err := exec.LookPath(name); err == nil && usable(p) {
			return p
		}
	}
	return ""
}

// BrowserAvailable indique si un navigateur de rendu est joignable sur ce poste.
// Les tests de bout en bout s'en servent pour se dispenser plutôt que d'échouer
// là où la machine n'a simplement pas de navigateur à leur offrir.
func BrowserAvailable() bool { return detectChrome() != "" }

// usable écarte les répertoires et les paquets confinés (snap, flatpak), qui ne
// peuvent pas lire les fichiers temporaires que gofact leur soumet.
func usable(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return false
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return true // pas de lien à suivre : on garde le candidat
	}
	for _, confined := range []string{"/snap/", "/var/lib/flatpak/", "/var/lib/snapd/"} {
		if strings.HasPrefix(real, confined) {
			return false
		}
	}
	return true
}

// chromeMissingError explique, selon la plateforme, comment débloquer la
// situation. Le message est destiné à être relayé tel quel à l'utilisateur.
func chromeMissingError() error {
	var hint string
	switch runtime.GOOS {
	case "darwin":
		hint = "installez Google Chrome (https://google.com/chrome) ou Microsoft Edge"
	case "windows":
		hint = "Microsoft Edge est normalement préinstallé ; sinon installez Google Chrome (https://google.com/chrome)"
	default:
		hint = "installez Google Chrome (https://google.com/chrome) ou le paquet chromium de votre distribution " +
			"— attention, la version snap est confinée et inutilisable"
	}
	// Le recours proposé doit exister dans les deux modes. En MCP il n'y a pas
	// de ligne de commande où passer -chrome : c'est le .env qui sert, et il est
	// désormais lu par `gofact mcp` comme par le mode direct.
	return fmt.Errorf("facturx: aucun navigateur trouvé pour le rendu : %s. "+
		"Vous pouvez aussi désigner l'exécutable en posant %s=/chemin/vers/chrome "+
		"dans ./.env ou ~/.config/gofact/.env (ou avec l'option -chrome en ligne de commande)",
		hint, envChrome)
}
