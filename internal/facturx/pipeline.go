package facturx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// mustangVersion est la version de la CLI Mustang (implémentation EN 16931 de
// référence) téléchargée à la demande. Identique au spike validé du projet facture.
const mustangVersion = "2.23.1"

// Options pilote la génération Factur-X.
type Options struct {
	HTMLPath   string // facture HTML prête à imprimer (placeholders déjà remplis)
	OutPath    string // PDF Factur-X de sortie
	ChromePath string // exécutable Chrome ; vide ⇒ auto-détection
	XMLOut     string // si non vide, écrit aussi le XML CII à ce chemin (debug)
	Validate   bool   // valide le résultat avec Mustang après assemblage
	Verbose    bool
}

// Result résume la sortie.
type Result struct {
	OutPath   string
	Valid     bool   // résultat de la validation (si demandée)
	Validated bool   // la validation a-t-elle été exécutée
	Report    string // extrait du rapport de validation
}

// Generate exécute la chaîne complète : HTML → PDF (Chrome) → PDF/A-3 +
// embarquement du XML CII (Go pur) → validation optionnelle.
func Generate(ctx context.Context, inv Invoice, opt Options) (Result, error) {
	var res Result
	res.OutPath = opt.OutPath

	work, err := os.MkdirTemp("", "gofact-")
	if err != nil {
		return res, fmt.Errorf("facturx: répertoire de travail: %w", err)
	}
	defer os.RemoveAll(work)

	// 1 — XML CII (EN 16931)
	xml, err := BuildCII(inv)
	if err != nil {
		return res, err
	}
	if opt.XMLOut != "" {
		if err := os.WriteFile(opt.XMLOut, xml, 0o644); err != nil {
			return res, fmt.Errorf("facturx: écriture XML: %w", err)
		}
	}
	logv(opt, "✓ XML CII généré (%d octets)", len(xml))

	// 2 — HTML → PDF (Chrome headless, fidèle au Ctrl+P)
	rawPDF, err := renderHTML(ctx, opt.HTMLPath, opt.ChromePath)
	if err != nil {
		return res, err
	}
	basePDF := filepath.Join(work, "base.pdf")
	if err := os.WriteFile(basePDF, rawPDF, 0o644); err != nil {
		return res, fmt.Errorf("facturx: écriture PDF rendu: %w", err)
	}
	logv(opt, "✓ PDF rendu via Chrome (%d octets)", len(rawPDF))

	// 3 — PDF/A-3 + embarquement verbatim du XML (Go pur, cf. assemble.go)
	// On embarque le XML tel quel : un assembleur qui le re-sérialise via son
	// modèle perdrait les champs étendus (adresses de routage PDP, BT-34/BT-49).
	xmlPath := filepath.Join(work, "factur-x.xml")
	if err := os.WriteFile(xmlPath, xml, 0o644); err != nil {
		return res, fmt.Errorf("facturx: écriture XML temp: %w", err)
	}
	if err := embedFacturX(basePDF, xmlPath, opt.OutPath, inv.IssueDate); err != nil {
		return res, err
	}
	logv(opt, "✓ Factur-X assemblé (PDF/A-3, XML verbatim) → %s", opt.OutPath)

	// 4 — Validation (optionnelle, via Mustang)
	if opt.Validate {
		jar, err := ensureMustang(ctx, opt)
		if err != nil {
			return res, err
		}
		valid, report, err := mustangValidate(ctx, jar, opt.OutPath)
		if err != nil {
			return res, err
		}
		res.Validated = true
		res.Valid = valid
		res.Report = report
	}
	return res, nil
}

// renderHTML rend un fichier HTML en PDF via Chrome headless (CDP). Force
// l'impression des arrière-plans et honore le @page dynamique du template
// (taille recalculée pour tenir sur une page) via preferCSSPageSize.
func renderHTML(ctx context.Context, htmlPath, chromePath string) ([]byte, error) {
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return nil, fmt.Errorf("facturx: chemin HTML: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("facturx: HTML introuvable: %w", err)
	}
	// Encode le chemin (espaces, apostrophes…) en URL file:// valide.
	fileURL := (&url.URL{Scheme: "file", Path: abs}).String()

	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	opts = append(opts,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
	)
	if chromePath == "" {
		chromePath = detectChrome()
	}
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()
	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()
	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, 60*time.Second)
	defer cancelTimeout()

	// Recalcule la taille de page après chargement des polices (le handler
	// 'load' du template peut mesurer avant que la webfont Inter ne soit prête).
	const refit = `(function(){
		var p=document.querySelector('.page'); if(!p) return -1;
		var el=document.getElementById('page-size'); if(!el) return -1;
		var prev=p.style.minHeight; p.style.minHeight='auto';
		var px=p.getBoundingClientRect().height; p.style.minHeight=prev;
		var mm=Math.ceil(px*25.4/96)+3;
		el.textContent='@page { size: 210mm '+mm+'mm; margin: 0; }';
		return mm;
	})()`

	var pdf []byte
	err = chromedp.Run(taskCtx,
		chromedp.Navigate(fileURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var ready bool
			_ = chromedp.Evaluate(`document.fonts.ready.then(()=>true)`, &ready,
				func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) },
			).Do(ctx)
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var mm float64
			return chromedp.Evaluate(refit, &mm).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithMarginTop(0).WithMarginBottom(0).
				WithMarginLeft(0).WithMarginRight(0).
				Do(ctx)
			if err != nil {
				return err
			}
			pdf = buf
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("facturx: rendu Chrome: %w", err)
	}
	return pdf, nil
}

