package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newOrg(t *testing.T) *Org {
	t.Helper()
	org, err := Init(filepath.Join(t.TempDir(), "orga"), map[string]string{
		"GOFACT_SELLER_NAME":  "Studio Exemple",
		"GOFACT_SELLER_SIRET": "12345678900014",
	}, "")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return org
}

func TestMain(m *testing.M) {
	// Init inscrit chaque dossier dans le registre utilisateur : on isole le
	// XDG_CONFIG_HOME pour ne pas polluer la configuration réelle.
	tmp, err := os.MkdirTemp("", "gofact-ws-test-")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_CONFIG_HOME", tmp)
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func TestAllocateSequence(t *testing.T) {
	org := newOrg(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)

	next, err := org.NextNumber(now)
	if err != nil || next != "2026001" {
		t.Fatalf("NextNumber = %q, %v ; want 2026001", next, err)
	}
	// NextNumber ne consomme pas.
	if again, _ := org.NextNumber(now); again != next {
		t.Errorf("NextNumber a consommé le compteur : %q puis %q", next, again)
	}

	for i := 1; i <= 3; i++ {
		num, err := org.Allocate(now, RegistryEntry{Client: "ACME", Fichier: "f.html"})
		if err != nil {
			t.Fatalf("Allocate #%d: %v", i, err)
		}
		if want := fmt.Sprintf("2026%03d", i); num != want {
			t.Errorf("Allocate #%d = %q, want %q", i, num, want)
		}
	}
	if n, _ := org.InvoiceCount(); n != 3 {
		t.Errorf("InvoiceCount = %d, want 3", n)
	}
	// Le journal trace chaque attribution.
	journal, err := os.ReadFile(filepath.Join(org.Path, JournalFile))
	if err != nil || strings.Count(string(journal), "\"allocation\"") != 3 {
		t.Errorf("journal attendu avec 3 allocations : %v\n%s", err, journal)
	}
}

// Deux attributions concurrentes ne doivent jamais produire le même numéro ni
// laisser un trou — c'est une obligation légale, pas un détail.
func TestAllocateConcurrent(t *testing.T) {
	org := newOrg(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)

	const n = 20
	numbers := make(chan string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Chaque goroutine rouvre le dossier : simule des processus distincts.
			o, err := Open(org.Path)
			if err != nil {
				t.Error(err)
				return
			}
			num, err := o.Allocate(now, RegistryEntry{Client: "ACME", Fichier: "f.html"})
			if err != nil {
				t.Errorf("Allocate: %v", err)
				return
			}
			numbers <- num
		}()
	}
	wg.Wait()
	close(numbers)

	seen := map[string]bool{}
	for num := range numbers {
		if seen[num] {
			t.Fatalf("numéro %s attribué deux fois", num)
		}
		seen[num] = true
	}
	if len(seen) != n {
		t.Fatalf("%d numéros distincts, want %d", len(seen), n)
	}
	// Sans trou : tous les numéros de 1 à n existent.
	for i := 1; i <= n; i++ {
		if !seen[fmt.Sprintf("2026%03d", i)] {
			t.Errorf("trou dans la séquence : 2026%03d manquant", i)
		}
	}
}

