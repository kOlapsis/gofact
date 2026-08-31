package mcpsrv

import (
	"context"

	"github.com/kolapsis/gofact/internal/annuaire"
	"github.com/kolapsis/gofact/internal/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Découverte de client, par ordre de confiance :
//
//	1. l'HISTORIQUE du dossier — zéro réseau, coordonnées validées par l'usage,
//	   adresse de routage Peppol comprise ;
//	2. l'annuaire SIRENE — résout un nom en identité légale, sur invocation
//	   explicite seulement ;
//	3. l'annuaire Peppol — l'adressabilité électronique, via l'outil dédié.
//
// Seule la chaîne recherchée part sur le réseau ; GOFACT_OFFLINE=1 coupe tout.

func addClientTools(s *mcp.Server) {
	type searchIn struct {
		orgParam
		Query string `json:"query" jsonschema:"nom (sous-chaîne), SIREN ou SIRET du client recherché"`
	}
	type searchOut struct {
		Known     []workspace.Client   `json:"known_clients"`
		Directory []annuaire.Candidate `json:"directory,omitempty"`
		Note      string               `json:"note,omitempty"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "search_client",
		Description: "Recherche un client : d'abord l'historique de facturation de l'organisation " +
			"(coordonnées validées par l'usage — TOUJOURS les réutiliser), puis, si l'historique ne " +
			"connaît pas le nom, l'annuaire public des entreprises (SIRENE) pour résoudre nom, SIREN, " +
			"SIRET et adresse du siège. Ne retenir qu'un candidat actif, confirmé par l'utilisateur.",
		Annotations: &mcp.ToolAnnotations{Title: "Rechercher un client", ReadOnlyHint: true,
			DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(true)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in searchIn) (*mcp.CallToolResult, searchOut, error) {
		o, err := resolveOrg(in.Org)
		if err != nil {
			return nil, searchOut{}, err
		}
		known, err := o.FindClient(in.Query)
		if err != nil {
			return nil, searchOut{}, err
		}
		out := searchOut{Known: known}
		if out.Known == nil {
			out.Known = []workspace.Client{}
		}
		if len(known) > 0 {
			out.Note = "Client(s) déjà facturé(s) : réutiliser ces coordonnées telles quelles, " +
				"adresse de routage comprise."
			return nil, out, nil
		}

		// Inconnu de l'historique : l'annuaire public prend le relais.
		found, err := annuaire.SearchCompanies(ctx, in.Query, 5)
		if err != nil {
			out.Note = "Client inconnu de l'historique, et l'annuaire des entreprises n'a pas pu être " +
				"consulté (" + err.Error() + "). Demander les coordonnées à l'utilisateur — ne rien inventer."
			return nil, out, nil
		}
		out.Directory = found
		if len(found) == 0 {
			out.Note = "Aucun résultat, ni dans l'historique ni dans l'annuaire des entreprises. " +
				"Vérifier l'orthographe avec l'utilisateur, ou lui demander le SIREN."
		} else {
			out.Note = "Candidats de l'annuaire SIRENE : faire CONFIRMER le bon par l'utilisateur avant " +
				"de facturer, et compléter avec find_routing_address pour l'envoi PDP."
		}
		return nil, out, nil
	})

	type routingIn struct {
		SIREN string `json:"siren" jsonschema:"SIREN (9 chiffres) ou SIRET (14 chiffres, ramené au SIREN) du destinataire"`
	}
	type routingOut struct {
		Reachable bool               `json:"reachable"`
		Routes    []annuaire.Routing `json:"routes"`
		Note      string             `json:"note"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "find_routing_address",
		Description: "Vérifie dans l'annuaire Peppol qu'un destinataire est adressable électroniquement, " +
			"et sous quels identifiants. Indispensable avant un envoi PDP : sans adresse de routage " +
			"valide, le dépôt est rejeté.",
		Annotations: &mcp.ToolAnnotations{Title: "Adresse de routage", ReadOnlyHint: true,
			DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(true)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in routingIn) (*mcp.CallToolResult, routingOut, error) {
		routes, err := annuaire.RoutingAddresses(ctx, in.SIREN)
		if err != nil {
			return nil, routingOut{}, err
		}
		out := routingOut{Reachable: len(routes) > 0, Routes: routes}
		if out.Routes == nil {
			out.Routes = []annuaire.Routing{}
		}
		if !out.Reachable {
			out.Note = "Destinataire absent de l'annuaire : NE PAS tenter l'envoi PDP — livrer le PDF " +
				"Factur-X par un autre canal (e-mail) et le signaler à l'utilisateur."
		} else {
			out.Note = "Pour le sidecar : electronic_address = la valeur, electronic_address_scheme = le " +
				"scheme. Pour une PDP française, préférer 0225 ; si le dépôt est rejeté, réessayer en 0002."
		}
		return nil, out, nil
	})
}
