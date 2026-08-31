package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"time"
)

// Numérotation des factures. C'est une obligation légale : séquence continue,
// sans trou, jamais réutilisée. Deux conséquences de conception :
//
//   - l'attribution est ATOMIQUE et VERROUILLÉE : un serveur peut être appelé en
//     concurrence, deux appels ne produisent jamais le même numéro ;
//   - l'attribution est TRANSACTIONNELLE : le numéro n'existe qu'accompagné de
//     son entrée de registre. Pas d'API « réserver un numéro » qu'un appelant
//     pourrait consommer puis abandonner, laissant un trou.
//
// Le format des numéros est {YYYY}{NNN} — l'année puis un compteur annuel sur
// trois chiffres, sans préfixe (convention du registre existant).

// JournalFile est la piste d'audit du dossier : une ligne JSON par événement,
// en append seul — on n'y réécrit jamais rien.
const JournalFile = "journal.ndjson"

// RegistryEntry est l'entrée ajoutée au registre pour chaque facture émise.
// Les champs suivent le format du registre existant.
type RegistryEntry struct {
	Numero       string `json:"numero"`
	DateEmission string `json:"date_emission"` // ISO YYYY-MM-DD
	Client       string `json:"client"`
	Contact      string `json:"contact,omitempty"`
	Projet       string `json:"projet,omitempty"`
	MontantHT    int64  `json:"montant_ht_cents"`
	Statut       string `json:"statut"`
	Fichier      string `json:"fichier"`
	DevisRef     string `json:"devis_ref,omitempty"`
}

// NextNumber renvoie le prochain numéro SANS le consommer — pour l'annoncer à
// l'utilisateur avant confirmation. Rien ne garantit qu'il sera encore libre au
// moment d'Allocate ; seul Allocate fait foi.
func (o *Org) NextNumber(now time.Time) (string, error) {
	reg, err := o.readRegistry()
	if err != nil {
		return "", err
	}
	year := now.Format("2006")
	return fmt.Sprintf("%s%03d", year, reg.counters[year]+1), nil
}

// Allocate attribue le prochain numéro et inscrit la facture au registre, en une
// seule opération sous verrou. Renvoie le numéro attribué.
func (o *Org) Allocate(now time.Time, e RegistryEntry) (string, error) {
	return o.AllocateWith(now, func(string) (RegistryEntry, error) { return e, nil })
}

// AllocateWith attribue le prochain numéro en exécutant work SOUS LE VERROU :
// work reçoit le numéro et produit l'entrée de registre — typiquement après
// avoir généré la facture elle-même. Si work échoue, rien n'est persisté : ni
// compteur, ni entrée, ni fichier de l'appelant s'il fait son ménage. C'est la
// forme transactionnelle complète — un échec de génération ne laisse pas de
// trou dans la séquence.
func (o *Org) AllocateWith(now time.Time, work func(number string) (RegistryEntry, error)) (string, error) {
	unlock, err := o.lock()
	if err != nil {
		return "", err
	}
	defer unlock()

	reg, err := o.readRegistry()
	if err != nil {
		return "", err
	}
	year := now.Format("2006")
	next := reg.counters[year] + 1
	number := fmt.Sprintf("%s%03d", year, next)

	e, err := work(number)
	if err != nil {
		return "", err
	}
	e.Numero = number
	if e.DateEmission == "" {
		e.DateEmission = now.Format("2006-01-02")
	}
	if e.Statut == "" {
		e.Statut = "émise"
	}

	reg.counters[year] = next
	if err := reg.append(e); err != nil {
		return "", err
	}
	if err := o.writeRegistry(reg); err != nil {
		return "", err
	}
	if err := o.Journal("allocation", e); err != nil {
		return "", err
	}
	return number, nil
}