// Le registre appartient à l'utilisateur : une mise à jour ne doit perdre ni ses
// clés de premier niveau ni les champs libres de ses entrées existantes.
func TestRegistryPreservesUnknownFields(t *testing.T) {
	org := newOrg(t)
	seed := `{
  "_doc": "doc",
  "convention": "factures sans préfixe",
  "compteurs": {"2025": 7},
  "factures": [{"numero": "2025007", "client": "CESI", "note": "payée en retard"}],
  "champ_libre": {"conservé": true}
}`
	if err := os.WriteFile(filepath.Join(org.Path, RegistryFile), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := org.Allocate(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		RegistryEntry{Client: "ACME", Fichier: "f.html"}); err != nil {
		t.Fatalf("Allocate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(org.Path, RegistryFile))
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("registre réécrit illisible: %v", err)
	}
	for _, key := range []string{"_doc", "convention", "champ_libre"} {
		if _, ok := got[key]; !ok {
			t.Errorf("clé %q perdue à la réécriture", key)
		}
	}
	if !strings.Contains(string(raw), "payée en retard") {
		t.Error("champ libre d'une entrée existante perdu")
	}
	factures := got["factures"].([]any)
	if len(factures) != 2 {
		t.Fatalf("factures = %d entrées, want 2", len(factures))
	}
	if num := factures[1].(map[string]any)["numero"]; num != "2025008" {
		t.Errorf("nouveau numéro = %v, want 2025008 (compteur repris à 7)", num)
	}
}

func TestIdentityNeverExposesSecrets(t *testing.T) {
	org := newOrg(t)
	env := "SUPERPDP_CLIENT_ID=\"app\"\nSUPERPDP_CLIENT_SECRET=\"tres-secret\"\nGOFACT_SELLER_NAME=\"Studio Exemple\"\nGOFACT_PAYEE_IBAN=\"FR7630001007941234567890185\"\n"
	if err := os.WriteFile(filepath.Join(org.Path, EnvFile), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	org, err := Open(org.Path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(org.Identity())
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"tres-secret", "FR7630001007941234567890185"} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("Identity expose %q :\n%s", leak, raw)
		}
	}
	var id Identity
	_ = json.Unmarshal(raw, &id)
	if !id.HasIBAN || !id.HasPDP {
		t.Errorf("Identity doit signaler la présence de l'IBAN et de la PDP : %+v", id)
	}
	if !IsSecretKey("SUPERPDP_CLIENT_SECRET") || IsSecretKey("GOFACT_SELLER_NAME") {
		t.Error("IsSecretKey mal calibrée")
	}
}

func TestClientsFromHistory(t *testing.T) {
	org := newOrg(t)
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(org.Path, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("2026001 - ACME.json", `{"number":"2026001","issue_date":"2026-01-10",
	  "buyer":{"name":"ACME SAS","siret":"55208131700015","city":"Paris"},
	  "lines":[{"name":"x","unit_price_ht_cents":100}]}`)
	write("2026002 - ACME.json", `{"number":"2026002","issue_date":"2026-03-15",
	  "buyer":{"name":"ACME SAS","siret":"55208131700015","city":"Lyon",
	           "electronic_address":"552081317","electronic_address_scheme":"0225"},
	  "lines":[{"name":"x","unit_price_ht_cents":100}]}`)
	write("2026003 - Burger.json", `{"number":"2026003","issue_date":"2026-02-01",
	  "buyer":{"name":"Burger Queen","siren":"123456789"},
	  "lines":[{"name":"x","unit_price_ht_cents":100}]}`)
	write("notes.json", `{"pas":"une facture"}`)

	clients, err := org.Clients()
	if err != nil {
		t.Fatalf("Clients: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("clients = %d, want 2 (le JSON étranger doit être ignoré)", len(clients))
	}
	// Trié du plus récent au plus ancien ; les coordonnées viennent de la
	// facture la plus récente (adresse de routage comprise).
	if clients[0].Name != "ACME SAS" || clients[0].City != "Lyon" ||
		clients[0].EAddr != "552081317" || clients[0].Invoices != 2 {
		t.Errorf("client le plus récent inattendu : %+v", clients[0])
	}

	found, err := org.FindClient("acme")
	if err != nil || len(found) != 1 || found[0].LastInvoice != "2026002" {
		t.Errorf("FindClient(acme) = %+v, %v", found, err)
	}
	if found, _ := org.FindClient("123456789"); len(found) != 1 || found[0].Name != "Burger Queen" {
		t.Errorf("FindClient par SIREN en échec : %+v", found)
	}
}

// Un client déjà facturé doit être retrouvé quelle que soit la façon dont son
// nom est écrit : sinon l'appelant se rabat sur l'annuaire public, où un
// homonyme peut être facturé à sa place.
func TestFindClientNormalisation(t *testing.T) {
	org := newOrg(t)
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(org.Path, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("2026001 - NeoDTx.json", `{"number":"2026001","issue_date":"2026-01-10",
	  "buyer":{"name":"Neo DTx","siret":"94522621500016","city":"Yvré-l'Évêque"},
	  "lines":[{"name":"x","unit_price_ht_cents":100}]}`)
	write("2026002 - BSTEAM.json", `{"number":"2026002","issue_date":"2026-02-10",
	  "buyer":{"name":"BS'TEAM","siren":"891720971"},
	  "lines":[{"name":"x","unit_price_ht_cents":100}]}`)
	write("2026003 - Elevage.json", `{"number":"2026003","issue_date":"2026-03-10",
	  "buyer":{"name":"Élevage Créüs","siren":"111222333"},
	  "lines":[{"name":"x","unit_price_ht_cents":100}]}`)

	for _, tc := range []struct {
		query string
		want  string // nom attendu ; "" = aucun résultat
	}{
		{"Neo DTx", "Neo DTx"},             // orthographe exacte
		{"NeoDTx", "Neo DTx"},              // espace omise — le cas qui échouait
		{"neodtx", "Neo DTx"},              // casse
		{"neo-dtx", "Neo DTx"},             // séparateur exotique
		{"94522621500016", "Neo DTx"},      // SIRET
		{"945226215", "Neo DTx"},           // SIREN, alors que l'historique n'a que le SIRET
		{"945 226 215", "Neo DTx"},         // SIREN espacé
		{"bsteam", "BS'TEAM"},              // apostrophe omise
		{"BS TEAM", "BS'TEAM"},             // apostrophe remplacée par une espace
		{"891720971", "BS'TEAM"},           // SIREN exact
		{"elevage creus", "Élevage Créüs"}, // accents omis
		{"Élevage", "Élevage Créüs"},       // accents présents
		{"introuvable", ""},                // pas de faux positif
		{"", ""},                           // requête vide : aucun résultat
	} {
		found, err := org.FindClient(tc.query)
		if err != nil {
			t.Fatalf("FindClient(%q): %v", tc.query, err)
		}
		if tc.want == "" {
			if len(found) != 0 {
				t.Errorf("FindClient(%q) = %d résultat(s), want 0", tc.query, len(found))
			}
			continue
		}
		if len(found) != 1 {
			t.Errorf("FindClient(%q) = %d résultat(s), want 1 (%s)", tc.query, len(found), tc.want)
			continue
		}
		if found[0].Name != tc.want {
			t.Errorf("FindClient(%q) = %q, want %q", tc.query, found[0].Name, tc.want)
		}
	}
}

func TestTemplateFreezeAndDrift(t *testing.T) {
	org := newOrg(t)
	ref := `<style>.page{width:210mm}</style><div class="page"><h1>Facture {{NUMERO}}</h1><p class="total">100 €</p></div>`

	frozen, err := org.FreezeTemplate(ref)
	if err != nil || !frozen {
		t.Fatalf("FreezeTemplate: frozen=%v err=%v", frozen, err)
	}
	// Jamais d'écrasement.
	if frozen, _ = org.FreezeTemplate("<div>autre</div>"); frozen {
		t.Error("FreezeTemplate a remplacé un modèle existant")
	}

	// Même structure, contenu différent : pas de dérive.
	same := `<style>.page{width:210mm}</style><div class="page"><h1>Facture {{NUMERO}}</h1><p class="total">2 500 €</p></div>`
	if warn, _ := org.TemplateDrift(same); warn != "" {
		t.Errorf("faux positif de dérive : %s", warn)
	}
	// Structure différente : dérive signalée.
	other := `<div class="invoice-grid"><span>Facture</span></div>`
	if warn, _ := org.TemplateDrift(other); warn == "" {
		t.Error("dérive structurelle non détectée")
	}
}

func TestDiscoverPriorities(t *testing.T) {
	org := newOrg(t)
	// Chemin explicite : seul résultat, même si d'autres sources existent.
	orgs, err := Discover(org.Path)
	if err != nil || len(orgs) != 1 || orgs[0].Path != org.Path {
		t.Fatalf("Discover(explicit) = %v, %v", orgs, err)
	}
	// Init a inscrit le dossier au registre utilisateur → découvert sans rien.
	orgs, err = Discover("")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range orgs {
		if o.Path == org.Path {
			found = true
		}
	}
	if !found {
		t.Errorf("le dossier inscrit par Init doit être découvert, obtenu %d orgs", len(orgs))
	}
	// GOFACT_INVOICES_DIR reste honorée (compatibilité skill historique).
	other := newOrg(t)
	t.Setenv("GOFACT_INVOICES_DIR", other.Path)
	orgs, _ = Discover("")
	found = false
	for _, o := range orgs {
		if o.Path == other.Path {
			found = true
		}
	}
	if !found {
		t.Error("GOFACT_INVOICES_DIR doit être découverte")
	}
}

// Reprise d'une numérotation existante : le compteur monte, jamais ne descend.
func TestNumberingTakeover(t *testing.T) {
	org, err := Init(filepath.Join(t.TempDir(), "orga"), nil, "2026011")
	if err != nil {
		t.Fatalf("Init avec reprise : %v", err)
	}
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if next, _ := org.NextNumber(now); next != "2026012" {
		t.Fatalf("NextNumber = %s, want 2026012", next)
	}
	// Idempotent au même niveau ; refus en dessous.
	if err := org.RaiseCounter(2026, 11); err != nil {
		t.Errorf("RaiseCounter au même niveau doit être accepté : %v", err)
	}
	if err := org.RaiseCounter(2026, 5); err == nil {
		t.Error("abaisser le compteur doit être refusé — réutilisation de numéros")
	}
	// L'attribution suit la reprise.
	if num, err := org.Allocate(now, RegistryEntry{Client: "ACME", Fichier: "f.html"}); err != nil || num != "2026012" {
		t.Errorf("Allocate = %q, %v ; want 2026012", num, err)
	}
	// Numéro invalide → erreur explicite.
	if _, _, err := ParseNumber("FAC-2026-11"); err == nil {
		t.Error("ParseNumber doit refuser un format étranger")
	}
}

// Le remplacement délibéré du modèle est permis et journalisé — la mise en
// page n'est pas gravée dans le marbre, la numérotation l'est.
func TestReplaceTemplate(t *testing.T) {
	org := newOrg(t)
	if _, err := org.FreezeTemplate("<div class=\"v1\">a</div>"); err != nil {
		t.Fatal(err)
	}
	if err := org.ReplaceTemplate("<div class=\"v2\">b</div>"); err != nil {
		t.Fatal(err)
	}
	tpl, _ := org.Template()
	if !strings.Contains(tpl, "v2") {
		t.Errorf("le modèle doit être remplacé, obtenu %q", tpl)
	}
	journal, _ := os.ReadFile(filepath.Join(org.Path, JournalFile))
	if !strings.Contains(string(journal), "template_replaced") {
		t.Error("le remplacement du modèle doit être journalisé")
	}
}

// Une organisation créée sans IBAN est une impasse tant qu'on ne peut pas la
// corriger : la facture échoue sur BR-50 à la dernière étape et Init refuse
// d'écraser le dossier. UpdateIdentity est la sortie — à condition de ne rien
// détruire au passage, en particulier les identifiants de plateforme que
// l'utilisateur a pu ajouter à la main.
func TestUpdateIdentityPreservesEverythingElse(t *testing.T) {
	org := newOrg(t)
	envPath := filepath.Join(org.Path, EnvFile)

	raw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("lecture du .env : %v", err)
	}
	extra := string(raw) + "\n# ajouté à la main par l'utilisateur\n" +
		"SUPERPDP_CLIENT_ID=\"abc\"\nSUPERPDP_CLIENT_SECRET=\"chut\"\n"
	if err := os.WriteFile(envPath, []byte(extra), 0o600); err != nil {
		t.Fatalf("écriture du .env : %v", err)
	}
	org, err = Open(org.Path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if org.Config().IBAN != "" {
		t.Fatal("IBAN présent alors que l'organisation a été créée sans")
	}
	if err := org.UpdateIdentity(map[string]string{
		"GOFACT_PAYEE_IBAN":   "FR7630006000011234567890189",
		"GOFACT_SELLER_CITY":  "Bordeaux",
		"GOFACT_SELLER_SIRET": "98765432100019",
	}); err != nil {
		t.Fatalf("UpdateIdentity: %v", err)
	}

	if got := org.Config().IBAN; got != "FR7630006000011234567890189" {
		t.Errorf("IBAN non pris en compte : %q", got)
	}
	if got := org.Identity().City; got != "Bordeaux" {
		t.Errorf("ville non ajoutée : %q", got)
	}
	if got := org.Identity().SIRET; got != "98765432100019" {
		t.Errorf("SIRET non remplacé : %q", got)
	}
	if got := org.Identity().Name; got != "Studio Exemple" {
		t.Errorf("le nom a bougé sans qu'on le demande : %q", got)
	}

	// Ce qui n'était pas demandé doit avoir survécu, secrets et commentaire compris.
	after, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("relecture du .env : %v", err)
	}
	for _, want := range []string{"SUPERPDP_CLIENT_ID", `SUPERPDP_CLIENT_SECRET="chut"`, "ajouté à la main"} {
		if !strings.Contains(string(after), want) {
			t.Errorf("%q a disparu du .env :\n%s", want, after)
		}
	}
	if strings.Count(string(after), "GOFACT_SELLER_SIRET") != 1 {
		t.Errorf("SIRET dupliqué au lieu d'être remplacé :\n%s", after)
	}

	// Une valeur vide efface la clé — seul moyen de revenir à la franchise.
	if err := org.UpdateIdentity(map[string]string{"GOFACT_SELLER_CITY": ""}); err != nil {
		t.Fatalf("UpdateIdentity (effacement) : %v", err)
	}
	if got := org.Identity().City; got != "" {
		t.Errorf("ville non effacée : %q", got)
	}

	// Et on ne touche pas à ce qui n'est pas de l'identité.
	if err := org.UpdateIdentity(map[string]string{"SUPERPDP_CLIENT_SECRET": "vole"}); err == nil {
		t.Error("modification d'un secret de plateforme acceptée")
	}
}

func TestDefaultTemplate(t *testing.T) {
	org := newOrg(t)
	if err := org.UpdateIdentity(map[string]string{
		"GOFACT_SELLER_ADDRESS": "1 rue Sainte-Catherine",
		"GOFACT_SELLER_CITY":    "Bordeaux",
		"GOFACT_PAYEE_IBAN":     "FR7630001007941234567890185",
	}); err != nil {
		t.Fatalf("UpdateIdentity: %v", err)
	}

	html, err := org.DefaultTemplate()
	if err != nil {
		t.Fatalf("DefaultTemplate: %v", err)
	}

	// Le jeton survit au rendu : c'est le serveur qui inscrit le numéro.
	if !strings.Contains(html, "{{NUMERO}}") {
		t.Error("le jeton {{NUMERO}} a disparu du modèle par défaut")
	}
	// Pas d'action de template non résolue.
	if strings.Contains(html, "[[") || strings.Contains(html, "]]") {
		t.Error("délimiteur de template non résolu dans le modèle par défaut")
	}
	// L'identité de l'organisation est renseignée, IBAN groupé par quatre.
	for _, want := range []string{
		"Studio Exemple", "1 rue Sainte-Catherine", "Bordeaux",
		"SIRET 12345678900014", "FR76 3000 1007 9412 3456 7890 185",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("modèle par défaut : %q absent", want)
		}
	}
	// Les mentions légales obligatoires ne dépendent pas du modèle de langage.
	for _, want := range []string{"293 B du CGI", "trois fois le taux d'intérêt légal", "40 €", "Escompte"} {
		if !strings.Contains(html, want) {
			t.Errorf("mention légale absente du modèle par défaut : %q", want)
		}
	}
	// Contraintes de rendu Chrome hors ligne.
	if strings.Contains(html, "<a href") || strings.Contains(html, "@import") {
		t.Error("le modèle par défaut ne doit contenir ni lien ni ressource externe")
	}

	// Sans IBAN ni adresse, le rendu reste valide et n'annonce pas de virement.
	bare := newOrg(t)
	html, err = bare.DefaultTemplate()
	if err != nil {
		t.Fatalf("DefaultTemplate (identité minimale): %v", err)
	}
	if strings.Contains(html, "IBAN") {
		t.Error("mention IBAN rendue alors qu'aucun IBAN n'est configuré")
	}
}
