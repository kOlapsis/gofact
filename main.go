// Command gofact assemble et envoie des factures Factur-X (PDF/A-3 + XML CII
// EN 16931).
//
// Génération : règles EN 16931 → XML CII → PDF (Chrome) → PDF/A-3 avec
// embarquement du XML verbatim, en Go pur → auto-contrôle du résultat.
// Seule dépendance externe : un navigateur Chrome pour le rendu.
//
// Envoi : dépôt du PDF Factur-X sur la PDP SuperPDP (OAuth2 client credentials).
//
// Configuration : identité du vendeur, IBAN de règlement et identifiants PDP sont
// lus dans l'environnement (variables GOFACT_* et SUPERPDP_*), directement ou via
// un fichier .env non versionné (cf. .env.example). Rien n'est codé en dur.
//
// Usage :
//
//	gofact -html "2026011 - Client.html"        # génère le PDF Factur-X
//	gofact -html f.html -send                   # génère puis dépose sur la PDP
//	gofact send -pdf f.pdf -env ~/.gofact.env   # dépose un PDF existant
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/kolapsis/gofact/internal/dotenv"
	"github.com/kolapsis/gofact/internal/facturx"
	"github.com/kolapsis/gofact/internal/pdp"
	_ "github.com/kolapsis/gofact/internal/pdp/superpdp"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "send":
			runSend(os.Args[2:])
			return
		case "org":
			runOrg(os.Args[2:])
			return
		case "mcp":
			runMCP(os.Args[2:])
			return
		case "install":
			runInstall(os.Args[2:])
			return
		}
	}
	runGenerate(os.Args[1:])
}

// runGenerate génère (et, si -send, dépose) un Factur-X à partir d'un HTML + JSON.
func runGenerate(argv []string) {
	fs := flag.NewFlagSet("gofact", flag.ExitOnError)
	htmlPath := fs.String("html", "", "facture HTML prête à imprimer (requis)")
	dataPath := fs.String("data", "", "spécification JSON (défaut : même nom que -html en .json)")
	outPath := fs.String("out", "", "PDF Factur-X de sortie (défaut : même nom que -html en .pdf)")
	xmlPath := fs.String("xml", "", "écrit aussi le XML CII à ce chemin (debug)")
	chromePath := fs.String("chrome", "", "exécutable Chrome (défaut : auto-détection)")
	validate := fs.Bool("validate", true, "relit et vérifie le Factur-X produit")
	send := fs.Bool("send", false, "dépose le PDF sur SuperPDP après génération")
	envPath := fs.String("env", "", "fichier .env (vendeur, IBAN, identifiants PDP) ; défaut ./.env puis ~/.config/gofact/.env")
	poll := fs.Bool("poll", false, "après envoi, récupère les statuts du cycle de vie")
	quiet := fs.Bool("q", false, "silencieux (n'affiche que les erreurs)")
	_ = fs.Parse(argv)

	if *htmlPath == "" {
		fmt.Fprintln(os.Stderr, "erreur : -html est requis")
		fs.Usage()
		os.Exit(2)
	}

	if err := dotenv.LoadDefault(*envPath); err != nil {
		fail(err)
	}

	data := *dataPath
	if data == "" {
		data = swapExt(*htmlPath, ".json")
	}
	out := *outPath
	if out == "" {
		out = swapExt(*htmlPath, ".pdf")
	}

	spec, err := facturx.LoadSpec(data)
	if err != nil {
		fail(err)
	}
	inv, err := spec.ToInvoice()
	if err != nil {
		fail(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	res, err := facturx.Generate(ctx, inv, facturx.Options{
		HTMLPath:   *htmlPath,
		OutPath:    out,
		ChromePath: *chromePath,
		XMLOut:     *xmlPath,
		Validate:   *validate,
		Verbose:    !*quiet,
	})
	if err != nil {
		fail(err)
	}
	if res.Validated && !res.Valid {
		fmt.Fprintf(os.Stderr, "✗ Factur-X NON conforme : %s\n%s\n", res.OutPath, res.Report)
		os.Exit(1)
	}
	if res.Validated {
		fmt.Printf("✓ Factur-X conforme (EN 16931) : %s\n", res.OutPath)
	} else {
		fmt.Printf("✓ Factur-X généré : %s\n", res.OutPath)
	}

	if *send {
		sendPDF(ctx, out, *envPath, *poll)
	}
}

// runSend dépose un PDF Factur-X existant sur SuperPDP.
func runSend(argv []string) {
	fs := flag.NewFlagSet("gofact send", flag.ExitOnError)
	pdfPath := fs.String("pdf", "", "PDF Factur-X à déposer (requis)")
	envPath := fs.String("env", "", "fichier .env pour les identifiants PDP ; défaut ./.env puis ~/.config/gofact/.env")
	poll := fs.Bool("poll", false, "après envoi, récupère les statuts du cycle de vie")
	_ = fs.Parse(argv)

	if *pdfPath == "" {
		fmt.Fprintln(os.Stderr, "erreur : -pdf est requis")
		fs.Usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	sendPDF(ctx, *pdfPath, *envPath, *poll)
}

// sendPDF dépose le PDF sur la PDP configurée, en affichant le résultat.
func sendPDF(ctx context.Context, pdf, envPath string, poll bool) {
	if err := dotenv.LoadDefault(envPath); err != nil {
		fail(err)
	}
	provider, err := pdp.Open(os.Getenv)
	if err != nil {
		fail(err)
	}
	receipt, err := provider.Send(ctx, pdf)
	if err != nil {
		fail(err)
	}
	fmt.Printf("✓ Déposée sur %s — référence %s\n", receipt.Provider, receipt.Reference)
	printEvents(receipt.Events)

	if poll {
		events, err := provider.Status(ctx, receipt.Reference)
		if err != nil {
			fail(err)
		}
		fmt.Println("— cycle de vie —")
		printEvents(events)
	}
}

func printEvents(events []pdp.Event) {
	for _, e := range events {
		fmt.Printf("  %s  %-16s %s\n", e.CreatedAt, e.StatusCode, e.StatusText)
	}
}

func swapExt(path, ext string) string {
	if i := strings.LastIndex(path, "."); i >= 0 && !strings.ContainsAny(path[i:], "/\\") {
		return path[:i] + ext
	}
	return path + ext
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "erreur :", err)
	os.Exit(1)
}
