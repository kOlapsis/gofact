package mcpsrv

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Le prompt « nouvelle-facture » encode le déroulé complet côté client MCP :
// c'est la version portable du skill Claude Code, utilisable depuis n'importe
// quel client qui expose les prompts.

const newInvoiceFlow = `Tu aides un utilisateur à émettre une facture électronique française (Factur-X)
avec les outils gofact. Déroulé :

1. list_organizations — identifier l'entité émettrice. S'il y en a plusieurs,
   demander laquelle. S'il n'y en a aucune, proposer init_organization et
   collecter l'identité auprès de l'utilisateur (ne rien inventer).
2. search_client — retrouver le client dans l'historique. S'il est inconnu,
   demander ses coordonnées : nom exact, SIRET, adresse postale, e-mail, et si
   possible son adresse de routage Peppol (souvent le SIREN, scheme 0225).
3. Collecter les prestations : libellé, quantité (jours ou unités), prix
   unitaire HT. Tous les montants se transmettent en CENTIMES dans spec.
4. get_invoice_template — s'il existe un modèle figé, REPARTIR de ce modèle et
   n'adapter que les contenus. Sinon composer un HTML de facture soigné :
   format A4, CSS embarqué, polices système, pas de liens <a href>, logo
   vectoriel — et le jeton {{NUMERO}} à l'emplacement du numéro.
5. preview_next_number — annoncer le numéro, récapituler la facture (client,
   lignes, totaux HT/TTC, date) et demander confirmation à l'utilisateur.
6. create_invoice — après accord seulement. Relayer le chemin du PDF produit
   et tout avertissement retourné.
7. Si l'utilisateur veut l'envoyer à sa plateforme (PDP) : demander une
   confirmation explicite, puis send_invoice avec confirm=true, et suivre le
   cycle de vie avec get_invoice_status.

Règles : jamais de numéro inventé ni réutilisé ; jamais de montant recalculé à
la main quand l'outil le renvoie ; toute erreur d'outil se relaie en termes
simples avec l'action corrective qu'elle indique.`

func addPrompts(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "nouvelle-facture",
		Title:       "Créer une facture",
		Description: "Déroulé complet pour émettre une facture Factur-X avec gofact.",
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "Créer une facture avec gofact",
			Messages: []*mcp.PromptMessage{{
				Role:    "user",
				Content: &mcp.TextContent{Text: newInvoiceFlow},
			}},
		}, nil
	})
}
