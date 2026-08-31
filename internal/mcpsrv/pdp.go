package mcpsrv

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kolapsis/gofact/internal/pdp"
	_ "github.com/kolapsis/gofact/internal/pdp/superpdp"
	"github.com/kolapsis/gofact/internal/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Envoi à la plateforme de dématérialisation (PDP). C'est le SEUL outil
// destructif du serveur : un dépôt est irréversible — la facture entre dans le
// circuit légal — d'où la confirmation explicite exigée.

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
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "send_invoice",
		Description: "Dépose une facture émise sur la plateforme de dématérialisation (PDP) de " +
			"l'organisation. IRRÉVERSIBLE : la facture entre dans le circuit de la facturation " +
			"électronique. N'appeler qu'après un accord explicite de l'utilisateur, avec confirm=true.",
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
		return nil, sendOut{Provider: receipt.Provider, Reference: receipt.Reference, Events: receipt.Events}, nil
	})

	type statusIn struct {
		orgParam
		Number string `json:"number" jsonschema:"numéro de la facture"`
	}
	type statusOut struct {
		Provider string      `json:"provider"`
		Events   []pdp.Event `json:"events"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_invoice_status",
		Description: "Cycle de vie d'une facture déjà déposée sur la PDP (fr:200 déposée → fr:201 émise → " +
			"fr:202 reçue…).",
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
		return nil, statusOut{Provider: provName, Events: events}, nil
	})
}

// invoicePDF retrouve le PDF d'une facture inscrite au registre.
func invoicePDF(o *workspace.Org, number string) (string, error) {
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
		pdf := strings.TrimSuffix(file, filepath.Ext(file)) + ".pdf"
		path := filepath.Join(o.Path, pdf)
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("PDF introuvable pour la facture %s (%s) : régénérer la facture ?", number, path)
		}
		return path, nil
	}
	return "", fmt.Errorf("facture %s inconnue du registre — vérifier le numéro avec list_invoices", number)
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
