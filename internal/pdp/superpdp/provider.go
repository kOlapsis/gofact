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
	id, err := parseRef(reference)
	if err != nil {
		return nil, err
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

func (p *provider) ReportPayment(ctx context.Context, reference string, pay pdp.Payment) error {
	id, err := parseRef(reference)
	if err != nil {
		return err
	}
	if err := p.cli.Authenticate(ctx); err != nil {
		return err
	}
	var data []ReportedData
	for _, part := range pay.Parts {
		data = append(data, ReportedData{TypeCode: "MEN", Amount: part.Amount, CurrencyCode: pay.Currency,
			Date: pay.Date, ValuePercent: part.VATRate})
	}
	return p.cli.PostInvoiceEvent(ctx, id, "fr:212", data)
}

func (p *provider) Received(ctx context.Context, afterID int64) ([]pdp.Incoming, error) {
	if err := p.cli.Authenticate(ctx); err != nil {
		return nil, err
	}
	list, err := p.cli.ListInvoices(ctx, "in", afterID)
	if err != nil {
		return nil, err
	}
	out := make([]pdp.Incoming, 0, len(list))
	for _, inv := range list {
		currency := inv.EN.CurrencyCode
		if currency == "" {
			currency = "EUR"
		}
		out = append(out, pdp.Incoming{
			Provider:   p.Name(),
			Reference:  strconv.FormatInt(inv.ID, 10),
			Cursor:     inv.ID,
			ReceivedAt: inv.CreatedAt,
			Number:     inv.EN.Number,
			IssueDate:  inv.EN.IssueDate,
			Seller:     inv.EN.Seller.Name,
			SellerID:   inv.EN.Seller.Legal.Value,
			TotalHT:    string(inv.EN.Totals.WithoutVAT),
			TotalVAT:   string(inv.EN.Totals.VAT),
			TotalTTC:   string(inv.EN.Totals.WithVAT),
			Currency:   currency,
		})
	}
	return out, nil
}

func (p *provider) Download(ctx context.Context, reference string) ([]byte, error) {
	id, err := parseRef(reference)
	if err != nil {
		return nil, err
	}
	if err := p.cli.Authenticate(ctx); err != nil {
		return nil, err
	}
	return p.cli.GetInvoiceFile(ctx, id, "factur-x")
}

func parseRef(reference string) (int64, error) {
	id, err := strconv.ParseInt(reference, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("superpdp: référence %q invalide", reference)
	}
	return id, nil
}

func convertEvents(events []Event) []pdp.Event {
	out := make([]pdp.Event, 0, len(events))
	for _, e := range events {
		out = append(out, pdp.Event{CreatedAt: e.CreatedAt, StatusCode: e.StatusCode,
			StatusText: e.StatusText, Reasons: e.Reasons()})
	}
	return out
}