// detectChrome privilégie un Chrome/Chromium « classique » (deb) au chromium snap,
// ce dernier étant confiné et incapable de lire des fichiers hors de $HOME (/tmp…).
func detectChrome() string {
	candidates := []string{
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/opt/google/chrome/chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			// Évite le wrapper snap (lien vers /snap/…) qui est confiné.
			if real, err := filepath.EvalSymlinks(c); err == nil && strings.HasPrefix(real, "/snap/") {
				continue
			}
			return c
		}
	}
	if p, err := exec.LookPath("google-chrome"); err == nil {
		return p
	}
	return "" // laisse chromedp tenter sa propre détection
}

// facturxConformance est le niveau de conformité inscrit dans le XMP Factur-X.
// Aligné sur le guideline (BT-24) du XML CII produit par BuildCII.
const facturxConformance = "EN 16931"

// ensureMustang renvoie le chemin du jar Mustang, le téléchargeant dans le cache
// utilisateur s'il est absent.
func ensureMustang(ctx context.Context, opt Options) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	dir := filepath.Join(cacheDir, "gofact")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("facturx: cache: %w", err)
	}
	jar := filepath.Join(dir, "Mustang-CLI-"+mustangVersion+".jar")
	if fi, err := os.Stat(jar); err == nil && fi.Size() > 0 {
		return jar, nil
	}
	url := fmt.Sprintf("https://repo1.maven.org/maven2/org/mustangproject/Mustang-CLI/%s/Mustang-CLI-%s.jar",
		mustangVersion, mustangVersion)
	logv(opt, "→ Téléchargement de Mustang CLI %s…", mustangVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("facturx: requête Mustang: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("facturx: téléchargement Mustang: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("facturx: téléchargement Mustang: statut %d", resp.StatusCode)
	}
	tmp := jar + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return "", fmt.Errorf("facturx: création jar: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("facturx: écriture jar: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, jar); err != nil {
		return "", fmt.Errorf("facturx: finalisation jar: %w", err)
	}
	return jar, nil
}

// verdictLine capte le verdict synthétique de Mustang : « …PDF:valid XML:valid… ».
var verdictLine = regexp.MustCompile(`PDF:(valid|invalid)\s+XML:(valid|invalid)`)

func mustangValidate(ctx context.Context, jar, pdf string) (bool, string, error) {
	args := []string{"-jar", jar, "--action", "validate", "--source", pdf, "--disable-file-logging"}
	cmd := exec.CommandContext(ctx, "java", args...)
	out, _ := cmd.CombinedOutput() // code de sortie non nul si invalide ; on lit le verdict
	report := string(out)
	m := verdictLine.FindStringSubmatch(report)
	valid := len(m) == 3 && m[1] == "valid" && m[2] == "valid"
	return valid, summarize(report), nil
}

// summarize extrait l'essentiel du rapport : le verdict et les assertions en échec
// (hors notices non françaises), pour un diagnostic lisible en cas d'invalidité.
func summarize(report string) string {
	var b strings.Builder
	for _, line := range strings.Split(report, "\n") {
		t := strings.TrimSpace(line)
		if strings.Contains(t, "Parsed PDF:") {
			b.WriteString(t)
			b.WriteString("\n")
		}
		if strings.Contains(t, "FailedAssert") && (strings.Contains(t, "ERROR") || strings.Contains(t, "[BR-")) {
			b.WriteString(t)
			b.WriteString("\n")
		}
	}
	if b.Len() == 0 {
		return strings.TrimSpace(report)
	}
	return strings.TrimSpace(b.String())
}

func logv(opt Options, format string, a ...any) {
	if opt.Verbose {
		fmt.Fprintf(os.Stderr, format+"\n", a...)
	}
}
