package annuaire

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Annuaire Peppol public (directory.peppol.eu), indexé par SIREN — le SIRET ne
// renvoie rien. Il sert à deux choses : savoir si un destinataire est
// adressable sur le réseau, et sous quels schémas.
//
// Attention aux référentiels : l'annuaire Peppol identifie les entreprises
// françaises en 0009 (SIRET) ou 0002 (SIREN), tandis que la réforme française
// route en 0225 (SIREN, liste EAS). Ce ne sont pas les mêmes codes — on renvoie
// TOUT ce que l'annuaire connaît et l'appelant choisit selon le contexte
// (dépôt PDP française : préférer 0225, replis 0002).

// PeppolBase est l'URL de l'annuaire ; surchargeable pour les tests.
var PeppolBase = "https://directory.peppol.eu"

// RoutingAddresses cherche un SIREN dans l'annuaire Peppol et renvoie les
// identifiants de participant trouvés. Une liste vide signifie : destinataire
// non adressable sur le réseau — ne pas tenter d'envoi PDP.
func RoutingAddresses(ctx context.Context, siren string) ([]Routing, error) {
	if Offline() {
		return nil, fmt.Errorf("annuaire: recherches réseau désactivées (%s)", EnvOffline)
	}
	siren = strings.ReplaceAll(strings.TrimSpace(siren), " ", "")
	if len(siren) == 14 {
		siren = siren[:9] // l'annuaire est indexé par SIREN, pas par SIRET
	}
	if len(siren) != 9 {
		return nil, fmt.Errorf("annuaire: SIREN %q invalide (9 chiffres attendus)", siren)
	}

	u := fmt.Sprintf("%s/search/1.0/json?q=%s", PeppolBase, url.QueryEscape(siren))
	var resp peppolResponse
	err := cached("peppol:"+siren, &resp, func() ([]byte, error) { return get(ctx, u) })
	if err != nil {
		return nil, fmt.Errorf("annuaire: annuaire Peppol indisponible (%w) — réessayer plus tard, "+
			"ou livrer la facture par un autre canal", err)
	}

	var out []Routing
	for _, m := range resp.Matches {
		scheme, value, ok := strings.Cut(m.ParticipantID.Value, ":")
		if !ok {
			continue
		}
		out = append(out, Routing{Scheme: scheme, Value: value})
	}
	return out, nil
}

type peppolResponse struct {
	TotalResultCount int `json:"total-result-count"`
	Matches          []struct {
		ParticipantID struct {
			Scheme string `json:"scheme"`
			Value  string `json:"value"`
		} `json:"participantID"`
	} `json:"matches"`
}
