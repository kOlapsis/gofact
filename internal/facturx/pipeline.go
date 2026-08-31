package facturx

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// Options pilote la génération Factur-X.
type Options struct {
	HTMLPath   string // facture HTML prête à imprimer (placeholders déjà remplis)
	OutPath    string // PDF Factur-X de sortie
	ChromePath string // exécutable Chrome ; vide ⇒ auto-détection
	XMLOut     string // si non vide, écrit aussi le XML CII à ce chemin (debug)
	Validate   bool   // relit et vérifie le Factur-X produit (auto-contrôle)
	Verbose    bool
}

// Result résume la sortie.
type Result struct {
	OutPath   string
	Valid     bool   // résultat de la validation (si demandée)
	Validated bool   // la validation a-t-elle été exécutée
	Report    string // extrait du rapport de validation
}

// Generate exécute la chaîne complète : règles EN 16931 → XML CII → PDF (Chrome)
// → PDF/A-3 + embarquement du XML (Go pur) → auto-contrôle du résultat.
// Aucune dépendance externe hormis Chrome pour le rendu.
func Generate(ctx context.Context, inv Invoice, opt Options) (Result, error) {
	var res Result
	res.OutPath = opt.OutPath

	work, err := os.MkdirTemp("", "gofact-")
	if err != nil {
		return res, fmt.Errorf("facturx: répertoire de travail: %w", err)
	}
	defer os.RemoveAll(work)

	// 1 — Règles métier EN 16931, puis XML CII.
	// On refuse de produire un document qu'on sait non conforme plutôt que de le
	// valider après coup : l'erreur reste réparable et porte un identifiant de règle.
	if err := inv.Validate(); err != nil {
		return res, err
	}
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

	// 4 — Auto-contrôle : on relit le fichier écrit et on vérifie que les
	// structures Factur-X y sont, XML embarqué compris. Aucune dépendance
	// externe, quelques millisecondes.
	if opt.Validate {
		bad, err := SelfCheck(opt.OutPath, xml)
		if err != nil {
			return res, err
		}
		res.Validated = true
		res.Valid = len(bad) == 0
		res.Report = strings.Join(bad, "\n")
	}
	return res, nil
}

// RenderHTML rend un fichier HTML en PDF, sans assemblage Factur-X ni
// enregistrement d'aucune sorte — la brique d'aperçu, pour itérer sur un
// modèle de facture avant d'émettre quoi que ce soit.
func RenderHTML(ctx context.Context, htmlPath, chromePath string) ([]byte, error) {
	return renderHTML(ctx, htmlPath, chromePath)
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
	// Encode le chemin (espaces, apostrophes…) en URL file:// valide. Sous
	// Windows le chemin devient /C:/… : séparateurs en barres obliques et barre
	// initiale devant la lettre de lecteur, sinon Chrome répond ERR_INVALID_URL.
	p := filepath.ToSlash(abs)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	fileURL := (&url.URL{Scheme: "file", Path: p}).String()

	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	opts = append(opts,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
	)
	if chromePath == "" {
		chromePath = detectChrome()
	}
	if chromePath == "" {
		// Échouer ici avec une explication vaut mieux que de laisser chromedp
		// remonter une erreur de connexion incompréhensible.
		return nil, chromeMissingError()
	}
	opts = append(opts, chromedp.ExecPath(chromePath))

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

// facturxConformance est le niveau de conformité inscrit dans le XMP Factur-X.
// Aligné sur le guideline (BT-24) du XML CII produit par BuildCII.
const facturxConformance = "EN 16931"

func logv(opt Options, format string, a ...any) {
	if opt.Verbose {
		fmt.Fprintf(os.Stderr, format+"\n", a...)
	}
}
