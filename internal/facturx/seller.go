package facturx

import (
	"os"
	"strings"
)

// Vendeur et coordonnées de règlement : lus dans l'environnement, jamais codés
// en dur. Un fichier .env (non versionné) est le support habituel — voir
// .env.example. Sans ces variables, le JSON de facture doit porter un bloc
// "seller" complet, faute de quoi la génération échoue.
const (
	envSellerName       = "GOFACT_SELLER_NAME"
	envSellerSIREN      = "GOFACT_SELLER_SIREN"
	envSellerSIRET      = "GOFACT_SELLER_SIRET"
	envSellerVATNumber  = "GOFACT_SELLER_VAT_NUMBER"
	envSellerEmail      = "GOFACT_SELLER_EMAIL"
	envSellerAddress    = "GOFACT_SELLER_ADDRESS"
	envSellerPostalCode = "GOFACT_SELLER_POSTAL_CODE"
	envSellerCity       = "GOFACT_SELLER_CITY"
	envSellerCountry    = "GOFACT_SELLER_COUNTRY"
	envSellerEAddr      = "GOFACT_SELLER_ELECTRONIC_ADDRESS"
	envSellerEAddrSch   = "GOFACT_SELLER_ELECTRONIC_ADDRESS_SCHEME"

	envPayeeIBAN = "GOFACT_PAYEE_IBAN"

	envVATExemptMention = "GOFACT_VAT_EXEMPTION_MENTION"
	envVATExemptCode    = "GOFACT_VAT_EXEMPTION_CODE"
)

// Régime de TVA par défaut : franchise en base (art. 293 B du CGI), le cas le
// plus courant pour un indépendant français. Ce sont des mentions légales
// génériques, surchargeables par l'environnement ou par le JSON.
const (
	fallbackVATMention = "TVA non applicable, art. 293 B du CGI"
	fallbackVATExCode  = "VATEX-FR-FRANCHISE"
)

// defaultEAddrScheme est le schéma d'adresse électronique de routage utilisé
// quand le vendeur n'en déclare pas : 0225 = SIREN français (Peppol).
const defaultEAddrScheme = "0225"

func env(key string) string { return strings.TrimSpace(os.Getenv(key)) }

func envOr(key, fallback string) string {
	if v := env(key); v != "" {
		return v
	}
	return fallback
}

// sellerFromEnv construit le vendeur par défaut depuis l'environnement. Le
// résultat a un Name vide si aucune identité n'est configurée.
func sellerFromEnv() PartySpec {
	p := PartySpec{
		Name:        env(envSellerName),
		SIREN:       env(envSellerSIREN),
		SIRET:       env(envSellerSIRET),
		VATNumber:   env(envSellerVATNumber),
		Email:       env(envSellerEmail),
		Address:     env(envSellerAddress),
		PostalCode:  env(envSellerPostalCode),
		City:        env(envSellerCity),
		Country:     envOr(envSellerCountry, "FR"),
		EAddr:       env(envSellerEAddr),
		EAddrSchema: env(envSellerEAddrSch),
	}
	// Sans adresse de routage explicite, une PDP française route sur le SIREN
	// (BT-34, scheme 0225) plutôt que sur l'e-mail, qui n'est pas adressable.
	if p.EAddr == "" {
		if siren := siren9(p.SIREN, p.SIRET); siren != "" {
			p.EAddr = siren
			if p.EAddrSchema == "" {
				p.EAddrSchema = defaultEAddrScheme
			}
		}
	}
	return p
}

// defaultIBAN renvoie l'IBAN de règlement (BT-84) configuré, ou "" si aucun.
func defaultIBAN() string {
	return strings.ReplaceAll(env(envPayeeIBAN), " ", "")
}
