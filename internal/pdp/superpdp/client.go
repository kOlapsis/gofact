// Package superpdp est un client minimal de la PDP SuperPDP (superpdp.tech) :
// authentification OAuth2 (client credentials), dépôt, suivi et encaissement
// des factures émises, récupération des factures reçues.
package superpdp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// DefaultBase est l'URL de l'API SuperPDP.
const DefaultBase = "https://api.superpdp.tech"

// Config porte les identifiants d'application (OAuth2 client credentials).
type Config struct {
	Base         string
	ClientID     string
	ClientSecret string
}

// Client appelle l'API SuperPDP.
type Client struct {
	cfg   Config
	http  *http.Client
	token string
}

// New construit un client. base vide ⇒ DefaultBase.
func New(cfg Config) *Client {
	if cfg.Base == "" {
		cfg.Base = DefaultBase
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: 90 * time.Second}}
}

// Event est un statut du cycle de vie d'une facture. Sur un rejet (fr:213),
// l'API porte le motif dans data.reason et le détail règle par règle dans
// details[].notes[].contents[].content — c'est là que se lit « BR-FR-05/BT-22 :
// la mention PMT est absente », pas dans status_text. Les deux champs sont
// gardés bruts : leur forme varie selon l'événement, et un événement illisible
// ne doit pas faire échouer la lecture du cycle de vie entier.
type Event struct {
	CreatedAt  string          `json:"created_at"`
	StatusCode string          `json:"status_code"`
	StatusText string          `json:"status_text"`
	Data       json.RawMessage `json:"data,omitempty"`
	Details    json.RawMessage `json:"details,omitempty"`
}

// eventData est la partie exploitée de data.
type eventData struct {
	Reason string `json:"reason"`
}

// eventDetail est un détail de rejet : un motif codé et ses notes explicatives.
type eventDetail struct {
	Reason string `json:"reason"`
	Notes  []struct {
		ContentCode string `json:"content_code"`
		Subject     string `json:"subject"`
		Contents    []struct {
			Content string `json:"content"`
		} `json:"contents"`
	} `json:"notes"`
}

// Reasons aplatit les motifs portés par l'événement, un par ligne, dans l'ordre
// : le motif général (data.reason), puis chaque contenu de note des détails —
// ou leur code (content_code, sinon reason) quand une note n'a pas de texte.
// Vide pour un événement ordinaire.
func (e Event) Reasons() []string {
	var out []string
	seen := map[string]bool{}
	push := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	var data eventData
	if len(e.Data) > 0 && json.Unmarshal(e.Data, &data) == nil {
		push(data.Reason)
	}
	var details []eventDetail
	if len(e.Details) > 0 && json.Unmarshal(e.Details, &details) == nil {
		for _, d := range details {
			pushed := false
			for _, n := range d.Notes {
				withText := false
				for _, c := range n.Contents {
					if strings.TrimSpace(c.Content) != "" {
						push(c.Content)
						withText, pushed = true, true
					}
				}
				if !withText && n.ContentCode != "" {
					push(n.ContentCode)
					pushed = true
				}
			}
			if !pushed {
				push(d.Reason)
			}
		}
	}
	return out
}

// Invoice est la facture telle que renvoyée par l'API.
type Invoice struct {
	ID        int64   `json:"id"`
	CompanyID int64   `json:"company_id"`
	Direction string  `json:"direction"`
	Events    []Event `json:"events"`
}

// apiError décode le corps d'erreur JSON de l'API.
type apiError struct {
	Status  int    `json:"http_status_code"`
	Message string `json:"message"`
}