// Invoices renvoie les entrées du registre, de la plus récente à la plus
// ancienne, sans présumer de leur forme : le registre peut contenir des entrées
// antérieures à gofact avec leurs propres champs.
func (o *Org) Invoices() ([]map[string]any, error) {
	reg, err := o.readRegistry()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(reg.invoices))
	for i := len(reg.invoices) - 1; i >= 0; i-- {
		var m map[string]any
		if err := json.Unmarshal(reg.invoices[i], &m); err == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

// Journal ajoute un événement horodaté à la piste d'audit.
func (o *Org) Journal(event string, payload any) error {
	f, err := os.OpenFile(filepath.Join(o.Path, JournalFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("workspace: journal: %w", err)
	}
	defer func() { _ = f.Close() }()
	line, err := json.Marshal(map[string]any{
		"at":    time.Now().Format(time.RFC3339),
		"event": event,
		"data":  payload,
	})
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

// lock pose un verrou d'exclusion sur le registre (fichier créé en O_EXCL,
// portable partout). Un verrou plus vieux que staleLock est considéré comme
// l'héritage d'un processus mort et est repris.
const (
	lockRetry = 50 * time.Millisecond
	lockWait  = 5 * time.Second
	staleLock = 30 * time.Second
)

func (o *Org) lock() (func(), error) {
	path := filepath.Join(o.Path, RegistryFile+".lock")
	deadline := time.Now().Add(lockWait)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(f, "pid=%d\n", os.Getpid())
			_ = f.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if fi, statErr := os.Stat(path); statErr == nil && time.Since(fi.ModTime()) > staleLock {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("workspace: registre verrouillé (%s) — une autre attribution est en cours ?", path)
		}
		time.Sleep(lockRetry)
	}
}

// registry manipule numerotation.json en préservant tout ce qu'on ne gère pas :
// le fichier appartient à l'utilisateur (mentions _doc, convention, champs
// libres des entrées…) et une mise à jour ne doit jamais en perdre une clé.
type registry struct {
	top      map[string]json.RawMessage
	counters map[string]int
	invoices []json.RawMessage
}

func (o *Org) readRegistry() (*registry, error) {
	raw, err := os.ReadFile(filepath.Join(o.Path, RegistryFile))
	if err != nil {
		return nil, fmt.Errorf("workspace: lecture du registre: %w", err)
	}
	r := &registry{
		top:      map[string]json.RawMessage{},
		counters: map[string]int{},
	}
	if err := json.Unmarshal(raw, &r.top); err != nil {
		return nil, fmt.Errorf("workspace: registre illisible: %w", err)
	}
	if c, ok := r.top["compteurs"]; ok {
		if err := json.Unmarshal(c, &r.counters); err != nil {
			return nil, fmt.Errorf("workspace: compteurs illisibles: %w", err)
		}
	}
	if f, ok := r.top["factures"]; ok {
		if err := json.Unmarshal(f, &r.invoices); err != nil {
			return nil, fmt.Errorf("workspace: liste des factures illisible: %w", err)
		}
	}
	return r, nil
}

func (r *registry) append(e RegistryEntry) error {
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	r.invoices = append(r.invoices, raw)
	return nil
}

// writeRegistry ré-assemble et écrit le registre de façon atomique (fichier
// temporaire puis renommage) : une coupure ne laisse jamais un registre tronqué.
func (o *Org) writeRegistry(r *registry) error {
	var err error
	if r.top["compteurs"], err = json.Marshal(r.counters); err != nil {
		return err
	}
	if r.top["factures"], err = json.Marshal(r.invoices); err != nil {
		return err
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(orderedTop(r.top)); err != nil {
		return err
	}

	path := filepath.Join(o.Path, RegistryFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// orderedTop impose un ordre stable et lisible aux clés de premier niveau :
// documentation d'abord, compteurs, puis la liste des factures, puis le reste.
func orderedTop(top map[string]json.RawMessage) json.RawMessage {
	known := []string{"_doc", "convention", "compteurs", "factures"}
	var buf bytes.Buffer
	buf.WriteString("{")
	first := true
	write := func(k string, v json.RawMessage) {
		if !first {
			buf.WriteString(",")
		}
		first = false
		key, _ := json.Marshal(k)
		buf.Write(key)
		buf.WriteString(":")
		buf.Write(v)
	}
	for _, k := range known {
		if v, ok := top[k]; ok {
			write(k, v)
		}
	}
	rest := make([]string, 0, len(top))
	for k := range top {
		if !slices.Contains(known, k) {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		write(k, top[k])
	}
	buf.WriteString("}")
	return buf.Bytes()
}

// CounterFor renvoie l'état du compteur d'une année (0 si vierge).
func (o *Org) CounterFor(year int) (int, error) {
	reg, err := o.readRegistry()
	if err != nil {
		return 0, err
	}
	return reg.counters[strconv.Itoa(year)], nil
}

// InvoiceCount renvoie le nombre de factures inscrites au registre.
func (o *Org) InvoiceCount() (int, error) {
	reg, err := o.readRegistry()
	if err != nil {
		return 0, err
	}
	return len(reg.invoices), nil
}
