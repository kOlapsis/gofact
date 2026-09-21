package workspace

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// EnvExportDir désigne le dossier où l'export comptable est écrit.
const EnvExportDir = "GOFACT_EXPORT_DIR"

// ExportResult résume un export mensuel.
type ExportResult struct {
	Dir      string   `json:"dir"`
	Issued   int      `json:"issued"`
	Received int      `json:"received"`
	Copied   int      `json:"copied"`
	Missing  []string `json:"missing,omitempty"`
}

// Export écrit dans dest/<organisation>/<AAAA-MM>/ les PDF émis et reçus du mois, avec un récapitulatif CSV.
func (o *Org) Export(month, dest string) (ExportResult, error) {
	if _, err := time.Parse("2006-01", month); err != nil {
		return ExportResult{}, fmt.Errorf("mois %q invalide : format attendu AAAA-MM", month)
	}
	if strings.TrimSpace(dest) == "" {
		return ExportResult{}, fmt.Errorf("aucune destination d'export : renseigner %s dans le .env de l'organisation, "+
			"ou indiquer un dossier", EnvExportDir)
	}
	res := ExportResult{Dir: filepath.Join(dest, safeName(o.Name()), month)}
	for _, sub := range []string{"emises", "recues"} {
		if err := os.MkdirAll(filepath.Join(res.Dir, sub), 0o755); err != nil {
			return res, err
		}
	}

	payments, err := o.Payments()
	if err != nil {
		return res, err
	}
	rows := [][]string{{"sens", "numero", "date", "tiers", "montant_ht", "tva", "montant_ttc", "devise",
		"encaissement", "fichier"}}

	invoices, err := o.Invoices()
	if err != nil {
		return res, err
	}
	for i := len(invoices) - 1; i >= 0; i-- {
		inv := invoices[i]
		date, _ := inv["date_emission"].(string)
		if !strings.HasPrefix(date, month) {
			continue
		}
		number, _ := inv["numero"].(string)
		client, _ := inv["client"].(string)
		file, _ := inv["fichier"].(string)
		res.Issued++
		pdf := strings.TrimSuffix(file, filepath.Ext(file)) + ".pdf"
		copied, err := copyIfChanged(filepath.Join(o.Path, pdf), filepath.Join(res.Dir, "emises", filepath.Base(pdf)))
		if err != nil {
			res.Missing = append(res.Missing, number)
		} else if copied {
			res.Copied++
		}
		ht, vat, ttc, currency := o.issuedTotals(inv)
		rows = append(rows, []string{"emise", number, date, client, ht, vat, ttc, currency,
			payments[number].Date, "emises/" + filepath.Base(pdf)})
	}

	received, err := o.ReceivedInvoices(month)
	if err != nil {
		return res, err
	}
	for i := len(received) - 1; i >= 0; i-- {
		r := received[i]
		res.Received++
		copied, err := copyIfChanged(filepath.Join(o.Path, r.File), filepath.Join(res.Dir, "recues", filepath.Base(r.File)))
		if err != nil {
			res.Missing = append(res.Missing, r.Number)
		} else if copied {
			res.Copied++
		}
		rows = append(rows, []string{"recue", r.Number, r.IssueDate, r.Seller, frDecimal(r.TotalHT),
			frDecimal(r.TotalVAT), frDecimal(r.TotalTTC), r.Currency, "", "recues/" + filepath.Base(r.File)})
	}

	var buf bytes.Buffer
	buf.WriteString("\xef\xbb\xbf") // BOM : sans lui, Excel lit le CSV en Windows-1252.
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	if err := w.WriteAll(rows); err != nil {
		return res, err
	}
	return res, writeAtomic(filepath.Join(res.Dir, "recapitulatif.csv"), buf.Bytes())
}

// issuedTotals lit les totaux d'une facture émise dans son sidecar, avec repli sur le HT du registre.
func (o *Org) issuedTotals(inv map[string]any) (ht, vat, ttc, currency string) {
	currency = "EUR"
	if cents, ok := inv["montant_ht_cents"].(float64); ok {
		ht = frCents(int64(cents))
	}
	number, _ := inv["numero"].(string)
	full, err := o.issuedInvoice(number)
	if err != nil {
		return ht, "", "", currency
	}
	if full.Currency != "" {
		currency = full.Currency
	}
	return frCents(full.TaxBasisTotal), frCents(full.TaxTotal), frCents(full.GrandTotal), currency
}

func copyIfChanged(src, dst string) (bool, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return false, err
	}
	if prev, err := os.ReadFile(dst); err == nil && bytes.Equal(prev, data) {
		return false, nil
	}
	return true, writeAtomic(dst, data)
}

func frCents(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return fmt.Sprintf("%s%d,%02d", sign, c/100, c%100)
}

func frDecimal(s string) string {
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return s
	}
	return strings.Replace(s, ".", ",", 1)
}
