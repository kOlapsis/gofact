//go:build darwin

package facturx

import (
	"os"
	"path/filepath"
)

// Sur macOS les navigateurs sont des bundles ; l'exécutable est enfoui dedans.
// On regarde les installations système puis celles propres à l'utilisateur.
func browserCandidates() []string {
	bundles := []string{
		"Google Chrome.app/Contents/MacOS/Google Chrome",
		"Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"Brave Browser.app/Contents/MacOS/Brave Browser",
		"Chromium.app/Contents/MacOS/Chromium",
	}
	roots := []string{"/Applications"}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, "Applications"))
	}
	var out []string
	for _, root := range roots {
		for _, b := range bundles {
			out = append(out, filepath.Join(root, b))
		}
	}
	return out
}

func browserCommands() []string {
	return []string{"google-chrome", "chromium"}
}
