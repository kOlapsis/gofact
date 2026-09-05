package mcpsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kolapsis/gofact/internal/facturx"
	"github.com/kolapsis/gofact/internal/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Ces tests parlent au serveur comme un vrai client MCP, via le transport en
// mémoire du SDK : même protocole, mêmes schémas, mêmes annotations que ce que
// verra Claude Desktop ou LM Studio.

func session(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	server := New("test")
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	st, ct := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// testOrg crée une organisation isolée et force sa découverte.
func testOrg(t *testing.T) *workspace.Org {
	t.Helper()
	return testOrgWith(t, nil)
}

// testOrgWith crée une organisation isolée dont le .env reçoit en plus les clés
// de extra — typiquement un compte PDP pointant sur un serveur de test.
func testOrgWith(t *testing.T, extra map[string]string) *workspace.Org {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // registre utilisateur isolé
	env := map[string]string{
		"GOFACT_SELLER_NAME":        "Studio Exemple",
		"GOFACT_SELLER_SIRET":       "12345678900014",
		"GOFACT_SELLER_EMAIL":       "contact@exemple.test",
		"GOFACT_SELLER_ADDRESS":     "1 rue de l'Exemple",
		"GOFACT_SELLER_POSTAL_CODE": "33000",
		"GOFACT_SELLER_CITY":        "Bordeaux",
		"GOFACT_PAYEE_IBAN":         "FR7630001007941234567890185",
		"SUPERPDP_CLIENT_SECRET":    "jamais-dans-une-sortie",
	}
	org, err := workspace.Init(filepath.Join(t.TempDir(), "orga"), env, "")
	if err != nil {
		t.Fatal(err)
	}
	// Init n'écrit que les clés d'identité : le reste (compte PDP…) se pose
	// directement dans le .env du dossier, comme le ferait l'utilisateur.
	if len(extra) > 0 {
		f, err := os.OpenFile(filepath.Join(org.Path, workspace.EnvFile), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range extra {
			fmt.Fprintf(f, "%s=%q\n", k, v)
		}
		_ = f.Close()
		if org, err = workspace.Open(org.Path); err != nil {
			t.Fatal(err)
		}
	}
	// Neutralise toute config parasite du poste de dev.
	for _, k := range []string{"GOFACT_INVOICES_DIR", "GOFACT_ORGS_DIRS"} {
		t.Setenv(k, "")
	}
	return org
}

func call(t *testing.T, cs *mcp.ClientSession, tool string, args any) (*mcp.CallToolResult, string) {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: m})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", tool, err)
	}
	out, _ := json.Marshal(res)
	return res, string(out)
}

func TestToolAnnotations(t *testing.T) {
	cs := session(t)
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*mcp.Tool{}
	for _, tool := range tools.Tools {
		byName[tool.Name] = tool
	}
	for _, name := range []string{"list_organizations", "get_organization", "get_invoice_template",
		"search_client", "find_routing_address", "preview_next_number", "preview_invoice",
		"list_invoices", "create_invoice", "send_invoice", "get_invoice_status",
		"init_organization", "initialize_numbering"} {
		if byName[name] == nil {
			t.Errorf("outil %s absent", name)
		}
	}
	// Seul send_invoice est destructif ; les lectures sont annoncées comme telles.
	for name, tool := range byName {
		ann := tool.Annotations
		if ann == nil {
			t.Errorf("%s sans annotations", name)
			continue
		}
		isDestructive := ann.DestructiveHint != nil && *ann.DestructiveHint
		if name == "send_invoice" && !isDestructive {
			t.Error("send_invoice doit être destructif")
		}
		if name != "send_invoice" && isDestructive {
			t.Errorf("%s ne doit pas être destructif", name)
		}
	}
}

func TestFullInvoiceFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("rendu Chrome requis")
	}
	if !facturx.BrowserAvailable() {
		t.Skip("aucun navigateur de rendu sur ce poste — définissez GOFACT_CHROME pour jouer ce test")
	}
	t.Setenv("GOFACT_OFFLINE", "1") // pas de réseau en test : historique seul
	org := testOrg(t)
	cs := session(t)

	// 1 — l'organisation est découverte, sans fuite de secret.
	_, raw := call(t, cs, "list_organizations", struct{}{})
	if !strings.Contains(raw, "Studio Exemple") {
		t.Fatalf("organisation absente : %s", raw)
	}
	for _, leak := range []string{"jamais-dans-une-sortie", "FR7630001007941234567890185"} {
		if strings.Contains(raw, leak) {
			t.Fatalf("SECRET dans une sortie d'outil : %s", leak)
		}
	}

	// 2 — pas de modèle figé : le modèle par défaut est servi, jamais une page
	// blanche, avec l'identité et les mentions légales déjà en place.
	_, raw = call(t, cs, "get_invoice_template", orgParam{})
	if !strings.Contains(raw, "{{NUMERO}}") {
		t.Errorf("le modèle servi doit porter le jeton : %s", raw)
	}
	if !strings.Contains(raw, `"is_default":true`) {
		t.Errorf("le modèle par défaut doit être annoncé comme tel : %s", raw)
	}
	for _, want := range []string{"Studio Exemple", "293 B du CGI", "L441-10"} {
		if !strings.Contains(raw, want) {
			t.Errorf("modèle par défaut : %q absent de %s", want, raw)
		}
	}

	// 3 — le numéro annoncé n'est pas consommé.
	_, raw = call(t, cs, "preview_next_number", orgParam{})
	if !strings.Contains(raw, "2026001") {
		t.Errorf("preview inattendu : %s", raw)
	}

	// 4 — création : transactionnelle, conforme, modèle figé.
	html := `<!doctype html><meta charset="utf-8"><title>Facture {{NUMERO}} — ACME SAS</title>
<style>.page{font:14px sans-serif;margin:2cm}.total{font-weight:bold}</style>
<div class="page"><h1>Facture {{NUMERO}}</h1><p>Studio Exemple → ACME SAS</p>
<p class="total">Total HT : 1 200,00 € — TVA non applicable, art. 293 B du CGI</p></div>`
	spec := map[string]any{
		"issue_date": "2026-08-31",
		"buyer": map[string]any{
			"name": "ACME SAS", "siret": "55208131700015", "email": "compta@acme.example",
			"address": "1 rue de la Paix", "postal_code": "75002", "city": "Paris",
			"electronic_address": "552081317", "electronic_address_scheme": "0225",
		},
		"lines": []map[string]any{{
			"name": "Développement", "unit": "day", "quantity": "2.00",
			"unit_price_ht_cents": 60000, "amount_ht_cents": 120000,
		}},
	}
	res, raw := call(t, cs, "create_invoice", map[string]any{"html": html, "spec": spec})
	if res.IsError {
		t.Fatalf("create_invoice en erreur : %s", raw)
	}
	var out createOutT
	sc, _ := json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(sc, &out); err != nil {
		t.Fatalf("sortie illisible : %v — %s", err, raw)
	}
	if out.Number != "2026001" || !out.Conform || !out.TemplateFrozen {
		t.Fatalf("création inattendue : %+v", out)
	}
	if _, err := os.Stat(out.PDFPath); err != nil {
		t.Fatalf("PDF absent : %v", err)
	}
	if _, err := os.Stat(filepath.Join(org.Path, workspace.TemplateFile)); err != nil {
		t.Fatalf("modèle non figé : %v", err)
	}
	// Le HTML écrit porte le numéro réel, plus le jeton.
	written, _ := os.ReadFile(out.HTMLPath)
	if strings.Contains(string(written), "{{NUMERO}}") || !strings.Contains(string(written), "2026001") {
		t.Error("substitution du numéro défaillante dans le HTML archivé")
	}

	// 5 — le client est désormais connu de l'historique.
	_, raw = call(t, cs, "search_client", map[string]any{"query": "acme"})
	if !strings.Contains(raw, "55208131700015") || !strings.Contains(raw, "552081317") {
		t.Errorf("client absent de l'historique : %s", raw)
	}

	// 6 — le registre a consommé le numéro.
	_, raw = call(t, cs, "preview_next_number", orgParam{})
	if !strings.Contains(raw, "2026002") {
		t.Errorf("compteur non incrémenté : %s", raw)
	}
}

