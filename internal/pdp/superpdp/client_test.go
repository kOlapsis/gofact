package superpdp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kolapsis/gofact/internal/pdp"
)

// Réponse réelle (anonymisée) de GET /v1.beta/invoices/{id} après le rejet
// d'une facture privée de ses mentions légales : les motifs ne sont ni dans
// status_code ni dans status_text, mais dans data et details.
const rejectedInvoiceJSON = `{
  "id": 453372, "company_id": 7, "direction": "outbound",
  "events": [
    {"created_at": "2026-09-05T09:12:01Z", "status_code": "api:uploaded", "status_text": "Déposée", "data": {}},
    {"created_at": "2026-09-05T09:12:02Z", "status_code": "fr:213", "status_text": "Rejetée",
     "data": {"reason": "Facture non conforme aux règles BR-FR"},
     "details": [{"reason": "REJ_SEMAN", "notes": [
       {"content_code": "BR-FR-05_BT-22_PMT", "subject": "AAO",
        "contents": [{"content": "BR-FR-05/BT-22 : La mention relative aux frais de recouvrement (code PMT) est absente. Elle est obligatoire dans les notes (BG-1)."}]},
       {"content_code": "BR-FR-05_BT-22_PMD", "subject": "AAO",
        "contents": [{"content": "BR-FR-05/BT-22 : La mention relative aux pénalités de retard (code PMD) est absente."}]},
       {"content_code": "BR-FR-05_BT-22_AAB", "subject": "AAO", "contents": []}
     ]}]},
    {"created_at": "2026-09-05T09:12:03Z", "status_code": "ppf:rejected", "status_text": "Rejetée par le PPF", "data": null, "details": []}
  ]
}`

func TestRejectionReasonsAreDecoded(t *testing.T) {
	var inv Invoice
	if err := json.Unmarshal([]byte(rejectedInvoiceJSON), &inv); err != nil {
		t.Fatalf("décodage : %v", err)
	}
	events := convertEvents(inv.Events)
	if len(events) != 3 {
		t.Fatalf("3 événements attendus, %d obtenus", len(events))
	}
	if len(events[0].Reasons) != 0 {
		t.Errorf("un dépôt ordinaire ne porte aucun motif : %v", events[0].Reasons)
	}
	got := events[1].Reasons
	wants := []string{
		"Facture non conforme aux règles BR-FR",
		"frais de recouvrement (code PMT)",
		"pénalités de retard (code PMD)",
		"BR-FR-05_BT-22_AAB", // note sans texte : son code fait foi
	}
	for _, w := range wants {
		found := false
		for _, r := range got {
			if strings.Contains(r, w) {
				found = true
			}
		}
		if !found {
			t.Errorf("motif %q absent de %v", w, got)
		}
	}

	rejected, reasons := pdp.Rejection(events)
	if !rejected {
		t.Fatal("le cycle de vie doit être vu comme rejeté")
	}
	if len(reasons) != 5 { // 4 motifs détaillés + le libellé du ppf:rejected sans détail
		t.Errorf("5 motifs attendus, obtenu %d : %v", len(reasons), reasons)
	}
}

// Un événement dont data ou details ont une forme inattendue ne doit pas
// empêcher la lecture du reste du cycle de vie.
func TestUnexpectedEventShapesAreTolerated(t *testing.T) {
	raw := `{"id": 1, "events": [
	  {"status_code": "fr:200", "status_text": "Déposée", "data": "texte libre", "details": {"pas": "une liste"}},
	  {"status_code": "fr:201", "status_text": "Émise", "data": [1,2]}
	]}`
	var inv Invoice
	if err := json.Unmarshal([]byte(raw), &inv); err != nil {
		t.Fatalf("décodage : %v", err)
	}
	events := convertEvents(inv.Events)
	if rejected, _ := pdp.Rejection(events); rejected {
		t.Error("aucun rejet attendu")
	}
	for _, e := range events {
		if len(e.Reasons) != 0 {
			t.Errorf("aucun motif attendu sur %s : %v", e.StatusCode, e.Reasons)
		}
	}
}
