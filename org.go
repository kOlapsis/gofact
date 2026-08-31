package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kolapsis/gofact/internal/workspace"
)

// Sous-commande org : gestion des dossiers d'organisation depuis le CLI.
// Le serveur MCP s'appuiera sur les mêmes primitives (internal/workspace).
func runOrg(argv []string) {
	if len(argv) == 0 {
		orgUsage()
		os.Exit(2)
	}
	switch argv[0] {
	case "list":
		runOrgList(argv[1:])
	case "init":
		runOrgInit(argv[1:])
	case "show":
		runOrgShow(argv[1:])
	default:
		fmt.Fprintf(os.Stderr, "erreur : sous-commande org inconnue %q\n", argv[0])
		orgUsage()
		os.Exit(2)
	}
}

func orgUsage() {
	fmt.Fprintln(os.Stderr, `usage :
  gofact org list                       # organisations découvertes
  gofact org init -path DIR [identité]  # crée un dossier d'organisation
  gofact org show [-org DIR]            # fiche d'une organisation (sans secrets)`)
}

func runOrgList(argv []string) {
	fs := flag.NewFlagSet("gofact org list", flag.ExitOnError)
	_ = fs.Parse(argv)

	orgs, err := workspace.Discover("")
	if err != nil {
		fail(err)
	}
	if len(orgs) == 0 {
		fmt.Println("Aucune organisation. Créez-en une : gofact org init -path <dossier> -name \"Mon activité\"")
		return
	}
	for _, o := range orgs {
		id := o.Identity()
		count, _ := o.InvoiceCount()
		next, _ := o.NextNumber(time.Now())
		fmt.Printf("%-30s %s\n", id.Name, o.Path)
		fmt.Printf("  %d facture(s) · prochain numéro %s · IBAN %s · PDP %s\n",
			count, next, yesNo(id.HasIBAN), yesNo(id.HasPDP))
	}
}

func runOrgInit(argv []string) {
	fs := flag.NewFlagSet("gofact org init", flag.ExitOnError)
	path := fs.String("path", "", "dossier à créer (requis)")
	name := fs.String("name", "", "nom de l'entité émettrice")
	siret := fs.String("siret", "", "SIRET (14 chiffres)")
	siren := fs.String("siren", "", "SIREN (9 chiffres)")
	vat := fs.String("vat", "", "n° TVA intracommunautaire (vide si franchise 293 B)")
	email := fs.String("email", "", "e-mail")
	address := fs.String("address", "", "adresse (rue)")
	cp := fs.String("postal-code", "", "code postal")
	city := fs.String("city", "", "ville")
	iban := fs.String("iban", "", "IBAN de règlement")
	_ = fs.Parse(argv)

	if *path == "" {
		fmt.Fprintln(os.Stderr, "erreur : -path est requis")
		fs.Usage()
		os.Exit(2)
	}
	org, err := workspace.Init(*path, map[string]string{
		"GOFACT_SELLER_NAME":        *name,
		"GOFACT_SELLER_SIRET":       *siret,
		"GOFACT_SELLER_SIREN":       *siren,
		"GOFACT_SELLER_VAT_NUMBER":  *vat,
		"GOFACT_SELLER_EMAIL":       *email,
		"GOFACT_SELLER_ADDRESS":     *address,
		"GOFACT_SELLER_POSTAL_CODE": *cp,
		"GOFACT_SELLER_CITY":        *city,
		"GOFACT_PAYEE_IBAN":         *iban,
	})
	if err != nil {
		fail(err)
	}
	fmt.Printf("✓ Organisation « %s » créée : %s\n", org.Name(), org.Path)
	fmt.Println("  Registre de numérotation initialisé ; identité dans .env (à compléter au besoin).")
}

func runOrgShow(argv []string) {
	fs := flag.NewFlagSet("gofact org show", flag.ExitOnError)
	dir := fs.String("org", "", "dossier de l'organisation (défaut : découverte)")
	_ = fs.Parse(argv)

	orgs, err := workspace.Discover(*dir)
	if err != nil {
		fail(err)
	}
	if len(orgs) == 0 {
		fail(fmt.Errorf("aucune organisation trouvée"))
	}
	o := orgs[0]
	id := o.Identity()
	count, _ := o.InvoiceCount()
	next, _ := o.NextNumber(time.Now())
	tpl, _ := o.Template()
	clients, _ := o.Clients()

	fmt.Printf("Organisation : %s\n", id.Name)
	fmt.Printf("Dossier      : %s\n", o.Path)
	if id.SIRET != "" {
		fmt.Printf("SIRET        : %s\n", id.SIRET)
	} else if id.SIREN != "" {
		fmt.Printf("SIREN        : %s\n", id.SIREN)
	}
	if id.Email != "" {
		fmt.Printf("E-mail       : %s\n", id.Email)
	}
	if id.Address != "" {
		fmt.Printf("Adresse      : %s, %s %s\n", id.Address, id.PostalCode, id.City)
	}
	fmt.Printf("IBAN         : %s · PDP : %s\n", yesNo(id.HasIBAN), yesNo(id.HasPDP))
	fmt.Printf("Factures     : %d · prochain numéro : %s\n", count, next)
	fmt.Printf("Modèle figé  : %s · clients connus : %d\n", yesNo(tpl != ""), len(clients))
}

func yesNo(b bool) string {
	if b {
		return "oui"
	}
	return "non"
}