// Un create_invoice invalide ne doit ni consommer de numéro ni laisser de
// fichier — et l'erreur doit être actionnable par une IA.
func TestCreateInvoiceFailuresAreClean(t *testing.T) {
	org := testOrg(t)
	cs := session(t)

	// HTML sans jeton.
	res, raw := call(t, cs, "create_invoice", map[string]any{
		"html": "<div>pas de jeton</div>",
		"spec": map[string]any{"issue_date": "2026-08-31",
			"buyer": map[string]any{"name": "X"},
			"lines": []map[string]any{{"name": "p", "unit_price_ht_cents": 100}}},
	})
	if !res.IsError || !strings.Contains(raw, "{{NUMERO}}") {
		t.Errorf("erreur attendue nommant le jeton : %s", raw)
	}

	// Spec violant une règle EN 16931 (pas d'acheteur).
	res, raw = call(t, cs, "create_invoice", map[string]any{
		"html": "<div>{{NUMERO}}</div>",
		"spec": map[string]any{"issue_date": "2026-08-31",
			"buyer": map[string]any{"name": ""},
			"lines": []map[string]any{{"name": "p", "unit_price_ht_cents": 100}}},
	})
	if !res.IsError {
		t.Fatalf("erreur attendue : %s", raw)
	}

	// Rien n'a été consommé ni écrit.
	if next, _ := org.NextNumber(time.Now()); next != "2026001" {
		t.Errorf("un échec a consommé un numéro : %s", next)
	}
	entries, _ := os.ReadDir(org.Path)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".pdf") || strings.HasSuffix(e.Name(), ".html") {
			t.Errorf("fichier orphelin après échec : %s", e.Name())
		}
	}
}

func TestSendInvoiceRequiresConfirmation(t *testing.T) {
	testOrg(t)
	cs := session(t)
	res, raw := call(t, cs, "send_invoice", map[string]any{"number": "2026001", "confirm": false})
	if !res.IsError || !strings.Contains(raw, "confirm") {
		t.Errorf("le dépôt sans confirmation doit être refusé : %s", raw)
	}
}

