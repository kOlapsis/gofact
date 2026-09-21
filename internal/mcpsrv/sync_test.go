package mcpsrv

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kolapsis/gofact/internal/facturx"
	"github.com/kolapsis/gofact/internal/workspace"
)

type syncMock struct {
	mu     sync.Mutex
	events []map[string]any
	afters []string
}

// superPDPSyncMock imite SuperPDP pour un dépôt accepté, les événements de cycle de vie et deux pages de factures reçues.
func superPDPSyncMock(t *testing.T) (*httptest.Server, *syncMock) {
	t.Helper()
	m := &syncMock{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"jeton-test"}`))
	})
	mux.HandleFunc("POST /v1.beta/invoices", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":453372,"events":[{"status_code":"api:uploaded","status_text":"Déposée"}]}`))
	})
	mux.HandleFunc("GET /v1.beta/invoices/453372", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":453372,"events":[{"status_code":"fr:201","status_text":"Émise"}]}`))
	})
	mux.HandleFunc("POST /v1.beta/invoice_events", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		m.mu.Lock()
		m.events = append(m.events, body)
		m.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	})
	mux.HandleFunc("GET /v1.beta/invoices", func(w http.ResponseWriter, r *http.Request) {
		after := r.URL.Query().Get("starting_after_id")
		m.mu.Lock()
		m.afters = append(m.afters, after)
		m.mu.Unlock()
		if r.URL.Query().Get("direction") != "in" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch after {
		case "0":
			_, _ = w.Write([]byte(`{"count":1,"has_after":true,"has_before":false,"data":[
			  {"id":9001,"direction":"in","created_at":"2026-09-10T08:00:00Z","en_invoice":{
			    "number":"F-77","issue_date":"2026-09-09","currency_code":"EUR",
			    "seller":{"name":"Fournisseur A","legal_registration_identifier":{"scheme":"0002","value":"111111111"}},
			    "totals":{"total_without_vat":"100.00","total_vat_amount":{"value":"20.00","currency_code":"EUR"},"total_with_vat":"120.00"}}}]}`))
		case "9001":
			_, _ = w.Write([]byte(`{"count":1,"has_after":false,"has_before":true,"data":[
			  {"id":9002,"direction":"in","created_at":"2026-09-11T08:00:00Z","en_invoice":{
			    "number":"B/12","issue_date":"2026-08-30",
			    "seller":{"name":"Fournisseur: B"},
			    "totals":{"total_without_vat":"50.00","total_vat_amount":"10.00","total_with_vat":"60.00"}}}]}`))
		default:
			_, _ = w.Write([]byte(`{"count":0,"has_after":false,"has_before":true,"data":[]}`))
		}
	})
	mux.HandleFunc("GET /v1.beta/invoices/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "factur-x" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.7 reçue " + r.PathValue("id")))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, m
}

