package mcpsrv

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kolapsis/gofact/internal/superpdp"
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
		Provider  string           `json:"provider"`
		Reference int64            `json:"reference"`
		Events    []superpdp.Event `json:"events"`
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
		cli, err := pdpClient(o)
		if err != nil {
			return nil, sendOut{}, err
		}
		if err := cli.Authenticate(ctx); err != nil {
			return nil, sendOut{}, err
		}
		inv, err := cli.SendPDF(ctx, pdf)
		if err != nil {
			return nil, sendOut{}, err
		}
		_ = o.Journal("pdp_sent", map[string]any{"numero": in.Number, "provider": "superpdp", "reference": inv.ID})
		return nil, sendOut{Provider: "superpdp", Reference: inv.ID, Events: inv.Events}, nil
	})

	type statusIn struct {
		orgParam
		Number string `json:"number" jsonschema:"numéro de la facture"`
	}
	type statusOut struct {
		Provider string           `json:"provider"`
		Events   []superpdp.Event `json:"events"`
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
		ref, err := sentReference(o, in.Number)
		if err != nil {
			return nil, statusOut{}, err
		}
		cli, err := pdpClient(o)
		if err != nil {
			return nil, statusOut{}, err
		}
		if err := cli.Authenticate(ctx); err != nil {
			return nil, statusOut{}, err
		}
		inv, err := cli.GetInvoice(ctx, ref)
		if err != nil {
			return nil, statusOut{}, err
		}
		return nil, statusOut{Provider: "superpdp", Events: inv.Events}, nil
	})
}

// pdpClient construit le client PDP depuis la configuration DE L'ORGANISATION —
// jamais l'environnement global seul : chaque entité a son propre compte.
func pdpClient(o *workspace.Org) (*superpdp.Client, error) {
	cfg := superpdp.Config{
		Base:         o.Lookup("SUPERPDP_BASE"),
		ClientID:     o.Lookup("SUPERPDP_CLIENT_ID"),
		ClientSecret: o.Lookup("SUPERPDP_CLIENT_SECRET"),
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("aucun compte PDP configuré pour cette organisation. L'utilisateur doit " +
			"renseigner SUPERPDP_CLIENT_ID et SUPERPDP_CLIENT_SECRET dans le fichier .env du dossier " +
			"de l'organisation (identifiants fournis par sa plateforme). Ne jamais demander ces " +
			"valeurs en conversation : elles se placent directement dans le fichier")
	}
	return superpdp.New(cfg), nil
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

// sentReference retrouve, dans le journal, la référence PDP d'une facture déjà
// déposée.
func sentReference(o *workspace.Org, number string) (int64, error) {
	f, err := os.Open(filepath.Join(o.Path, workspace.JournalFile))
	if err != nil {
		return 0, fmt.Errorf("aucun dépôt PDP tracé pour cette organisation")
	}
	defer func() { _ = f.Close() }()

	var ref int64 = -1
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var line struct {
			Event string `json:"event"`
			Data  struct {
				Numero    string `json:"numero"`
				Reference int64  `json:"reference"`
			} `json:"data"`
		}
		if json.Unmarshal(sc.Bytes(), &line) != nil {
			continue
		}
		if line.Event == "pdp_sent" && line.Data.Numero == number {
			ref = line.Data.Reference // le plus récent gagne
		}
	}
	if ref < 0 {
		return 0, fmt.Errorf("la facture %s n'a pas été déposée sur la PDP (aucune trace au journal). "+
			"La déposer d'abord avec send_invoice", number)
	}
	return ref, nil
}
