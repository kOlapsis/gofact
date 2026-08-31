package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Sous-commande install : enregistre gofact comme serveur MCP dans les clients
// installés sur le poste. C'est ce qui remplace le « double-clic » qui n'existe
// pas encore pour les serveurs MCP locaux : une commande, appelable aussi bien
// par l'utilisateur que par une IA, qui écrit les configurations à sa place.
//
// Prudence obligatoire : ces fichiers appartiennent à l'utilisateur et à
// d'autres outils. On ne modifie JAMAIS sans montrer ce qui va changer (défaut
// -dry-run), on sauvegarde avant d'écrire, et on ne touche pas à une entrée
// « gofact » existante qui pointerait ailleurs, sauf -force.
func runInstall(argv []string) {
	fs := flag.NewFlagSet("gofact install", flag.ExitOnError)
	apply := fs.Bool("yes", false, "applique les modifications (défaut : les afficher seulement)")
	force := fs.Bool("force", false, "remplace une entrée gofact existante qui pointe ailleurs")
	_ = fs.Parse(argv)

	exe, err := os.Executable()
	if err != nil {
		fail(err)
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		fail(err)
	}

	clients := detectClients()
	if len(clients) == 0 {
		fmt.Println("Aucun client MCP détecté (Claude Desktop, Claude Code, LM Studio, Cursor).")
		fmt.Println("Installez-en un puis relancez : gofact install -yes")
		return
	}

	changed := 0
	for _, c := range clients {
		did, err := c.register(exe, *apply, *force)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ %-14s %v\n", c.name, err)
			continue
		}
		if did {
			changed++
		}
	}
	if !*apply && changed > 0 {
		fmt.Println("\nRien n'a été modifié. Pour appliquer : gofact install -yes")
	}
}

// mcpClient est un client MCP détecté sur le poste.
type mcpClient struct {
	name     string
	register func(exe string, apply, force bool) (bool, error)
}

func detectClients() []mcpClient {
	var out []mcpClient

	// Claude Code : sa CLI gère elle-même sa configuration — on la laisse faire.
	if path, err := exec.LookPath("claude"); err == nil {
		out = append(out, mcpClient{name: "Claude Code", register: func(exe string, apply, _ bool) (bool, error) {
			if !apply {
				fmt.Printf("→ %-14s claude mcp add --scope user gofact -- %q mcp\n", "Claude Code", exe)
				return true, nil
			}
			cmd := exec.Command(path, "mcp", "add", "--scope", "user", "gofact", "--", exe, "mcp")
			if outb, err := cmd.CombinedOutput(); err != nil {
				return false, fmt.Errorf("claude mcp add : %s", string(outb))
			}
			fmt.Printf("✓ %-14s serveur « gofact » enregistré (portée utilisateur)\n", "Claude Code")
			return true, nil
		}})
	}

	// Clients à fichier mcpServers { "gofact": {command, args} }.
	for _, fc := range fileClients() {
		if _, err := os.Stat(filepath.Dir(fc.path)); err != nil {
			continue // client non installé
		}
		fc := fc
		out = append(out, mcpClient{name: fc.name, register: func(exe string, apply, force bool) (bool, error) {
			return registerInFile(fc.name, fc.path, exe, apply, force)
		}})
	}
	return out
}

type fileClient struct{ name, path string }

// fileClients liste les clients configurés par un fichier JSON, aux
// emplacements qu'ils documentent, par plateforme.
func fileClients() []fileClient {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var desktop string
	switch runtime.GOOS {
	case "darwin":
		desktop = filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		desktop = filepath.Join(os.Getenv("APPDATA"), "Claude", "claude_desktop_config.json")
	default:
		desktop = filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	}
	return []fileClient{
		{"Claude Desktop", desktop},
		{"LM Studio", filepath.Join(home, ".lmstudio", "mcp.json")},
		{"Cursor", filepath.Join(home, ".cursor", "mcp.json")},
	}
}

// registerInFile fusionne l'entrée gofact dans un fichier de configuration
// mcpServers, sans jamais perdre le reste du fichier.
func registerInFile(name, path, exe string, apply, force bool) (bool, error) {
	cfg := map[string]json.RawMessage{}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return false, fmt.Errorf("%s illisible — non modifié : %w", path, err)
		}
	}
	servers := map[string]json.RawMessage{}
	if raw, ok := cfg["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return false, fmt.Errorf("mcpServers illisible dans %s : %w", path, err)
		}
	}

	entry := map[string]any{"command": exe, "args": []string{"mcp"}}
	wanted, _ := json.Marshal(entry)

	if existing, ok := servers["gofact"]; ok {
		if jsonEqual(existing, wanted) {
			fmt.Printf("✓ %-14s déjà configuré (%s)\n", name, path)
			return false, nil
		}
		if !force {
			return false, fmt.Errorf("une entrée « gofact » différente existe déjà dans %s — relancer avec -force pour la remplacer", path)
		}
	}

	if !apply {
		fmt.Printf("→ %-14s ajouterait mcpServers.gofact = {command: %q, args: [\"mcp\"]} dans %s\n", name, exe, path)
		return true, nil
	}

	servers["gofact"] = wanted
	raw, err := json.Marshal(servers)
	if err != nil {
		return false, err
	}
	cfg["mcpServers"] = raw
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, err
	}

	// Sauvegarde horodatée avant toute écriture, puis écriture atomique.
	if _, err := os.Stat(path); err == nil {
		backup := fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102-150405"))
		if prev, err := os.ReadFile(path); err == nil {
			_ = os.WriteFile(backup, prev, 0o600)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o600); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return false, err
	}
	fmt.Printf("✓ %-14s configuré (%s — sauvegarde .bak conservée)\n", name, path)
	return true, nil
}

func jsonEqual(a, b []byte) bool {
	var va, vb any
	if json.Unmarshal(a, &va) != nil || json.Unmarshal(b, &vb) != nil {
		return false
	}
	ja, _ := json.Marshal(va)
	jb, _ := json.Marshal(vb)
	return string(ja) == string(jb)
}
