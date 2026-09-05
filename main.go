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
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/kolapsis/gofact/internal/dotenv"
	"github.com/kolapsis/gofact/internal/facturx"
	"github.com/kolapsis/gofact/internal/pdp"
	_ "github.com/kolapsis/gofact/internal/pdp/superpdp"
	"github.com/kolapsis/gofact/internal/workspace"
)

func main() {
	if len(os.Args) == 1 {
		usage(os.Stdout)
		return
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "help", "-h", "--help":
			usage(os.Stdout)
			return
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
		case "version":
			fmt.Println("gofact", version)
			return
		}
	}
	runGenerate(os.Args[1:])
}

// usage décrit le binaire par ce qu'il sait faire, pas par ses drapeaux. La
// commande la plus utilisée est `mcp`, lancée par le client IA : la sortie par
// défaut doit la nommer, sans quoi l'utilisateur qui tape « gofact » pour voir
// ne trouve que le mode direct.
func usage(w io.Writer) {
	fmt.Fprint(w, `gofact — factures Factur-X (PDF/A-3 + XML CII EN 16931), en local.

Commandes :
  gofact install               déclare le serveur MCP auprès des clients IA du poste
  gofact mcp                   démarre le serveur MCP (stdio) — lancé par votre client IA
  gofact org init|list|show|set-counter
                               gère les dossiers d'organisation et la numérotation
  gofact send -pdf <fichier>   dépose un Factur-X existant sur votre plateforme agréée
  gofact version               affiche la version

Mode direct, sans IA :
  gofact -html <facture.html>  génère le Factur-X à partir du HTML et de son JSON
                               (ne consomme aucun numéro, n'inscrit rien au registre)
                               (`+"`gofact -html x.html -h`"+` pour les options)

Pour commencer : `+"`gofact install`"+`, puis demandez une facture à votre IA.
Documentation : https://gofact.kolapsis.com/

gofact n'est pas une plateforme agréée : il produit le fichier, votre
plateforme agréée le transporte.
`)
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
		if !strings.HasPrefix(argv[0], "-") {
			fmt.Fprintf(os.Stderr, "erreur : %q n'est pas une commande gofact.\n\n", argv[0])
		} else {
			fmt.Fprint(os.Stderr, "erreur : -html est requis en mode direct.\n\n")
		}
		usage(os.Stderr)
		os.Exit(2)
	}

	if err := guardRegistry(*htmlPath); err != nil {
		fail(err)
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
// guardRegistry empêche le mode direct d'émettre une facture fantôme.
//
// `-html` est un convertisseur : il rend le HTML qu'on lui donne, sans attribuer
// de numéro ni rien inscrire au registre. Hors d'un dossier d'organisation c'est
// exactement ce qu'on veut. Dedans, c'est un piège : deux appels de suite
// produisent deux factures portant le même numéro, et le registre — qui se
// présente comme la source de vérité d'une séquence continue, sans trou, jamais
// réutilisée — les ignore toutes les deux. On refuse donc, sauf s'il s'agit du
// re-rendu d'une facture déjà inscrite : là, le numéro existe déjà.
func guardRegistry(htmlPath string) error {
	o, err := workspace.Open(filepath.Dir(htmlPath))
	if err != nil {
		return nil // pas un dossier d'organisation : rien à garantir
	}
	invoices, err := o.Invoices()
	if err != nil {
		return err
	}
	base := filepath.Base(htmlPath)
	for _, inv := range invoices {
		if f, _ := inv["fichier"].(string); f == base {
			return nil // re-rendu d'une facture déjà émise
		}
	}
	return fmt.Errorf("%s est un dossier d'organisation, et %s n'est inscrit à aucune facture de %s.\n\n"+
		"Le mode direct rend le HTML tel quel : il n'attribue pas de numéro et n'inscrit rien au "+
		"registre. L'utiliser ici produirait une facture que le registre ignore — et le prochain "+
		"appel réutiliserait le même numéro.\n\n"+
		"Pour émettre une facture, passez par votre IA (outil create_invoice), qui attribue le "+
		"numéro et inscrit la facture en une transaction.",
		o.Path, base, workspace.RegistryFile)
}

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

	// Un rejet arrive en moins d'une seconde : le lire tout de suite, plutôt que
	// de laisser « Déposée » passer pour un succès.
	time.Sleep(time.Second)
	events, err := provider.Status(ctx, receipt.Reference)
	if err != nil {
		if poll {
			fail(err)
		}
		return
	}
	if rejected, reasons := pdp.Rejection(events); rejected {
		fmt.Fprintln(os.Stderr, "✗ Facture REJETÉE par la plateforme :")
		for _, r := range reasons {
			fmt.Fprintln(os.Stderr, "  -", r)
		}
		os.Exit(1)
	}
	if poll {
		fmt.Println("— cycle de vie —")
		printEvents(events)
	}
}

func printEvents(events []pdp.Event) {
	for _, e := range events {
		fmt.Printf("  %s  %-16s %s\n", e.CreatedAt, e.StatusCode, e.StatusText)
		for _, r := range e.Reasons {
			fmt.Printf("  %s  %-16s   - %s\n", strings.Repeat(" ", len(e.CreatedAt)), "", r)
		}
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