// L'onboarding : aperçu sans consommation, reprise de numérotation,
// remplacement délibéré du modèle.
func TestOnboardingFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("rendu Chrome requis")
	}
	if !facturx.BrowserAvailable() {
		t.Skip("aucun navigateur de rendu sur ce poste — définissez GOFACT_CHROME pour jouer ce test")
	}
	t.Setenv("GOFACT_OFFLINE", "1")
	org := testOrg(t)
	cs := session(t)

	// 1 — reprise d'une numérotation existante.
	_, raw := call(t, cs, "initialize_numbering", map[string]any{"last_invoice_number": "2026011"})
	if !strings.Contains(raw, "2026012") {
		t.Fatalf("reprise attendue à 2026012 : %s", raw)
	}
	// L'abaisser est refusé, avec une erreur explicable.
	res, raw := call(t, cs, "initialize_numbering", map[string]any{"last_invoice_number": "2026005"})
	if !res.IsError || !strings.Contains(raw, "réutiliserait") {
		t.Errorf("abaissement accepté ou erreur muette : %s", raw)
	}

	// 2 — aperçu : rien n'est consommé, le PDF SPÉCIMEN existe.
	html := `<!doctype html><meta charset="utf-8"><title>Modèle</title>
<style>.page{font:14px sans-serif}</style><div class="page"><h1>Facture {{NUMERO}}</h1></div>`
	_, raw = call(t, cs, "preview_invoice", map[string]any{"html": html})
	if !strings.Contains(raw, "apercu.pdf") {
		t.Fatalf("aperçu attendu : %s", raw)
	}
	if _, err := os.Stat(filepath.Join(org.Path, "apercu.pdf")); err != nil {
		t.Fatalf("apercu.pdf absent : %v", err)
	}
	if next, _ := org.NextNumber(time.Now()); next != "2026012" {
		t.Errorf("l'aperçu a consommé un numéro : %s", next)
	}

	// 3 — première facture : le modèle est figé ; puis nouvelle mise en page
	// assumée avec update_template.
	spec := map[string]any{"issue_date": "2026-09-01",
		"buyer": map[string]any{"name": "ACME SAS", "siret": "55208131700015",
			"address": "1 rue de la Paix", "postal_code": "75002", "city": "Paris"},
		"lines": []map[string]any{{"name": "Dev", "unit": "day", "quantity": "1.00",
			"unit_price_ht_cents": 60000}}}
	res, raw = call(t, cs, "create_invoice", map[string]any{"html": html, "spec": spec})
	if res.IsError {
		t.Fatalf("create_invoice : %s", raw)
	}
	if !strings.Contains(raw, "2026012") {
		t.Errorf("numéro attendu 2026012 après reprise : %s", raw)
	}

	html2 := `<!doctype html><meta charset="utf-8"><title>Modèle v2</title>
<style>.sheet{font:13px serif}</style><div class="sheet"><h2>FACTURE {{NUMERO}}</h2></div>`
	res, raw = call(t, cs, "create_invoice", map[string]any{"html": html2, "spec": spec, "update_template": true})
	if res.IsError {
		t.Fatalf("create_invoice v2 : %s", raw)
	}
	tpl, _ := org.Template()
	if !strings.Contains(tpl, "sheet") {
		t.Error("update_template doit remplacer le modèle de référence")
	}
	// Et sans update_template, une mise en page différente n'est qu'un
	// avertissement — jamais un blocage.
	res, raw = call(t, cs, "create_invoice", map[string]any{"html": html, "spec": spec})
	if res.IsError {
		t.Fatalf("une dérive de mise en page ne doit pas bloquer : %s", raw)
	}
	if !strings.Contains(raw, "update_template") {
		t.Errorf("l'avertissement doit orienter vers update_template : %s", raw)
	}
}

// Le modèle par défaut est un plancher de conformité : il doit produire un
// Factur-X valide sans qu'un modèle de langage ait rien à composer.
func TestDefaultTemplateProducesConformInvoice(t *testing.T) {
	if testing.Short() {
		t.Skip("rendu Chrome requis")
	}
	if !facturx.BrowserAvailable() {
		t.Skip("aucun navigateur de rendu sur ce poste — définissez GOFACT_CHROME pour jouer ce test")
	}
	t.Setenv("GOFACT_OFFLINE", "1")
	testOrg(t)
	cs := session(t)

	res, raw := call(t, cs, "get_invoice_template", orgParam{})
	if res.IsError {
		t.Fatalf("get_invoice_template en erreur : %s", raw)
	}
	sc, _ := json.Marshal(res.StructuredContent)
	var tpl struct {
		HTML      string `json:"html"`
		IsDefault bool   `json:"is_default"`
	}
	if err := json.Unmarshal(sc, &tpl); err != nil {
		t.Fatalf("sortie illisible : %v — %s", err, raw)
	}
	if !tpl.IsDefault || tpl.HTML == "" {
		t.Fatalf("modèle par défaut attendu : %s", raw)
	}

	res, raw = call(t, cs, "create_invoice", map[string]any{
		"html": tpl.HTML,
		"spec": map[string]any{
			"issue_date": "2026-08-31",
			"buyer": map[string]any{
				"name": "ACME SAS", "siret": "55208131700015",
				"address": "1 rue de la Paix", "postal_code": "75002", "city": "Paris",
			},
			"lines": []map[string]any{{"name": "Prestation", "unit_price_ht_cents": 100000}},
		},
	})
	if res.IsError {
		t.Fatalf("le modèle par défaut doit produire une facture : %s", raw)
	}
	var out createOutT
	sc, _ = json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(sc, &out); err != nil {
		t.Fatalf("sortie illisible : %v — %s", err, raw)
	}
	if !out.Conform {
		t.Errorf("Factur-X non conforme depuis le modèle par défaut : %+v", out)
	}
}

