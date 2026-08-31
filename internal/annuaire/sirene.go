package annuaire

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// API Recherche d'entreprises (annuaire-entreprises.data.gouv.fr) : publique,
// sans clé, ~7 requêtes/s. Résout un nom, un SIREN ou un SIRET en identité
// légale — nom exact, SIREN/SIRET du siège, adresse, état administratif.

// SireneBase est l'URL de l'API ; surchargeable pour les tests.
var SireneBase = "https://recherche-entreprises.api.gouv.fr"

// SearchCompanies interroge l'annuaire des entreprises. Renvoie au plus limit
// candidats. Erreur explicite si les sources réseau sont coupées.
func SearchCompanies(ctx context.Context, query string, limit int) ([]Candidate, error) {
	if Offline() {
		return nil, fmt.Errorf("annuaire: recherches réseau désactivées (%s) — utiliser l'historique local", EnvOffline)
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("annuaire: recherche vide")
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}

	u := fmt.Sprintf("%s/search?q=%s&page=1&per_page=%d", SireneBase, url.QueryEscape(query), limit)
	var resp sireneResponse
	err := cached("sirene:"+query, &resp, func() ([]byte, error) { return get(ctx, u) })
	if err != nil {
		return nil, fmt.Errorf("annuaire: recherche d'entreprises indisponible (%w) — demander les "+
			"coordonnées à l'utilisateur, ou réessayer plus tard", err)
	}

	out := make([]Candidate, 0, len(resp.Results))
	for _, r := range resp.Results {
		name := r.NomComplet
		if name == "" {
			name = r.NomRaisonSociale
		}
		out = append(out, Candidate{
			Source:     "sirene",
			Name:       name,
			SIREN:      r.SIREN,
			SIRET:      r.Siege.SIRET,
			Address:    r.Siege.Adresse,
			PostalCode: r.Siege.CodePostal,
			City:       r.Siege.LibelleCommune,
			Active:     r.EtatAdministratif == "A",
		})
	}
	return out, nil
}

type sireneResponse struct {
	Results []struct {
		SIREN             string `json:"siren"`
		NomComplet        string `json:"nom_complet"`
		NomRaisonSociale  string `json:"nom_raison_sociale"`
		EtatAdministratif string `json:"etat_administratif"`
		Siege             struct {
			SIRET          string `json:"siret"`
			Adresse        string `json:"adresse"`
			CodePostal     string `json:"code_postal"`
			LibelleCommune string `json:"libelle_commune"`
		} `json:"siege"`
	} `json:"results"`
}

func get(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("statut HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}