// Authenticate échange les identifiants contre un jeton bearer (client credentials).
func (c *Client) Authenticate(ctx context.Context) error {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Base+"/oauth2/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("superpdp: requête token: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("superpdp: appel token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("superpdp: authentification refusée (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil || tok.AccessToken == "" {
		return fmt.Errorf("superpdp: jeton illisible: %s", strings.TrimSpace(string(body)))
	}
	c.token = tok.AccessToken
	return nil
}

// SendPDF dépose un PDF Factur-X (Content-Type application/pdf) et renvoie la
// facture créée. Authenticate doit avoir été appelé au préalable.
func (c *Client) SendPDF(ctx context.Context, pdfPath string) (*Invoice, error) {
	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("superpdp: lecture PDF: %w", err)
	}
	body, err := c.do(ctx, http.MethodPost, "/v1.beta/invoices", "application/pdf", bytes.NewReader(pdf))
	if err != nil {
		return nil, err
	}
	var inv Invoice
	if err := json.Unmarshal(body, &inv); err != nil {
		return nil, fmt.Errorf("superpdp: réponse illisible: %s", strings.TrimSpace(string(body)))
	}
	return &inv, nil
}

// GetInvoice récupère une facture et ses statuts (suivi du cycle de vie).
func (c *Client) GetInvoice(ctx context.Context, id int64) (*Invoice, error) {
	body, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1.beta/invoices/%d", id), "", nil)
	if err != nil {
		return nil, err
	}
	var inv Invoice
	if err := json.Unmarshal(body, &inv); err != nil {
		return nil, fmt.Errorf("superpdp: réponse illisible: %s", strings.TrimSpace(string(body)))
	}
	return &inv, nil
}

// GetInvoiceFile télécharge une facture dans le format demandé (factur-x, cii, ubl, original).
func (c *Client) GetInvoiceFile(ctx context.Context, id int64, format string) ([]byte, error) {
	return c.do(ctx, http.MethodGet,
		fmt.Sprintf("/v1.beta/invoices/%d?format=%s", id, url.QueryEscape(format)), "", nil)
}

// Overview est une facture de la liste, avec son contenu EN 16931 résumé.
type Overview struct {
	ID        int64     `json:"id"`
	Direction string    `json:"direction"`
	CreatedAt string    `json:"created_at"`
	EN        ENInvoice `json:"en_invoice"`
}

// ENInvoice est la partie du contenu EN 16931 exploitée par gofact.
type ENInvoice struct {
	Number       string `json:"number"`
	IssueDate    string `json:"issue_date"`
	CurrencyCode string `json:"currency_code"`
	Seller       struct {
		Name  string `json:"name"`
		Legal struct {
			Value string `json:"value"`
		} `json:"legal_registration_identifier"`
	} `json:"seller"`
	Totals struct {
		WithoutVAT Amount `json:"total_without_vat"`
		VAT        Amount `json:"total_vat_amount"`
		WithVAT    Amount `json:"total_with_vat"`
	} `json:"totals"`
}

// Amount est un montant décimal que l'API sert tantôt en chaîne, tantôt en
// objet {value, currency_code}.
type Amount string

func (a *Amount) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*a = Amount(s)
		return nil
	}
	var o struct {
		Value json.RawMessage `json:"value"`
	}
	if json.Unmarshal(b, &o) == nil && len(o.Value) > 0 {
		*a = Amount(strings.Trim(string(o.Value), `"`))
	}
	return nil
}

// ListInvoices renvoie, toutes pages comprises, les factures d'un sens ("in" ou "out") d'id supérieur à afterID.
func (c *Client) ListInvoices(ctx context.Context, direction string, afterID int64) ([]Overview, error) {
	var out []Overview
	for {
		q := url.Values{
			"direction":         {direction},
			"order":             {"asc"},
			"limit":             {"1000"},
			"starting_after_id": {fmt.Sprint(afterID)},
			"expand[]":          {"en_invoice", "en_invoice.seller"},
		}
		body, err := c.do(ctx, http.MethodGet, "/v1.beta/invoices?"+q.Encode(), "", nil)
		if err != nil {
			return nil, err
		}
		var page struct {
			Data     []Overview `json:"data"`
			HasAfter bool       `json:"has_after"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("superpdp: liste illisible: %s", strings.TrimSpace(string(body)))
		}
		out = append(out, page.Data...)
		if !page.HasAfter || len(page.Data) == 0 {
			return out, nil
		}
		afterID = page.Data[len(page.Data)-1].ID
	}
}

// ReportedData est une donnée jointe à un événement de cycle de vie (MDG-43).
type ReportedData struct {
	TypeCode     string `json:"type_code"`
	Amount       string `json:"amount,omitempty"`
	CurrencyCode string `json:"currency_code,omitempty"`
	Date         string `json:"date,omitempty"`
	ValuePercent string `json:"value_percent,omitempty"`
}

// PostInvoiceEvent ajoute un statut au cycle de vie d'une facture (fr:212 « Encaissée »…).
func (c *Client) PostInvoiceEvent(ctx context.Context, id int64, status string, data []ReportedData) error {
	payload := map[string]any{"invoice_id": id, "status_code": status}
	if len(data) > 0 {
		payload["details"] = []map[string]any{{"reported_data": data}}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = c.do(ctx, http.MethodPost, "/v1.beta/invoice_events", "application/json", bytes.NewReader(raw))
	return err
}

// do exécute un appel authentifié et renvoie le corps d'une réponse 2xx.
func (c *Client) do(ctx context.Context, method, path, contentType string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.Base+path, body)
	if err != nil {
		return nil, fmt.Errorf("superpdp: requête %s: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("superpdp: appel %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, decodeError(resp.StatusCode, raw)
	}
	return raw, nil
}

func decodeError(status int, body []byte) error {
	var e apiError
	if json.Unmarshal(body, &e) == nil && e.Message != "" {
		return fmt.Errorf("superpdp: rejet (HTTP %d): %s", status, e.Message)
	}
	return fmt.Errorf("superpdp: rejet (HTTP %d): %s", status, strings.TrimSpace(string(body)))
}
