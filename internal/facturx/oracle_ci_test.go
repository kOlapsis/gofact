//go:build ci

package facturx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Oracle de conformité — intégration continue uniquement.
//
// Mustang (qui embarque veraPDF et le Schematron EN 16931) est la seule
// vérification externe et indépendante que gofact produit bien des factures
// conformes. Il exige un JRE et un JAR de 56 Mo : hors de question de le faire
// peser sur l'utilisateur. Il vit donc ici, derrière le tag de compilation `ci`,
// et ne sert qu'à confirmer que notre auto-contrôle en Go ne se ment pas à
// lui-même.
//
//	go test -tags=ci ./internal/facturx -run TestOracle

const mustangVersion = "2.23.1"

var verdictLine = regexp.MustCompile(`PDF:(valid|invalid)\s+XML:(valid|invalid)`)

func TestOracleMustangValidatesOutput(t *testing.T) {
	html := os.Getenv("GOFACT_ORACLE_HTML")
	if html == "" {
		t.Skip("GOFACT_ORACLE_HTML non défini (chemin d'une facture HTML de test)")
	}
	spec, err := LoadSpec(strings.TrimSuffix(html, filepath.Ext(html)) + ".json")
	if err != nil {
		t.Fatalf("spec: %v", err)
	}
	inv, err := spec.ToInvoice()
	if err != nil {
		t.Fatalf("ToInvoice: %v", err)
	}

	out := filepath.Join(t.TempDir(), "oracle.pdf")
	ctx := context.Background()
	res, err := Generate(ctx, inv, Options{HTMLPath: html, OutPath: out, Validate: true})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !res.Valid {
		t.Fatalf("auto-contrôle Go en échec :\n%s", res.Report)
	}

	jar, err := ensureMustang(ctx, t)
	if err != nil {
		t.Fatalf("Mustang: %v", err)
	}
	report := runMustang(ctx, jar, out)
	m := verdictLine.FindStringSubmatch(report)
	if len(m) != 3 || m[1] != "valid" || m[2] != "valid" {
		t.Fatalf("Mustang refuse le document produit :\n%s", summarize(report))
	}
	if !strings.Contains(report, "flavour=3b") || !strings.Contains(report, "isCompliant=true") {
		t.Errorf("veraPDF ne confirme pas PDF/A-3b :\n%s", summarize(report))
	}
	t.Logf("Mustang : %s", m[0])
}

func ensureMustang(ctx context.Context, t *testing.T) (string, error) {
	t.Helper()
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	dir := filepath.Join(cacheDir, "gofact")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	jar := filepath.Join(dir, "Mustang-CLI-"+mustangVersion+".jar")
	if fi, err := os.Stat(jar); err == nil && fi.Size() > 0 {
		return jar, nil
	}

	url := fmt.Sprintf("https://repo1.maven.org/maven2/org/mustangproject/Mustang-CLI/%s/Mustang-CLI-%s.jar",
		mustangVersion, mustangVersion)
	t.Logf("téléchargement de Mustang %s…", mustangVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	cli := &http.Client{Timeout: 5 * time.Minute}
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("statut %d", resp.StatusCode)
	}
	tmp := jar + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return jar, os.Rename(tmp, jar)
}

func runMustang(ctx context.Context, jar, pdf string) string {
	cmd := exec.CommandContext(ctx, "java", "-jar", jar,
		"--action", "validate", "--source", pdf, "--disable-file-logging")
	out, _ := cmd.CombinedOutput() // code non nul si invalide ; on lit le verdict
	return string(out)
}

// summarize réduit le rapport au verdict et aux assertions en échec.
func summarize(report string) string {
	var b strings.Builder
	for _, line := range strings.Split(report, "\n") {
		t := strings.TrimSpace(line)
		if strings.Contains(t, "Parsed PDF:") || strings.Contains(t, "ValidationResult") ||
			(strings.Contains(t, "FailedAssert") && strings.Contains(t, "[BR-")) {
			b.WriteString(t + "\n")
		}
	}
	if b.Len() == 0 {
		return strings.TrimSpace(report)
	}
	return strings.TrimSpace(b.String())
}