func TestReceivedPaymentAndExport(t *testing.T) {
	srv, mock := superPDPSyncMock(t)
	org := testOrgWith(t, map[string]string{
		"SUPERPDP_BASE": srv.URL, "SUPERPDP_CLIENT_ID": "id-test", "SUPERPDP_CLIENT_SECRET": "secret-test",
	})
	cs := session(t)
	prev := statusDelay
	statusDelay = 0
	t.Cleanup(func() { statusDelay = prev })

	number := registerInvoice(t, org, buyerSpec(facturx.PartySpec{
		Name: "ACME SAS", SIRET: "55208131700015", EAddr: "552081317", EAddrSchema: "0002",
		Address: "1 rue de la Paix", PostalCode: "75002", City: "Paris",
	}))

	// Encaissement d'une facture jamais déposée : refusé, sans appel à la plateforme.
	res, raw := call(t, cs, "report_payment", map[string]any{"number": number, "confirm": true})
	if !res.IsError || !strings.Contains(raw, "send_invoice") {
		t.Fatalf("une facture non déposée ne s'encaisse pas : %s", raw)
	}
	if res, raw = call(t, cs, "send_invoice", map[string]any{"number": number, "confirm": true}); res.IsError {
		t.Fatalf("send_invoice : %s", raw)
	}
	if res, raw = call(t, cs, "report_payment", map[string]any{"number": number}); !res.IsError {
		t.Fatalf("report_payment sans confirm doit être refusé : %s", raw)
	}
	if res, raw = call(t, cs, "report_payment", map[string]any{"number": number, "confirm": true}); res.IsError {
		t.Fatalf("report_payment total : %s", raw)
	}
	if res, raw = call(t, cs, "report_payment", map[string]any{"number": number, "confirm": true,
		"date": "2026-09-12", "amount": "300,00"}); res.IsError {
		t.Fatalf("report_payment partiel : %s", raw)
	}
	if len(mock.events) != 2 {
		t.Fatalf("2 événements attendus, %d reçus", len(mock.events))
	}
	if mock.events[0]["status_code"] != "fr:212" || mock.events[0]["invoice_id"] != float64(453372) || mock.events[0]["details"] != nil {
		t.Errorf("encaissement total mal formé : %v", mock.events[0])
	}
	partial, _ := json.Marshal(mock.events[1]["details"])
	for _, want := range []string{`"type_code":"MEN"`, `"amount":"300.00"`, `"date":"2026-09-12"`} {
		if !strings.Contains(string(partial), want) {
			t.Errorf("encaissement partiel sans %s : %s", want, partial)
		}
	}
	// Une date sans montant : le montant transmis est le total de la facture.
	if res, raw = call(t, cs, "report_payment", map[string]any{"number": number, "confirm": true,
		"date": "2026-09-14"}); res.IsError {
		t.Fatalf("report_payment daté : %s", raw)
	}
	dated, _ := json.Marshal(mock.events[2]["details"])
	if !strings.Contains(string(dated), `"date":"2026-09-14"`) || !strings.Contains(string(dated), `"amount":"`) {
		t.Errorf("encaissement daté sans montant du total : %s", dated)
	}
	if _, raw = call(t, cs, "list_invoices", map[string]any{}); !strings.Contains(raw, `"encaissee_le":"2026-09-14"`) {
		t.Errorf("list_invoices doit montrer l'encaissement : %s", raw)
	}

	// Réception : deux pages, puis plus rien au passage suivant.
	res, raw = call(t, cs, "list_received_invoices", map[string]any{"refresh": true})
	if res.IsError || !strings.Contains(raw, `"new":2`) {
		t.Fatalf("2 nouvelles factures attendues : %s", raw)
	}
	res, raw = call(t, cs, "list_received_invoices", map[string]any{"refresh": true})
	if res.IsError || !strings.Contains(raw, `"new":0`) {
		t.Fatalf("un second passage ne doit rien rapatrier : %s", raw)
	}
	if got := strings.Join(mock.afters, ","); got != "0,9001,9002" {
		t.Errorf("curseurs interrogés = %s", got)
	}
	pdf := filepath.Join(org.Path, workspace.ReceivedDir, "2026-08", "2026-08-30 - Fournisseur_ B - B_12.pdf")
	if data, err := os.ReadFile(pdf); err != nil || string(data) != "%PDF-1.7 reçue 9002" {
		t.Errorf("PDF reçu absent ou faux (%v) : %q", err, data)
	}

	// Export du mois : factures émises et reçues, récapitulatif CSV.
	dest := t.TempDir()
	res, raw = call(t, cs, "export_invoices", map[string]any{"month": "2026-09", "to": dest})
	if res.IsError {
		t.Fatalf("export_invoices : %s", raw)
	}
	dir := filepath.Join(dest, "Studio Exemple", "2026-09")
	csv, err := os.ReadFile(filepath.Join(dir, "recapitulatif.csv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"emise;" + number + ";2026-09-05;ACME SAS;600,00;", ";2026-09-14;emises/FACTURE - Client.pdf",
		"recue;F-77;2026-09-09;Fournisseur A;100,00;20,00;120,00;EUR;"} {
		if !strings.Contains(string(csv), want) {
			t.Errorf("CSV sans %q :\n%s", want, csv)
		}
	}
	if strings.Contains(string(csv), "B/12") {
		t.Error("une facture d'août ne doit pas figurer dans l'export de septembre")
	}
	for _, f := range []string{"emises/FACTURE - Client.pdf", "recues/2026-09-09 - Fournisseur A - F-77.pdf"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("export sans %s", f)
		}
	}
	if _, raw = call(t, cs, "export_invoices", map[string]any{"month": "2026-09", "to": dest}); !strings.Contains(raw, `"copied":0`) {
		t.Errorf("un second export ne recopie rien : %s", raw)
	}
}
