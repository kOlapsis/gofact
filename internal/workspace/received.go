package workspace

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kolapsis/gofact/internal/facturx"
	"github.com/kolapsis/gofact/internal/pdp"
)

// ReceivedDir range les factures reçues de la plateforme, un sous-dossier par mois d'émission.
const ReceivedDir = "recues"

// SyncStateFile garde le curseur de la dernière facture reçue.
const SyncStateFile = "pdp-sync.json"

// Received est une facture reçue, rangée dans le dossier.
type Received struct {
	pdp.Incoming
	File string `json:"file"`
}

type syncState struct {
	LastInID int64 `json:"last_in_id"`
}

// SyncReceived récupère les factures reçues depuis le dernier passage et renvoie les nouvelles.
func (o *Org) SyncReceived(ctx context.Context, p pdp.Provider) ([]Received, error) {
	state, err := o.readSyncState()
	if err != nil {
		return nil, err
	}
	incoming, err := p.Received(ctx, state.LastInID)
	if err != nil {
		return nil, err
	}
	var fresh []Received
	for _, inc := range incoming {
		r, created, err := o.storeReceived(ctx, p, inc)
		if err != nil {
			return fresh, err
		}
		if created {
			fresh = append(fresh, r)
		}
		// Le curseur n'avance qu'une fois le PDF écrit : un passage interrompu reprend là où il s'est arrêté.
		if inc.Cursor > state.LastInID {
			state.LastInID = inc.Cursor
			if err := o.writeSyncState(state); err != nil {
				return fresh, err
			}
		}
	}
	return fresh, nil
}

func (o *Org) storeReceived(ctx context.Context, p pdp.Provider, inc pdp.Incoming) (Received, bool, error) {
	month := monthOf(inc.IssueDate, inc.ReceivedAt)
	dir := filepath.Join(o.Path, ReceivedDir, month)
	base := safeName(fmt.Sprintf("%s - %s - %s", firstNonEmpty(inc.IssueDate, month), firstNonEmpty(inc.Seller, "vendeur inconnu"),
		firstNonEmpty(inc.Number, inc.Reference)))
	r := Received{Incoming: inc, File: filepath.Join(ReceivedDir, month, base+".pdf")}
	pdfPath := filepath.Join(o.Path, r.File)
	if _, err := os.Stat(pdfPath); err == nil {
		return r, false, nil
	}
	pdf, err := p.Download(ctx, inc.Reference)
	if err != nil {
		return r, false, fmt.Errorf("téléchargement de la facture reçue %s : %w", inc.Reference, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return r, false, err
	}
	meta, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return r, false, err
	}
	if err := writeAtomic(filepath.Join(dir, base+".json"), meta); err != nil {
		return r, false, err
	}
	if err := writeAtomic(pdfPath, pdf); err != nil {
		return r, false, err
	}
	_ = o.Journal("pdp_received", map[string]any{"provider": inc.Provider, "reference": inc.Reference,
		"numero": inc.Number, "vendeur": inc.Seller, "fichier": r.File})
	return r, true, nil
}

// ReceivedInvoices liste les factures reçues rangées dans le dossier, pour un mois AAAA-MM ou toutes si month est vide.
func (o *Org) ReceivedInvoices(month string) ([]Received, error) {
	pattern := filepath.Join(o.Path, ReceivedDir, "*", "*.json")
	if month != "" {
		pattern = filepath.Join(o.Path, ReceivedDir, month, "*.json")
	}
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	out := []Received{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var r Received
		if json.Unmarshal(raw, &r) != nil || r.Reference == "" {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Cursor > out[j].Cursor })
	return out, nil
}

func (o *Org) readSyncState() (syncState, error) {
	var s syncState
	raw, err := os.ReadFile(filepath.Join(o.Path, SyncStateFile))
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("workspace: %s illisible : %w", SyncStateFile, err)
	}
	return s, nil
}

func (o *Org) writeSyncState(s syncState) error {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(o.Path, SyncStateFile), raw)
}

// PaymentRecord est un encaissement signalé à la plateforme.
type PaymentRecord struct {
	Date   string `json:"date"`
	Amount string `json:"amount,omitempty"`
}

// Payments renvoie, par numéro de facture, le dernier encaissement inscrit au journal.
func (o *Org) Payments() (map[string]PaymentRecord, error) {
	out := map[string]PaymentRecord{}
	err := o.scanJournal(func(event string, data json.RawMessage) {
		if event != "payment_reported" {
			return
		}
		var d struct {
			Numero string `json:"numero"`
			PaymentRecord
		}
		if json.Unmarshal(data, &d) == nil && d.Numero != "" {
			out[d.Numero] = d.PaymentRecord
		}
	})
	return out, err
}

// SentReference renvoie la référence et le fournisseur du dernier dépôt d'une facture, vides si elle n'a jamais été déposée.
func (o *Org) SentReference(number string) (ref, provider string, err error) {
	err = o.scanJournal(func(event string, data json.RawMessage) {
		if event != "pdp_sent" {
			return
		}
		var d struct {
			Numero    string `json:"numero"`
			Provider  string `json:"provider"`
			Reference string `json:"reference"`
		}
		if json.Unmarshal(data, &d) == nil && d.Numero == number {
			ref, provider = d.Reference, d.Provider
		}
	})
	return ref, provider, err
}

