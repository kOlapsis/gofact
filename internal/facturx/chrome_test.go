package facturx

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDetectChromeHonoursOverride(t *testing.T) {
	// Un exécutable désigné explicitement doit primer sur toute détection.
	exe := filepath.Join(t.TempDir(), "navigateur")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envChrome, exe)
	if got := detectChrome(); got != exe {
		t.Errorf("detectChrome() = %q, want %q", got, exe)
	}
}

func TestUsableRejectsDirectoriesAndMissing(t *testing.T) {
	dir := t.TempDir()
	if usable(dir) {
		t.Error("un répertoire ne doit pas être retenu comme navigateur")
	}
	if usable(filepath.Join(dir, "absent")) {
		t.Error("un chemin inexistant ne doit pas être retenu")
	}
}

func TestUsableRejectsConfinedPackages(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("le confinement snap/flatpak ne concerne que Linux")
	}
	// Un lien vers /snap/... est écarté : le paquet confiné ne peut pas lire les
	// fichiers temporaires que gofact lui soumet.
	dir := t.TempDir()
	link := filepath.Join(dir, "chromium")
	if err := os.Symlink("/snap/chromium/current/bin/chromium", link); err != nil {
		t.Skipf("lien symbolique impossible : %v", err)
	}
	if usable(link) {
		t.Error("un navigateur confiné dans /snap doit être écarté")
	}
}

func TestBrowserCandidatesCoverEdge(t *testing.T) {
	// Edge est la garantie que gofact fonctionne sur un Windows nu.
	joined := strings.ToLower(strings.Join(browserCandidates(), " "))
	want := map[string]string{
		"windows": "msedge",
		"darwin":  "microsoft edge",
		"linux":   "microsoft-edge",
	}[runtime.GOOS]
	if want != "" && !strings.Contains(joined, want) {
		t.Errorf("les candidats %s devraient inclure Edge (%q)", runtime.GOOS, want)
	}
	if len(browserCommands()) == 0 {
		t.Error("au moins un nom de commande doit être proposé en repli")
	}
}

func TestChromeMissingErrorIsActionable(t *testing.T) {
	msg := chromeMissingError().Error()
	for _, want := range []string{envChrome, "-chrome", "google.com/chrome"} {
		if !strings.Contains(msg, want) {
			t.Errorf("le message doit mentionner %q :\n%s", want, msg)
		}
	}
}
