package superpdp

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kolapsis/gofact/internal/pdp"
)

// Adaptation du client SuperPDP à l'interface pdp.Provider.

func init() {
	pdp.RegisterProvider("superpdp", func(lookup func(string) string) (pdp.Provider, error) {
		cfg := Config{
			Base:         lookup("SUPERPDP_BASE"),
			ClientID:     lookup("SUPERPDP_CLIENT_ID"),
			ClientSecret: lookup("SUPERPDP_CLIENT_SECRET"),
		}
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return nil, fmt.Errorf("aucun compte SuperPDP configuré : renseigner SUPERPDP_CLIENT_ID et " +
				"SUPERPDP_CLIENT_SECRET dans le .env de l'organisation (identifiants fournis par la " +
				"plateforme). Ces valeurs ne se communiquent jamais en conversation : elles se placent " +
				"directement dans le fichier")
		}
		return &provider{cli: New(cfg)}, nil
	})
}

type provider struct{ cli *Client }

func (p *provider) Name() string { return "superpdp" }

func (p *provider) Send(ctx context.Context, pdfPath string) (pdp.Receipt, error) {
	if err := p.cli.Authenticate(ctx); err != nil {
		return pdp.Receipt{}, err
	}
	inv, err := p.cli.SendPDF(ctx, pdfPath)
	if err != nil {
		return pdp.Receipt{}, err
	}
	return pdp.Receipt{
		Provider:  p.Name(),
		Reference: strconv.FormatInt(inv.ID, 10),
		Events:    convertEvents(inv.Events),
	}, nil
}

func (p *provider) Status(ctx context.Context, reference string) ([]pdp.Event, error) {
	id, err := strconv.ParseInt(reference, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("superpdp: référence %q invalide", reference)
	}
	if err := p.cli.Authenticate(ctx); err != nil {
		return nil, err
	}
	inv, err := p.cli.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	return convertEvents(inv.Events), nil
}

func convertEvents(events []Event) []pdp.Event {
	out := make([]pdp.Event, 0, len(events))
	for _, e := range events {
		out = append(out, pdp.Event{CreatedAt: e.CreatedAt, StatusCode: e.StatusCode, StatusText: e.StatusText})
	}
	return out
}
