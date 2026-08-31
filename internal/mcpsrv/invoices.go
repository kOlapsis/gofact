package mcpsrv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kolapsis/gofact/internal/facturx"
	"github.com/kolapsis/gofact/internal/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// numberToken est le jeton que l'IA place dans son HTML à l'emplacement du
// numéro : c'est le serveur qui attribue le numéro, jamais le modèle.
const numberToken = "{{NUMERO}}"

func addInvoiceTools(s *mcp.Server) {
	type previewOut struct {
		Number string `json:"number"`
		Note   string `json:"note"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "preview_next_number",
		Description: "Prochain numéro de facture, SANS le consommer — pour l'annoncer à l'utilisateur " +
			"avant confirmation. Seul create_invoice attribue réellement le numéro.",
		Annotations: readOnly("Prochain numéro"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in orgParam) (*mcp.CallToolResult, previewOut, error) {
		o, err := resolveOrg(in.Org)
		if err != nil {
			return nil, previewOut{}, err
		}
		n, err := o.NextNumber(time.Now())
		if err != nil {
			return nil, previewOut{}, err
		}
		return nil, previewOut{Number: n, Note: "Indicatif : l'attribution définitive a lieu dans create_invoice. " +
			"Dans le HTML, écrire le jeton " + numberToken + ", pas ce numéro."}, nil
	})

	type listIn struct {
		orgParam
		Year   string `json:"year,omitempty" jsonschema:"filtrer par année d'émission (YYYY)"`
		Client string `json:"client,omitempty" jsonschema:"filtrer par nom de client (sous-chaîne)"`
		Limit  int    `json:"limit,omitempty" jsonschema:"nombre maximal d'entrées (défaut 20)"`
	}
	type listOut struct {
		Invoices []map[string]any `json:"invoices"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_invoices",
		Description: "Factures inscrites au registre de l'organisation, de la plus récente à la plus ancienne.",
		Annotations: readOnly("Lister les factures"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in listIn) (*mcp.CallToolResult, listOut, error) {
		o, err := resolveOrg(in.Org)
		if err != nil {
			return nil, listOut{}, err
		}
		all, err := o.Invoices()
		if err != nil {
			return nil, listOut{}, err
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 20
		}
		out := listOut{Invoices: []map[string]any{}}
		for _, inv := range all {
			if in.Year != "" && !strings.HasPrefix(str(inv["numero"]), in.Year) {
				continue
			}
			if in.Client != "" && !strings.Contains(strings.ToLower(str(inv["client"])), strings.ToLower(in.Client)) {
				continue
			}
			out.Invoices = append(out.Invoices, inv)
			if len(out.Invoices) >= limit {
				break
			}
		}
		return nil, out, nil
	})

	type createIn struct {
		orgParam
		HTML string       `json:"html" jsonschema:"la facture HTML complète et imprimable (A4, CSS embarqué), contenant le jeton {{NUMERO}} à l'emplacement du numéro"`
		Spec facturx.Spec `json:"spec" jsonschema:"les données structurées de la facture (montants en CENTIMES) ; ne pas renseigner number, le serveur l'attribue"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_invoice",
		Description: "Crée la facture : attribue le numéro (séquence légale, continue), écrit le HTML et " +
			"les données dans le dossier de l'organisation, génère le PDF Factur-X (PDF/A-3 + XML " +
			"EN 16931) et inscrit la facture au registre — le tout en une transaction. Avant d'appeler : " +
			"faire valider le contenu (client, lignes, montants, date) par l'utilisateur, en lui " +
			"annonçant le numéro via preview_next_number.",
		Annotations: writes("Créer une facture"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in createIn) (*mcp.CallToolResult, createOutT, error) {
		o, err := resolveOrg(in.Org)
		if err != nil {
			return nil, createOutT{}, err
		}
		out, err := createInvoice(ctx, o, in.HTML, in.Spec)
		return nil, out, err
	})
}

// createInvoice est la transaction complète : tout ce qui peut échouer échoue
// AVANT de consommer le numéro (validation des règles) ou SOUS le verrou
// (rendu, assemblage) — un échec ne laisse jamais de trou dans la séquence ni
// de fichier orphelin.
func createInvoice(ctx context.Context, o *workspace.Org, html string, spec facturx.Spec) (createOutT, error) {
	if strings.TrimSpace(html) == "" {
		return createOutT{}, fmt.Errorf("le HTML de la facture est vide. Composer d'abord la facture — " +
			"récupérer le modèle avec get_invoice_template")
	}
	if !strings.Contains(html, numberToken) {
		return createOutT{}, fmt.Errorf("le HTML ne contient pas le jeton %s : le serveur ne peut pas y "+
			"inscrire le numéro attribué. Placer %s à l'emplacement du numéro de facture", numberToken, numberToken)
	}
	cfg := o.Config()

	// Validation des règles AVANT toute attribution : un numéro n'est consommé
	// que pour une facture qu'on sait produisible.
	probe := spec
	probe.Number = "0000000"
	inv, err := probe.ToInvoiceWith(cfg)
	if err != nil {
		return createOutT{}, err
	}
	if err := inv.Validate(); err != nil {
		return createOutT{}, err
	}

	var out createOutT
	client := strings.TrimSpace(spec.Buyer.Name)
	now := time.Now()

	_, err = o.AllocateWith(now, func(number string) (workspace.RegistryEntry, error) {
		final := strings.ReplaceAll(html, numberToken, number)
		spec.Number = number
		if spec.IssueDate == "" {
			spec.IssueDate = now.Format("2006-01-02")
		}
		inv, err := spec.ToInvoiceWith(cfg)
		if err != nil {
			return workspace.RegistryEntry{}, err
		}

		base := fmt.Sprintf("%s - %s", number, sanitizeFilename(client))
		htmlPath := filepath.Join(o.Path, base+".html")
		if err := os.WriteFile(htmlPath, []byte(final), 0o644); err != nil {
			return workspace.RegistryEntry{}, err
		}
		if err := writeSpec(filepath.Join(o.Path, base+".json"), spec); err != nil {
			return workspace.RegistryEntry{}, err
		}
		pdfPath := filepath.Join(o.Path, base+".pdf")

		res, err := facturx.Generate(ctx, inv, facturx.Options{
			HTMLPath: htmlPath,
			OutPath:  pdfPath,
			Validate: true,
		})
		if err != nil {
			// Transaction abandonnée : on retire ce qu'on a posé.
			_ = os.Remove(htmlPath)
			_ = os.Remove(filepath.Join(o.Path, base+".json"))
			return workspace.RegistryEntry{}, err
		}
		if !res.Valid {
			_ = os.Remove(htmlPath)
			_ = os.Remove(filepath.Join(o.Path, base+".json"))
			_ = os.Remove(pdfPath)
			return workspace.RegistryEntry{}, fmt.Errorf("le Factur-X produit a échoué à l'auto-contrôle — "+
				"facture non émise, numéro non consommé :\n%s", res.Report)
		}

		out = createOutT{
			Number:        number,
			PDFPath:       pdfPath,
			HTMLPath:      htmlPath,
			TotalHTCents:  inv.LineTotal,
			TotalTTCCents: inv.GrandTotal,
			Conform:       true,
		}
		return workspace.RegistryEntry{
			Client:    client,
			Projet:    firstLineName(spec),
			MontantHT: inv.LineTotal,
			Fichier:   base + ".html",
			DevisRef:  spec.BuyerRef,
		}, nil
	})
	if err != nil {
		return createOutT{}, err
	}

	// Modèle : figé à la première facture (avec son jeton, pas le numéro
	// attribué), dérive signalée ensuite.
	if frozen, ferr := o.FreezeTemplate(html); ferr == nil && frozen {
		out.TemplateFrozen = true
	} else if ferr == nil {
		if warn, _ := o.TemplateDrift(html); warn != "" {
			out.Warnings = append(out.Warnings, warn)
		}
	}
	return out, nil
}

// createOutT double le type anonyme de l'outil pour que createInvoice reste
// testable hors SDK.
type createOutT struct {
	Number         string   `json:"number"`
	PDFPath        string   `json:"pdf_path"`
	HTMLPath       string   `json:"html_path"`
	TotalHTCents   int64    `json:"total_ht_cents"`
	TotalTTCCents  int64    `json:"total_ttc_cents"`
	Conform        bool     `json:"factur_x_conform"`
	TemplateFrozen bool     `json:"template_frozen,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

func writeSpec(path string, spec facturx.Spec) error {
	raw, err := facturx.MarshalSpec(spec)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// sanitizeFilename garde le nom du client lisible mais sûr pour un nom de
// fichier sur les trois OS.
func sanitizeFilename(name string) string {
	if name == "" {
		return "Client"
	}
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "-")
	return strings.TrimSpace(r.Replace(name))
}

func firstLineName(spec facturx.Spec) string {
	if len(spec.Lines) > 0 {
		return spec.Lines[0].Name
	}
	return ""
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
