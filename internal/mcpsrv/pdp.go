package mcpsrv

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kolapsis/gofact/internal/facturx"
	"github.com/kolapsis/gofact/internal/pdp"
	_ "github.com/kolapsis/gofact/internal/pdp/superpdp"
	"github.com/kolapsis/gofact/internal/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Envoi à la plateforme de dématérialisation (PDP). C'est le SEUL outil
// destructif du serveur : un dépôt est irréversible — la facture entre dans le
// circuit légal — d'où la confirmation explicite exigée.

// statusDelay est le délai laissé à la plateforme pour instruire le dépôt avant
// la première lecture du cycle de vie : un rejet arrive en moins d'une seconde.
// Variable pour que les tests ne l'attendent pas.
var statusDelay = time.Second

func addPDPTools(s *mcp.Server) {
	type sendIn struct {
		orgParam
		Number  string `json:"number" jsonschema:"numéro de la facture à déposer (ex. 2026007)"`
		Confirm bool   `json:"confirm" jsonschema:"doit valoir true, après confirmation EXPLICITE de l'utilisateur — un dépôt PDP est irréversible"`
	}
	type sendOut struct {
		Provider  string      `json:"provider"`
		Reference string      `json:"reference"`
		Events    []pdp.Event `json:"events"`
		Rejected  bool        `json:"rejected"`
		Reasons   []string    `json:"reasons,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "send_invoice",
		Description: "Dépose une facture émise sur la plateforme de dématérialisation (PDP) de " +
			"l'organisation. IRRÉVERSIBLE : la facture entre dans le circuit de la facturation " +
			"électronique. N'appeler qu'après un accord explicite de l'utilisateur, avec confirm=true. " +
			"Refuse un acheteur sans adresse de routage connue de l'annuaire, et remonte immédiatement " +
			"un rejet de la plateforme avec ses motifs.",
		Annotations: &mcp.ToolAnnotations{Title: "Déposer sur la PDP", ReadOnlyHint: false,
			DestructiveHint: boolPtr(true), OpenWorldHint: boolPtr(true)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in sendIn) (*mcp.CallToolResult, sendOut, error) {
		if !in.Confirm {
			return nil, sendOut{}, fmt.Errorf("dépôt refusé : confirm doit valoir true. Demander à "+
				"l'utilisateur une confirmation explicite (« déposer la facture %s sur la PDP ? ») "+
				"avant de rappeler cet outil", in.Number)
		}
		o, err := resolveOrg(in.Org)
		if err != nil {
			return nil, sendOut{}, err
		}
		pdf, err := invoicePDF(o, in.Number)
		if err != nil {
			return nil, sendOut{}, err
		}
		if err := checkRoutable(o, in.Number); err != nil {
			return nil, sendOut{}, err
		}
		provider, err := pdp.Open(o.Lookup)
		if err != nil {
			return nil, sendOut{}, err
		}
		receipt, err := provider.Send(ctx, pdf)
		if err != nil {
			return nil, sendOut{}, err
		}
		_ = o.Journal("pdp_sent", map[string]any{"numero": in.Number,
			"provider": receipt.Provider, "reference": receipt.Reference})

		out := sendOut{Provider: receipt.Provider, Reference: receipt.Reference, Events: receipt.Events}
		// Première lecture du cycle de vie : un rejet se manifeste tout de suite.
		// Le remonter maintenant, avec ses motifs, plutôt que de laisser
		// « api:uploaded » passer pour un dépôt réussi.
		if events := firstStatus(ctx, provider, receipt.Reference); len(events) > 0 {
			out.Events = events
		}
		out.Rejected, out.Reasons = pdp.Rejection(out.Events)
		if out.Rejected {
			_ = o.Journal("pdp_rejected", map[string]any{"numero": in.Number,
				"provider": receipt.Provider, "reference": receipt.Reference, "reasons": out.Reasons})
			return nil, out, rejectionError(in.Number, receipt.Reference, out.Reasons)
		}
		return nil, out, nil
	})

	type statusIn struct {
		orgParam
		Number string `json:"number" jsonschema:"numéro de la facture"`
	}
	type statusOut struct {
		Provider string      `json:"provider"`
		Events   []pdp.Event `json:"events"`
		Rejected bool        `json:"rejected"`
		Reasons  []string    `json:"reasons,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_invoice_status",
		Description: "Cycle de vie d'une facture déjà déposée sur la PDP (fr:200 déposée → fr:201 émise → " +
			"fr:202 reçue…). Un rejet (fr:213) est signalé avec ses motifs, règle par règle.",
		Annotations: &mcp.ToolAnnotations{Title: "Statut PDP", ReadOnlyHint: true, OpenWorldHint: boolPtr(true)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in statusIn) (*mcp.CallToolResult, statusOut, error) {
		o, err := resolveOrg(in.Org)
		if err != nil {
			return nil, statusOut{}, err
		}
		ref, provName, err := sentReference(o, in.Number)
		if err != nil {
			return nil, statusOut{}, err
		}
		provider, err := pdp.Open(o.Lookup)
		if err != nil {
			return nil, statusOut{}, err
		}
		events, err := provider.Status(ctx, ref)
		if err != nil {
			return nil, statusOut{}, err
		}
		out := statusOut{Provider: provName, Events: events}
		out.Rejected, out.Reasons = pdp.Rejection(events)
		return nil, out, nil
	})
}

