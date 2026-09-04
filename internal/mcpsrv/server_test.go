package mcpsrv

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // registre utilisateur isolé
	org, err := workspace.Init(filepath.Join(t.TempDir(), "orga"), map[string]string{
		"GOFACT_SELLER_NAME":        "Studio Exemple",
		"GOFACT_SELLER_SIRET":       "12345678900014",
		"GOFACT_SELLER_EMAIL":       "contact@exemple.test",
		"GOFACT_SELLER_ADDRESS":     "1 rue de l'Exemple",
		"GOFACT_SELLER_POSTAL_CODE": "33000",
		"GOFACT_SELLER_CITY":        "Bordeaux",
		"GOFACT_PAYEE_IBAN":         "FR7630001007941234567890185",
		"SUPERPDP_CLIENT_SECRET":    "jamais-dans-une-sortie",
	}, "")
	if err != nil {
		t.Fatal(err)
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

	// 2 — pas de modèle : la note d'amorçage guide l'IA.
	_, raw = call(t, cs, "get_invoice_template", orgParam{})
	if !strings.Contains(raw, "{{NUMERO}}") {
		t.Errorf("la note d'amorçage doit exiger le jeton : %s", raw)
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
