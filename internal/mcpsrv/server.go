// Package mcpsrv expose gofact en serveur MCP local (stdio) : l'IA de
// l'utilisateur — Claude, ChatGPT, LM Studio… — devient l'interface de
// facturation, et gofact le moteur qui garantit la conformité, la numérotation
// et l'archivage, en local.
//
// Règles de conception :
//
//   - stdout appartient au protocole JSON-RPC : rien d'autre ne s'y écrit ;
//   - aucune valeur secrète ne sort jamais dans une réponse d'outil ;
//   - les erreurs sont rédigées pour être RELAYÉES par l'IA à un utilisateur
//     non technicien : elles nomment le problème et l'action corrective ;
//   - seuls les outils qui écrivent le disent (readOnlyHint), et seul l'envoi
//     PDP — irréversible — est marqué destructif et exige une confirmation.
package mcpsrv

import (
	"fmt"
	"strings"

	"github.com/kolapsis/gofact/internal/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// New construit le serveur MCP avec toute la surface d'outils.
func New(version string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "gofact",
		Version: version,
		Title:   "gofact — facturation électronique Factur-X",
	}, nil)

	addOrgTools(s)
	addInvoiceTools(s)
	addClientTools(s)
	addPDPTools(s)
	addPrompts(s)
	return s
}

// boolPtr aide à renseigner les annotations à pointeur du SDK.
func boolPtr(b bool) *bool { return &b }

// readOnly est l'annotation des outils qui ne modifient rien.
func readOnly(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, DestructiveHint: boolPtr(false)}
}

// writes est l'annotation des outils qui écrivent localement, sans être
// destructifs ni irréversibles.
func writes(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: false, DestructiveHint: boolPtr(false)}
}

// resolveOrg localise l'organisation visée par un appel d'outil. path vide :
// s'il n'existe qu'une organisation, elle est choisie ; sinon on demande à
// l'IA de préciser — jamais de choix silencieux entre plusieurs entités
// émettrices.
func resolveOrg(path string) (*workspace.Org, error) {
	orgs, err := workspace.Discover(path)
	if err != nil {
		return nil, err
	}
	switch len(orgs) {
	case 0:
		return nil, fmt.Errorf("aucune organisation trouvée. Créez-en une avec l'outil init_organization, " +
			"ou indiquez son dossier via le paramètre org (ou la variable GOFACT_INVOICES_DIR)")
	case 1:
		return orgs[0], nil
	default:
		var names []string
		for _, o := range orgs {
			names = append(names, fmt.Sprintf("%s (%s)", o.Name(), o.Path))
		}
		return nil, fmt.Errorf("plusieurs organisations existent : %s. Précisez laquelle avec le paramètre org "+
			"— demandez à l'utilisateur si le contexte ne suffit pas", strings.Join(names, " ; "))
	}
}
