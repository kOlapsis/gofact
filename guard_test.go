package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kolapsis/gofact/internal/workspace"
)

// Le mode direct ne doit jamais pouvoir émettre une facture que le registre
// ignore : c'est la garantie de numérotation qui saute, pas un détail d'ergonomie.
func TestGuardRegistry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	dir := t.TempDir()
	if err := guardRegistry(filepath.Join(dir, "brouillon.html")); err != nil {
		t.Errorf("hors d'un dossier d'organisation, le mode direct reste libre : %v", err)
	}

	org, err := workspace.Init(filepath.Join(t.TempDir(), "orga"), map[string]string{
		"GOFACT_SELLER_NAME": "Studio Exemple",
	}, "")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	err = guardRegistry(filepath.Join(org.Path, "2026001 - ACME.html"))
	if err == nil {
		t.Fatal("un fichier inconnu du registre doit être refusé dans un dossier d'organisation")
	}
	if !strings.Contains(err.Error(), "create_invoice") {
		t.Errorf("le refus doit indiquer la sortie : %v", err)
	}

	number, err := org.Allocate(time.Now(), workspace.RegistryEntry{
		Client:  "ACME SAS",
		Fichier: "2026001 - ACME.html",
	})
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if number != "2026001" {
		t.Fatalf("numéro inattendu : %s", number)
	}
	if err := guardRegistry(filepath.Join(org.Path, "2026001 - ACME.html")); err != nil {
		t.Errorf("le re-rendu d'une facture déjà inscrite doit passer : %v", err)
	}
}
