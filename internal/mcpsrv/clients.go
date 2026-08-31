package mcpsrv

import (
	"context"

	"github.com/kolapsis/gofact/internal/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Découverte de client. Première source : l'historique du dossier — zéro
// réseau, coordonnées validées par l'usage, adresse de routage Peppol comprise.
// Les sources réseau (annuaire SIRENE, annuaire Peppol) viendront s'agréger ici
// avec les mêmes entrées/sorties.

func addClientTools(s *mcp.Server) {
	type searchIn struct {
		orgParam
		Query string `json:"query" jsonschema:"nom (sous-chaîne), SIREN ou SIRET du client recherché"`
	}
	type searchOut struct {
		Clients []workspace.Client `json:"clients"`
		Note    string             `json:"note,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "search_client",
		Description: "Recherche un client dans l'historique de facturation de l'organisation : coordonnées, " +
			"SIRET, adresse de routage Peppol de la dernière facture. Toujours réutiliser un client " +
			"connu plutôt que de ressaisir ses coordonnées.",
		Annotations: readOnly("Rechercher un client"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in searchIn) (*mcp.CallToolResult, searchOut, error) {
		o, err := resolveOrg(in.Org)
		if err != nil {
			return nil, searchOut{}, err
		}
		found, err := o.FindClient(in.Query)
		if err != nil {
			return nil, searchOut{}, err
		}
		out := searchOut{Clients: found}
		if len(found) == 0 {
			out.Clients = []workspace.Client{}
			out.Note = "Client inconnu de l'historique. Demander ses coordonnées à l'utilisateur " +
				"(nom exact, SIRET, adresse, e-mail) — ne rien inventer. Pour l'envoi à une PDP, " +
				"l'adresse de routage (electronic_address + scheme, généralement le SIREN en 0225) " +
				"est indispensable."
		}
		return nil, out, nil
	})
}