// registerInvoice inscrit une facture au registre sans passer par Chrome : une
// entrée, son sidecar JSON et un PDF factice — ce qu'il faut pour exercer
// send_invoice et get_invoice_status.
func registerInvoice(t *testing.T, org *workspace.Org, spec facturx.Spec) string {
	t.Helper()
	number, err := org.Allocate(time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC), workspace.RegistryEntry{
		Client: spec.Buyer.Name, Projet: "Test", MontantHT: 60000, Fichier: "FACTURE - Client.html",
	})
	if err != nil {
		t.Fatal(err)
	}
	spec.Number = number
	raw, err := facturx.MarshalSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string][]byte{
		"FACTURE - Client.json": raw,
		"FACTURE - Client.pdf":  []byte("%PDF-1.7 factice"),
	} {
		if err := os.WriteFile(filepath.Join(org.Path, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return number
}

func buyerSpec(buyer facturx.PartySpec) facturx.Spec {
	return facturx.Spec{
		IssueDate: "2026-09-05",
		Buyer:     buyer,
		Lines:     []facturx.LineSpec{{Name: "Prestation", Unit: "day", Quantity: "1.00", UnitPrice: 60000}},
	}
}

// Un acheteur sans adresse de routage connue de l'annuaire n'est pas déposé :
// la PDP le rejetterait comme indélivrable, plus tard et moins clairement.
func TestSendInvoiceRefusesUnroutableBuyer(t *testing.T) {
	org := testOrg(t)
	cs := session(t)
	number := registerInvoice(t, org, buyerSpec(facturx.PartySpec{
		Name: "Particulier", Email: "jean@exemple.test", Address: "2 rue X", PostalCode: "75001", City: "Paris",
	}))
	res, raw := call(t, cs, "send_invoice", map[string]any{"number": number, "confirm": true})
	if !res.IsError {
		t.Fatalf("le dépôt devait être refusé : %s", raw)
	}
	for _, want := range []string{"find_routing_address", "electronic_address", "EM", "FACTURE - Client.json"} {
		if !strings.Contains(raw, want) {
			t.Errorf("le refus doit mentionner %q : %s", want, raw)
		}
	}
	// Rien n'est parti : aucun dépôt tracé.
	if _, err := os.Stat(filepath.Join(org.Path, workspace.JournalFile)); err == nil {
		journal, _ := os.ReadFile(filepath.Join(org.Path, workspace.JournalFile))
		if strings.Contains(string(journal), "pdp_sent") {
			t.Error("un dépôt refusé ne doit pas être tracé comme envoyé")
		}
	}
}

// superPDPMock imite l'API SuperPDP : authentification, dépôt, puis cycle de
// vie — rejeté avec motifs détaillés, ou accepté, selon rejected.
func superPDPMock(t *testing.T, rejected *atomic.Bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"jeton-test"}`))
	})
	mux.HandleFunc("POST /v1.beta/invoices", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer jeton-test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":453372,"company_id":7,"direction":"outbound","events":[
		  {"created_at":"2026-09-05T09:12:01Z","status_code":"api:uploaded","status_text":"Déposée"}]}`))
	})
	mux.HandleFunc("GET /v1.beta/invoices/453372", func(w http.ResponseWriter, r *http.Request) {
		if rejected.Load() {
			_, _ = w.Write([]byte(`{"id":453372,"events":[
			  {"created_at":"2026-09-05T09:12:01Z","status_code":"api:uploaded","status_text":"Déposée"},
			  {"created_at":"2026-09-05T09:12:02Z","status_code":"fr:213","status_text":"Rejetée",
			   "data":{"reason":"Facture non conforme"},
			   "details":[{"reason":"REJ_SEMAN","notes":[
			     {"content_code":"BR-FR-05_BT-22_PMT","subject":"AAO","contents":[{"content":"BR-FR-05/BT-22 : La mention relative aux frais de recouvrement (code PMT) est absente. Elle est obligatoire dans les notes (BG-1)."}]},
			     {"content_code":"BR-FR-05_BT-22_PMD","subject":"AAO","contents":[{"content":"BR-FR-05/BT-22 : La mention relative aux pénalités de retard (code PMD) est absente."}]}]}]},
			  {"created_at":"2026-09-05T09:12:03Z","status_code":"ppf:rejected","status_text":"Rejetée par le PPF"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":453372,"events":[
		  {"created_at":"2026-09-05T09:12:01Z","status_code":"api:uploaded","status_text":"Déposée"},
		  {"created_at":"2026-09-05T09:12:02Z","status_code":"fr:200","status_text":"Déposée"},
		  {"created_at":"2026-09-05T09:12:04Z","status_code":"fr:201","status_text":"Émise"}]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// Un rejet de la plateforme remonte immédiatement, motif par motif — dans la
// réponse de send_invoice, dans get_invoice_status et dans le journal.
func TestSendInvoiceSurfacesRejectionReasons(t *testing.T) {
	var rejected atomic.Bool
	rejected.Store(true)
	srv := superPDPMock(t, &rejected)
	org := testOrgWith(t, map[string]string{
		"SUPERPDP_BASE": srv.URL, "SUPERPDP_CLIENT_ID": "id-test", "SUPERPDP_CLIENT_SECRET": "secret-test",
	})
	cs := session(t)
	prev := statusDelay
	statusDelay = 0
	t.Cleanup(func() { statusDelay = prev })

	number := registerInvoice(t, org, buyerSpec(facturx.PartySpec{
		Name: "ACME SAS", SIRET: "55208131700015", Email: "compta@acme.example",
		Address: "1 rue de la Paix", PostalCode: "75002", City: "Paris",
	}))

	res, raw := call(t, cs, "send_invoice", map[string]any{"number": number, "confirm": true})
	if !res.IsError {
		t.Fatalf("un rejet doit être une erreur d'outil : %s", raw)
	}
	for _, want := range []string{"REJETÉE", "453372", "frais de recouvrement (code PMT)", "pénalités de retard (code PMD)"} {
		if !strings.Contains(raw, want) {
			t.Errorf("le rejet doit porter %q : %s", want, raw)
		}
	}
	journal, _ := os.ReadFile(filepath.Join(org.Path, workspace.JournalFile))
	for _, want := range []string{`"pdp_sent"`, `"pdp_rejected"`, "code PMT"} {
		if !strings.Contains(string(journal), want) {
			t.Errorf("journal sans %q :\n%s", want, journal)
		}
	}

	// Le statut relit les mêmes motifs, structurés.
	res, raw = call(t, cs, "get_invoice_status", map[string]any{"number": number})
	if res.IsError {
		t.Fatalf("get_invoice_status : %s", raw)
	}
	if !strings.Contains(raw, `"rejected":true`) || !strings.Contains(raw, "code PMD") || !strings.Contains(raw, `"reasons"`) {
		t.Errorf("le statut doit exposer le rejet et ses motifs : %s", raw)
	}

	// Facture acceptée : le dépôt réussit, sans rejet, cycle de vie à jour.
	rejected.Store(false)
	number2 := registerInvoice(t, org, buyerSpec(facturx.PartySpec{
		Name: "ACME SAS", SIRET: "55208131700015", EAddr: "552081317", EAddrSchema: "0002",
		Address: "1 rue de la Paix", PostalCode: "75002", City: "Paris",
	}))
	res, raw = call(t, cs, "send_invoice", map[string]any{"number": number2, "confirm": true})
	if res.IsError {
		t.Fatalf("dépôt accepté attendu : %s", raw)
	}
	if !strings.Contains(raw, `"rejected":false`) || !strings.Contains(raw, "fr:201") {
		t.Errorf("le dépôt doit renvoyer le cycle de vie relu : %s", raw)
	}
}

// create_invoice inscrit dans le sidecar l'adresse de routage réellement
// utilisée dans le XML — dérivée du SIREN quand rien n'est déclaré — et au
// registre l'objet de la facture plutôt que le libellé de la première ligne.
func TestCreateInvoiceWritesRoutingAndTitle(t *testing.T) {
	if testing.Short() {
		t.Skip("rendu Chrome requis")
	}
	if !facturx.BrowserAvailable() {
		t.Skip("aucun navigateur de rendu sur ce poste — définissez GOFACT_CHROME pour jouer ce test")
	}
	t.Setenv("GOFACT_OFFLINE", "1")
	org := testOrg(t)
	cs := session(t)

	html := `<!doctype html><meta charset="utf-8"><title>Facture {{NUMERO}}</title>
<style>.page{font:14px sans-serif}</style><div class="page"><h1>Facture {{NUMERO}}</h1></div>`
	res, raw := call(t, cs, "create_invoice", map[string]any{"html": html, "spec": map[string]any{
		"issue_date": "2026-09-05",
		"title":      "Migration du serveur de fichiers",
		"buyer": map[string]any{
			"name": "ACME SAS", "siret": "55208131700015", "email": "compta@acme.example",
			"address": "1 rue de la Paix", "postal_code": "75002", "city": "Paris",
		},
		"lines": []map[string]any{{"name": "Préparation du disque additionnel", "unit": "day",
			"quantity": "1.00", "unit_price_ht_cents": 60000}},
		"notes": []map[string]any{{"content": "Intervention à distance."}},
	}})
	if res.IsError {
		t.Fatalf("create_invoice : %s", raw)
	}
	var out createOutT
	sc, _ := json.Marshal(res.StructuredContent)
	_ = json.Unmarshal(sc, &out)

	spec, err := facturx.LoadSpec(strings.TrimSuffix(out.HTMLPath, ".html") + ".json")
	if err != nil {
		t.Fatalf("sidecar : %v", err)
	}
	if spec.Buyer.EAddr != "552081317" || spec.Buyer.EAddrSchema != "0225" {
		t.Errorf("sidecar sans l'adresse de routage résolue : %q/%q", spec.Buyer.EAddr, spec.Buyer.EAddrSchema)
	}
	if len(spec.Notes) != 1 || spec.Notes[0].Content != "Intervention à distance." {
		t.Errorf("le sidecar garde les notes du spec telles quelles : %+v", spec.Notes)
	}
	invoices, _ := org.Invoices()
	if len(invoices) != 1 || str(invoices[0]["projet"]) != "Migration du serveur de fichiers" {
		t.Errorf("le registre doit porter l'objet de la facture : %v", invoices)
	}
}

func TestInvoiceTitleFallsBackToFirstLine(t *testing.T) {
	spec := facturx.Spec{Lines: []facturx.LineSpec{{Name: " Ligne une "}}}
	if got := invoiceTitle(spec); got != "Ligne une" {
		t.Errorf("invoiceTitle sans title = %q", got)
	}
	spec.Title = "Objet"
	if got := invoiceTitle(spec); got != "Objet" {
		t.Errorf("invoiceTitle avec title = %q", got)
	}
	if got := invoiceTitle(facturx.Spec{}); got != "" {
		t.Errorf("invoiceTitle vide = %q", got)
	}
}

// Le schéma de create_invoice documente ce que le modèle doit savoir pour ne
// pas reproduire le rejet de 2026015 : notes = compléments, title = objet.
func TestCreateInvoiceSchemaDocumentsNotesAndTitle(t *testing.T) {
	cs := session(t)
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name != "create_invoice" {
			continue
		}
		schema, _ := json.Marshal(tool.InputSchema)
		for _, want := range []string{"PMD/PMT/AAB", "AAI", "objet de la facture", "0225"} {
			if !strings.Contains(string(schema), want) {
				t.Errorf("le schéma de create_invoice doit mentionner %q", want)
			}
		}
		return
	}
	t.Fatal("create_invoice absent")
}
