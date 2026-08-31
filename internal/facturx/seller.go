package facturx

import (
	"os"
	"strings"
)

// Vendeur et coordonnées de règlement : jamais codés en dur. Ils viennent d'une
// Config, résolue soit depuis l'environnement du processus (cas du CLI), soit
// depuis n'importe quelle autre source de clés — typiquement le .env d'un
// dossier d'organisation, quand un même processus sert plusieurs entités
// émettrices. Sans configuration, le JSON de facture doit porter un bloc
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
// génériques, surchargeables par la configuration ou par le JSON.
const (
	fallbackVATMention = "TVA non applicable, art. 293 B du CGI"
	fallbackVATExCode  = "VATEX-FR-FRANCHISE"
)

// defaultEAddrScheme est le schéma d'adresse électronique de routage utilisé
// quand le vendeur n'en déclare pas : 0225 = SIREN français (réforme FR).
const defaultEAddrScheme = "0225"

// Config porte les défauts d'une entité émettrice : identité du vendeur, compte
// de règlement et mentions d'exonération. Le JSON de chaque facture peut
// surcharger n'importe lequel de ces champs.
type Config struct {
	Seller              PartySpec
	IBAN                string // BT-84 ; vide ⇒ omis (mais BR-50 exige un compte pour un virement)
	VATExemptMention    string // BT-120 par défaut si exonéré
	VATExemptReasonCode string // BT-121 par défaut si exonéré
}

// ConfigFrom résout une Config depuis une source de clés arbitraire. lookup
// renvoie "" pour une clé absente.
func ConfigFrom(lookup func(key string) string) Config {
	get := func(key string) string { return strings.TrimSpace(lookup(key)) }
	getOr := func(key, fallback string) string {
		if v := get(key); v != "" {
			return v
		}
		return fallback
	}

	p := PartySpec{
		Name:        get(envSellerName),
		SIREN:       get(envSellerSIREN),
		SIRET:       get(envSellerSIRET),
		VATNumber:   get(envSellerVATNumber),
		Email:       get(envSellerEmail),
		Address:     get(envSellerAddress),
		PostalCode:  get(envSellerPostalCode),
		City:        get(envSellerCity),
		Country:     getOr(envSellerCountry, "FR"),
		EAddr:       get(envSellerEAddr),
		EAddrSchema: get(envSellerEAddrSch),
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

	return Config{
		Seller:              p,
		IBAN:                strings.ReplaceAll(get(envPayeeIBAN), " ", ""),
		VATExemptMention:    getOr(envVATExemptMention, fallbackVATMention),
		VATExemptReasonCode: getOr(envVATExemptCode, fallbackVATExCode),
	}
}

// ConfigFromEnv résout la Config depuis l'environnement du processus — le
// comportement historique du CLI.
func ConfigFromEnv() Config {
	return ConfigFrom(os.Getenv)
}
