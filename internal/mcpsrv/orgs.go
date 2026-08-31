package mcpsrv

import (
	"context"
	"time"

	"github.com/kolapsis/gofact/internal/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Outils de gestion des organisations (entités émettrices).

type orgParam struct {
	Org string `json:"org,omitempty" jsonschema:"chemin du dossier de l'organisation ; omis si une seule organisation existe"`
}

type orgSummary struct {
	workspace.Identity
	Invoices    int    `json:"invoices"`
	NextNumber  string `json:"next_number"`
	HasTemplate bool   `json:"has_template"`
}

func summarize(o *workspace.Org) orgSummary {
	count, _ := o.InvoiceCount()
	next, _ := o.NextNumber(time.Now())
	tpl, _ := o.Template()
	return orgSummary{
		Identity:    o.Identity(),
		Invoices:    count,
		NextNumber:  next,
		HasTemplate: tpl != "",
	}
}

func addOrgTools(s *mcp.Server) {
	type listOut struct {
		Organizations []orgSummary `json:"organizations"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_organizations",
		Description: "Liste les organisations (entités émettrices de factures) configurées sur ce poste, " +
			"avec leur identité publique, leur compteur et leur prochain numéro. À appeler en premier.",
		Annotations: readOnly("Lister les organisations"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listOut, error) {
		orgs, err := workspace.Discover("")
		if err != nil {
			return nil, listOut{}, err
		}
		out := listOut{Organizations: []orgSummary{}}
		for _, o := range orgs {
			out.Organizations = append(out.Organizations, summarize(o))
		}
		return nil, out, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "get_organization",
		Description: "Fiche d'une organisation : identité de l'émetteur (jamais de secret ni d'IBAN en clair), " +
			"état de la numérotation, présence du modèle de facture figé.",
		Annotations: readOnly("Consulter une organisation"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in orgParam) (*mcp.CallToolResult, orgSummary, error) {
		o, err := resolveOrg(in.Org)
		if err != nil {
			return nil, orgSummary{}, err
		}
		return nil, summarize(o), nil
	})

	type initIn struct {
		Path       string `json:"path" jsonschema:"dossier à créer pour l'organisation"`
		Name       string `json:"name" jsonschema:"nom de l'entité émettrice"`
		SIRET      string `json:"siret,omitempty" jsonschema:"SIRET (14 chiffres)"`
		SIREN      string `json:"siren,omitempty" jsonschema:"SIREN (9 chiffres), si pas de SIRET"`
		VATNumber  string `json:"vat_number,omitempty" jsonschema:"n° TVA intracommunautaire ; vide en franchise 293 B"`
		Email      string `json:"email,omitempty"`
		Address    string `json:"address,omitempty" jsonschema:"adresse (rue)"`
		PostalCode string `json:"postal_code,omitempty"`
		City       string `json:"city,omitempty"`
		IBAN       string `json:"iban,omitempty" jsonschema:"IBAN de règlement — requis pour émettre des factures payables par virement"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "init_organization",
		Description: "Crée un dossier d'organisation : registre de numérotation vierge et identité de " +
			"l'émetteur. Refuse d'écraser un dossier existant. Demander les informations à l'utilisateur, " +
			"ne jamais les inventer.",
		Annotations: &mcp.ToolAnnotations{Title: "Créer une organisation", ReadOnlyHint: false,
			DestructiveHint: boolPtr(false), IdempotentHint: true},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in initIn) (*mcp.CallToolResult, orgSummary, error) {
		o, err := workspace.Init(in.Path, map[string]string{
			"GOFACT_SELLER_NAME":        in.Name,
			"GOFACT_SELLER_SIRET":       in.SIRET,
			"GOFACT_SELLER_SIREN":       in.SIREN,
			"GOFACT_SELLER_VAT_NUMBER":  in.VATNumber,
			"GOFACT_SELLER_EMAIL":       in.Email,
			"GOFACT_SELLER_ADDRESS":     in.Address,
			"GOFACT_SELLER_POSTAL_CODE": in.PostalCode,
			"GOFACT_SELLER_CITY":        in.City,
			"GOFACT_PAYEE_IBAN":         in.IBAN,
		})
		if err != nil {
			return nil, orgSummary{}, err
		}
		return nil, summarize(o), nil
	})

	type templateOut struct {
		HTML string `json:"html,omitempty"`
		Note string `json:"note"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_invoice_template",
		Description: "Modèle HTML de référence de l'organisation, figé lors de la première facture. " +
			"S'il existe, REPARTIR DE CE MODÈLE pour toute nouvelle facture (mêmes classes, même " +
			"structure, même feuille de style) : les factures d'une même entité doivent se ressembler.",
		Annotations: readOnly("Modèle de facture"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in orgParam) (*mcp.CallToolResult, templateOut, error) {
		o, err := resolveOrg(in.Org)
		if err != nil {
			return nil, templateOut{}, err
		}
		tpl, err := o.Template()
		if err != nil {
			return nil, templateOut{}, err
		}
		if tpl == "" {
			return nil, templateOut{Note: "Aucun modèle figé : c'est la première facture de cette organisation. " +
				"Composer un HTML de facture soigné et imprimable (A4, CSS embarqué, polices système, " +
				"pas de <a href>, logo vectoriel) avec le jeton {{NUMERO}} à la place du numéro ; " +
				"il deviendra le modèle de référence."}, nil
		}
		return nil, templateOut{HTML: tpl, Note: "Modèle de référence : reprendre sa structure telle quelle, " +
			"n'adapter que les contenus (client, lignes, montants, dates) et garder {{NUMERO}}."}, nil
	})
}
