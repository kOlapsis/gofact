//go:build windows

package facturx

import (
	"os"
	"path/filepath"
)

// Edge est préinstallé sur toutes les éditions de Windows encore supportées :
// c'est le candidat qui garantit que gofact fonctionne sans rien installer.
func browserCandidates() []string {
	suffixes := []string{
		`Google\Chrome\Application\chrome.exe`,
		`Microsoft\Edge\Application\msedge.exe`,
		`BraveSoftware\Brave-Browser\Application\brave.exe`,
		`Chromium\Application\chrome.exe`,
	}
	var roots []string
	for _, v := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
		if d := os.Getenv(v); d != "" {
			roots = append(roots, d)
		}
	}
	var out []string
	for _, root := range roots {
		for _, s := range suffixes {
			out = append(out, filepath.Join(root, s))
		}
	}
	return out
}

func browserCommands() []string {
	return []string{"chrome.exe", "msedge.exe", "brave.exe"}
}
