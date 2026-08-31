package annuaire

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
)

// Les tests rejouent des réponses enregistrées : aucun appel réseau réel en CI.

const sireneFixture = `{"results":[{
  "siren":"552081317","nom_complet":"ACME SAS","nom_raison_sociale":"ACME",
  "etat_administratif":"A",
  "siege":{"siret":"55208131700015","adresse":"1 RUE DE LA PAIX 75002 PARIS",
           "code_postal":"75002","libelle_commune":"PARIS"}}]}`

const peppolFixture = `{"total-result-count":2,"matches":[
  {"participantID":{"scheme":"iso6523-actorid-upis","value":"0225:552081317"}},
  {"participantID":{"scheme":"iso6523-actorid-upis","value":"0002:552081317"}}]}`

func TestMain(m *testing.M) {
	// Cache isolé : ni lecture ni pollution du cache réel de l'utilisateur.
	tmp, _ := os.MkdirTemp("", "gofact-annuaire-test-")
	os.Setenv("XDG_CACHE_HOME", tmp)
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func TestSearchCompanies(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if q := r.URL.Query().Get("q"); q != "acme" {
			t.Errorf("q = %q", q)
		}
		_, _ = w.Write([]byte(sireneFixture))
	}))
	defer srv.Close()
	old := SireneBase
	SireneBase = srv.URL
	defer func() { SireneBase = old }()

	got, err := SearchCompanies(context.Background(), "acme", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("candidats = %d, want 1", len(got))
	}
	c := got[0]
	if c.Name != "ACME SAS" || c.SIREN != "552081317" || c.SIRET != "55208131700015" ||
		c.City != "PARIS" || !c.Active || c.Source != "sirene" {
		t.Errorf("candidat inattendu : %+v", c)
	}

	// Deuxième appel : servi par le cache disque, zéro requête réseau.
	if _, err := SearchCompanies(context.Background(), "acme", 5); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Errorf("le cache n'a pas servi : %d appels réseau", calls.Load())
	}
}

func TestRoutingAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("q"); q != "552081317" {
			t.Errorf("q = %q — l'annuaire s'interroge par SIREN, pas par SIRET", q)
		}
		_, _ = w.Write([]byte(peppolFixture))
	}))
	defer srv.Close()
	old := PeppolBase
	PeppolBase = srv.URL
	defer func() { PeppolBase = old }()

	// Un SIRET en entrée est ramené au SIREN.
	got, err := RoutingAddresses(context.Background(), "55208131700015")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Scheme != "0225" || got[1].Scheme != "0002" ||
		got[0].Value != "552081317" {
		t.Errorf("routages inattendus : %+v", got)
	}
}

func TestOfflineCutsNetwork(t *testing.T) {
	t.Setenv(EnvOffline, "1")
	if _, err := SearchCompanies(context.Background(), "acme", 5); err == nil {
		t.Error("SIRENE doit être coupée hors ligne")
	}
	if _, err := RoutingAddresses(context.Background(), "552081317"); err == nil {
		t.Error("Peppol doit être coupé hors ligne")
	}
}
