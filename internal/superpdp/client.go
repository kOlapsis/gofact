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

// Event est un statut du cycle de vie d'une facture.
type Event struct {
	CreatedAt  string `json:"created_at"`
	StatusCode string `json:"status_code"`
	StatusText string `json:"status_text"`
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
