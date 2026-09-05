// Package superpdp est un client minimal de la PDP SuperPDP (superpdp.tech) :
// authentification OAuth2 (client credentials) et dépôt d'une facture Factur-X.
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Base+"/v1.beta/invoices", bytes.NewReader(pdf))
	if err != nil {
		return nil, fmt.Errorf("superpdp: requête dépôt: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/pdf")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("superpdp: appel dépôt: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, decodeError(resp.StatusCode, body)
	}
	var inv Invoice
	if err := json.Unmarshal(body, &inv); err != nil {
		return nil, fmt.Errorf("superpdp: réponse illisible: %s", strings.TrimSpace(string(body)))
	}
	return &inv, nil
}

// GetInvoice récupère une facture et ses statuts (suivi du cycle de vie).
func (c *Client) GetInvoice(ctx context.Context, id int64) (*Invoice, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1.beta/invoices/%d", c.cfg.Base, id), nil)
	if err != nil {
		return nil, fmt.Errorf("superpdp: requête statut: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("superpdp: appel statut: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, decodeError(resp.StatusCode, body)
	}
	var inv Invoice
	if err := json.Unmarshal(body, &inv); err != nil {
		return nil, fmt.Errorf("superpdp: réponse illisible: %s", strings.TrimSpace(string(body)))
	}
	return &inv, nil
}

func decodeError(status int, body []byte) error {
	var e apiError
	if json.Unmarshal(body, &e) == nil && e.Message != "" {
		return fmt.Errorf("superpdp: rejet (HTTP %d): %s", status, e.Message)
	}
	return fmt.Errorf("superpdp: rejet (HTTP %d): %s", status, strings.TrimSpace(string(body)))
}