func (o *Org) scanJournal(fn func(event string, data json.RawMessage)) error {
	f, err := os.Open(filepath.Join(o.Path, JournalFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var line struct {
			Event string          `json:"event"`
			Data  json.RawMessage `json:"data"`
		}
		if json.Unmarshal(sc.Bytes(), &line) == nil {
			fn(line.Event, line.Data)
		}
	}
	return sc.Err()
}

func monthOf(dates ...string) string {
	for _, d := range dates {
		if len(d) >= 7 && d[4] == '-' {
			return d[:7]
		}
	}
	return "sans-date"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// safeName retire d'un nom de fichier les caractères interdits sous Windows ou macOS.
func safeName(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, s)
	return strings.TrimRight(strings.TrimSpace(s), ". ")
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ErrNotSent signale une facture qui n'a jamais été déposée sur la plateforme.
var ErrNotSent = errors.New("facture jamais déposée sur la PDP")

// PaymentReport décrit un encaissement signalé.
type PaymentReport struct {
	Number    string `json:"number"`
	Reference string `json:"reference"`
	Date      string `json:"date"`
	Amount    string `json:"amount"`
}

// ReportPayment signale à la plateforme l'encaissement d'une facture déposée et l'inscrit au journal.
func (o *Org) ReportPayment(ctx context.Context, p pdp.Provider, number, date, amount string) (PaymentReport, error) {
	today := time.Now().Format("2006-01-02")
	out := PaymentReport{Number: number, Date: strings.TrimSpace(date), Amount: strings.TrimSpace(amount)}
	if out.Date == "" {
		out.Date = today
	}
	if _, err := time.Parse("2006-01-02", out.Date); err != nil {
		return out, fmt.Errorf("date d'encaissement %q invalide : format attendu AAAA-MM-JJ", date)
	}
	var paid int64
	if out.Amount != "" {
		v, err := strconv.ParseFloat(strings.Replace(out.Amount, ",", ".", 1), 64)
		if err != nil || v <= 0 {
			return out, fmt.Errorf("montant %q invalide : un décimal positif est attendu (ex. 1200.00)", amount)
		}
		paid = int64(math.Round(v * 100))
	}
	ref, _, err := o.SentReference(number)
	if err != nil {
		return out, err
	}
	if ref == "" {
		return out, fmt.Errorf("facture %s : %w", number, ErrNotSent)
	}
	out.Reference = ref

	pay := pdp.Payment{Date: out.Date, Currency: "EUR"}
	// La plateforme refuse un encaissement daté ou partiel qui n'est pas ventilé par taux de TVA.
	if paid > 0 || out.Date != today {
		inv, err := o.issuedInvoice(number)
		if err != nil {
			return out, fmt.Errorf("facture %s : ventilation par taux de TVA introuvable (%v) — signaler "+
				"l'encaissement total sans date ni montant", number, err)
		}
		if inv.Currency != "" {
			pay.Currency = inv.Currency
		}
		if paid == 0 {
			paid = inv.GrandTotal
		}
		if paid > inv.GrandTotal {
			return out, fmt.Errorf("montant encaissé (%s) supérieur au total de la facture (%s)",
				frCents(paid), frCents(inv.GrandTotal))
		}
		pay.Parts = splitByRate(paid, inv.VATBreakdown)
	}
	if err := p.ReportPayment(ctx, ref, pay); err != nil {
		return out, err
	}
	out.Amount = "total"
	if paid > 0 {
		out.Amount = strings.Replace(frCents(paid), ",", ".", 1)
	}
	_ = o.Journal("payment_reported", map[string]any{"numero": number, "reference": ref,
		"date": out.Date, "amount": out.Amount})
	return out, nil
}

// splitByRate répartit un montant encaissé entre les taux de TVA, au prorata du TTC de chacun.
func splitByRate(paid int64, breakdown []facturx.VATSubtotal) []pdp.PaymentPart {
	var total int64
	for _, b := range breakdown {
		total += b.BasisAmount + b.CalculatedTax
	}
	parts := make([]pdp.PaymentPart, 0, len(breakdown))
	var assigned int64
	for i, b := range breakdown {
		share := paid - assigned
		if i < len(breakdown)-1 && total != 0 {
			share = int64(math.Round(float64(paid) * float64(b.BasisAmount+b.CalculatedTax) / float64(total)))
		}
		assigned += share
		parts = append(parts, pdp.PaymentPart{Amount: strings.Replace(frCents(share), ",", ".", 1), VATRate: b.RatePct})
	}
	return parts
}

// issuedInvoice reconstitue une facture émise depuis son sidecar.
func (o *Org) issuedInvoice(number string) (facturx.Invoice, error) {
	invoices, err := o.Invoices()
	if err != nil {
		return facturx.Invoice{}, err
	}
	for _, inv := range invoices {
		if n, _ := inv["numero"].(string); n != number {
			continue
		}
		file, _ := inv["fichier"].(string)
		spec, err := facturx.LoadSpec(filepath.Join(o.Path, strings.TrimSuffix(file, filepath.Ext(file))+".json"))
		if err != nil {
			return facturx.Invoice{}, err
		}
		return spec.ToInvoiceWith(o.Config())
	}
	return facturx.Invoice{}, fmt.Errorf("facture inconnue du registre")
}
