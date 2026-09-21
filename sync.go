package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/kolapsis/gofact/internal/dotenv"
	"github.com/kolapsis/gofact/internal/notify"
	"github.com/kolapsis/gofact/internal/pdp"
	"github.com/kolapsis/gofact/internal/workspace"
)

// runSync récupère les factures reçues de chaque organisation et met à jour l'export comptable.
func runSync(argv []string) {
	fs := flag.NewFlagSet("gofact sync", flag.ExitOnError)
	dir := fs.String("org", "", "dossier de l'organisation (défaut : toutes celles découvertes)")
	envPath := fs.String("env", "", "fichier .env ; défaut ./.env puis ~/.config/gofact/.env")
	quiet := fs.Bool("q", false, "pas de notification sur le bureau")
	_ = fs.Parse(argv)

	if err := dotenv.LoadDefault(*envPath); err != nil {
		fail(err)
	}
	orgs, err := workspace.Discover(*dir)
	if err != nil {
		fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	failed := false
	for _, o := range orgs {
		if !o.Identity().HasPDP {
			if *dir != "" {
				fail(fmt.Errorf("%s n'a pas de compte PDP configuré", o.Name()))
			}
			continue
		}
		if err := syncOrg(ctx, o, !*quiet); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s : %v\n", o.Name(), err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func syncOrg(ctx context.Context, o *workspace.Org, notifyDesktop bool) error {
	provider, err := pdp.Open(o.Lookup)
	if err != nil {
		return err
	}
	fresh, err := o.SyncReceived(ctx, provider)
	for _, r := range fresh {
		fmt.Printf("✓ %s : facture reçue de %s, n° %s, %s %s → %s\n", o.Name(), r.Seller, r.Number, r.TotalTTC, r.Currency, r.File)
	}
	if notifyDesktop {
		switch {
		case len(fresh) == 1:
			r := fresh[0]
			notify.Send("gofact — facture reçue", fmt.Sprintf("%s · n° %s · %s %s", r.Seller, r.Number, r.TotalTTC, r.Currency))
		case len(fresh) > 1:
			notify.Send("gofact — factures reçues", fmt.Sprintf("%d nouvelles factures pour %s", len(fresh), o.Name()))
		}
	}
	if err != nil {
		return err
	}

	dest := o.Lookup(workspace.EnvExportDir)
	if dest == "" {
		return nil
	}
	now := time.Now()
	for _, month := range []string{now.AddDate(0, -1, 0).Format("2006-01"), now.Format("2006-01")} {
		res, err := o.Export(month, dest)
		if err != nil {
			return fmt.Errorf("export %s : %w", month, err)
		}
		if res.Copied > 0 {
			fmt.Printf("✓ %s : export %s à jour (%s)\n", o.Name(), month, res.Dir)
		}
	}
	return nil
}

// runExport écrit l'export comptable d'un mois.
func runExport(argv []string) {
	fs := flag.NewFlagSet("gofact export", flag.ExitOnError)
	dir := fs.String("org", "", "dossier de l'organisation (défaut : découverte)")
	month := fs.String("month", time.Now().Format("2006-01"), "mois à exporter, AAAA-MM")
	to := fs.String("to", "", "dossier de destination (défaut : GOFACT_EXPORT_DIR)")
	envPath := fs.String("env", "", "fichier .env ; défaut ./.env puis ~/.config/gofact/.env")
	_ = fs.Parse(argv)

	if err := dotenv.LoadDefault(*envPath); err != nil {
		fail(err)
	}
	o := singleOrg(*dir)
	dest := *to
	if dest == "" {
		dest = o.Lookup(workspace.EnvExportDir)
	}
	res, err := o.Export(*month, dest)
	if err != nil {
		fail(err)
	}
	fmt.Printf("✓ Export %s : %d émise(s), %d reçue(s) → %s\n", *month, res.Issued, res.Received, res.Dir)
	if len(res.Missing) > 0 {
		fmt.Fprintf(os.Stderr, "  PDF introuvable pour : %s\n", strings.Join(res.Missing, ", "))
	}
}

// runPaid signale à la PDP l'encaissement d'une facture émise.
func runPaid(argv []string) {
	fs := flag.NewFlagSet("gofact paid", flag.ExitOnError)
	dir := fs.String("org", "", "dossier de l'organisation (défaut : découverte)")
	number := fs.String("number", "", "numéro de la facture encaissée (requis)")
	date := fs.String("date", "", "date d'encaissement AAAA-MM-JJ (défaut : aujourd'hui)")
	amount := fs.String("amount", "", "montant encaissé, pour un paiement partiel (défaut : total)")
	yes := fs.Bool("yes", false, "ne pas demander de confirmation")
	envPath := fs.String("env", "", "fichier .env ; défaut ./.env puis ~/.config/gofact/.env")
	_ = fs.Parse(argv)

	if *number == "" {
		fmt.Fprintln(os.Stderr, "erreur : -number est requis")
		fs.Usage()
		os.Exit(2)
	}
	if err := dotenv.LoadDefault(*envPath); err != nil {
		fail(err)
	}
	o := singleOrg(*dir)
	if !*yes {
		what := "le total"
		if *amount != "" {
			what = *amount
		}
		fmt.Printf("Signaler à la PDP l'encaissement de la facture %s (%s) ? Il sera déclaré à l'administration. [o/N] ", *number, what)
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "o" && a != "oui" && a != "y" && a != "yes" {
			fmt.Println("Rien n'a été signalé.")
			return
		}
	}
	provider, err := pdp.Open(o.Lookup)
	if err != nil {
		fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	res, err := o.ReportPayment(ctx, provider, *number, *date, *amount)
	if errors.Is(err, workspace.ErrNotSent) {
		err = fmt.Errorf("%w : la déposer d'abord", err)
	}
	if err != nil {
		fail(err)
	}
	fmt.Printf("✓ Encaissement de la facture %s signalé (%s, %s)\n", res.Number, res.Date, res.Amount)
}

func singleOrg(dir string) *workspace.Org {
	orgs, err := workspace.Discover(dir)
	if err != nil {
		fail(err)
	}
	if len(orgs) != 1 {
		fail(fmt.Errorf("préciser l'organisation avec -org (trouvées : %d)", len(orgs)))
	}
	return orgs[0]
}