// firstStatus lit une première fois le cycle de vie d'un dépôt, après le délai
// d'instruction de la plateforme. Une erreur de lecture n'est pas une erreur de
// dépôt : on renvoie simplement rien, l'accusé reste valable.
func firstStatus(ctx context.Context, provider pdp.Provider, reference string) []pdp.Event {
	if statusDelay > 0 {
		select {
		case <-time.After(statusDelay):
		case <-ctx.Done():
			return nil
		}
	}
	events, err := provider.Status(ctx, reference)
	if err != nil {
		return nil
	}
	return events
}

// rejectionError rédige le rejet pour l'utilisateur : la facture, le dépôt, et
// chaque motif sur sa ligne — tel que la plateforme l'a formulé.
func rejectionError(number, reference string, reasons []string) error {
	lines := make([]string, 0, len(reasons))
	for _, r := range reasons {
		lines = append(lines, "  - "+r)
	}
	return fmt.Errorf("la facture %s a été REJETÉE par la plateforme (dépôt %s). Motifs :\n%s\n\n"+
		"Le dépôt rejeté n'entre pas dans le circuit. Corriger la facture, la régénérer, puis la "+
		"déposer à nouveau", number, reference, strings.Join(lines, "\n"))
}

// checkRoutable refuse un dépôt dont l'acheteur n'a pas d'adresse de routage
// que l'annuaire connaisse (scheme 0225 ou 0002) : la plateforme le rejetterait,
// plus tard et moins clairement. La résolution est celle de la génération
// (ResolveRouting), pour qu'un sidecar sans adresse explicite mais avec un
// SIRET passe — c'est ce que porte son XML. Sans sidecar (facture antérieure à
// gofact), il n'y a rien à vérifier : on laisse la plateforme juger.
func checkRoutable(o *workspace.Org, number string) error {
	spec, path, err := invoiceSpec(o, number)
	if err != nil || path == "" {
		return nil
	}
	addr, scheme := facturx.ResolveRouting(spec.Buyer)
	if addr != "" && facturx.IsRoutableScheme(scheme) {
		return nil
	}
	what := "aucune adresse de routage"
	if addr != "" {
		what = fmt.Sprintf("pour seule adresse électronique %s (scheme %s)", addr, scheme)
	}
	return fmt.Errorf("dépôt refusé : l'acheteur de la facture %s a %s, or une plateforme française "+
		"route sur l'annuaire (scheme 0225 ou 0002) — le dépôt serait rejeté comme indélivrable. "+
		"Vérifier l'adressabilité du client avec find_routing_address (son SIREN), renseigner "+
		"electronic_address et electronic_address_scheme dans %s, régénérer le PDF, puis déposer",
		number, what, filepath.Base(path))
}

// invoiceBase retrouve, dans le registre, le chemin d'une facture sans son
// extension : la base commune du HTML, du sidecar JSON et du PDF.
func invoiceBase(o *workspace.Org, number string) (string, error) {
	invoices, err := o.Invoices()
	if err != nil {
		return "", err
	}
	for _, inv := range invoices {
		if str(inv["numero"]) != number {
			continue
		}
		file := str(inv["fichier"])
		if file == "" {
			return "", fmt.Errorf("la facture %s n'a pas de fichier associé au registre", number)
		}
		return filepath.Join(o.Path, strings.TrimSuffix(file, filepath.Ext(file))), nil
	}
	return "", fmt.Errorf("facture %s inconnue du registre — vérifier le numéro avec list_invoices", number)
}

// invoicePDF retrouve le PDF d'une facture inscrite au registre.
func invoicePDF(o *workspace.Org, number string) (string, error) {
	base, err := invoiceBase(o, number)
	if err != nil {
		return "", err
	}
	path := base + ".pdf"
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("PDF introuvable pour la facture %s (%s) : régénérer la facture ?", number, path)
	}
	return path, nil
}

// invoiceSpec charge le sidecar JSON d'une facture. path est vide, sans erreur,
// quand la facture n'en a pas — une entrée de registre antérieure à gofact.
func invoiceSpec(o *workspace.Org, number string) (facturx.Spec, string, error) {
	base, err := invoiceBase(o, number)
	if err != nil {
		return facturx.Spec{}, "", err
	}
	path := base + ".json"
	if _, err := os.Stat(path); err != nil {
		return facturx.Spec{}, "", nil
	}
	spec, err := facturx.LoadSpec(path)
	if err != nil {
		return facturx.Spec{}, path, err
	}
	return spec, path, nil
}

// sentReference retrouve, dans le journal, la référence et le fournisseur du
// dernier dépôt d'une facture.
func sentReference(o *workspace.Org, number string) (ref, provider string, err error) {
	f, err := os.Open(filepath.Join(o.Path, workspace.JournalFile))
	if err != nil {
		return "", "", fmt.Errorf("aucun dépôt PDP tracé pour cette organisation")
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var line struct {
			Event string `json:"event"`
			Data  struct {
				Numero    string `json:"numero"`
				Provider  string `json:"provider"`
				Reference string `json:"reference"`
			} `json:"data"`
		}
		if json.Unmarshal(sc.Bytes(), &line) != nil {
			continue
		}
		if line.Event == "pdp_sent" && line.Data.Numero == number {
			ref, provider = line.Data.Reference, line.Data.Provider // le plus récent gagne
		}
	}
	if ref == "" {
		return "", "", fmt.Errorf("la facture %s n'a pas été déposée sur la PDP (aucune trace au journal). "+
			"La déposer d'abord avec send_invoice", number)
	}
	return ref, provider, nil
}
