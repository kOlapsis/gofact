package workspace

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kolapsis/gofact/internal/facturx"
)

// Modèle de facture de l'organisation. Le HTML est généré par une IA — c'est le
// choix du produit — mais une facture est un document comptable archivé dix
// ans : deux factures de la même entité ne doivent pas se ressembler « à peu
// près ». Le compromis : le dossier FIGE le HTML de la première facture comme
// modèle de référence, qui est resservi à l'IA pour chaque facture suivante.
// Un contrôle de dérive compare la STRUCTURE (pas le contenu, qui change à
// chaque facture) et produit un avertissement — jamais un blocage :
// l'utilisateur reste souverain sur son modèle.

// TemplateFile est le modèle de référence figé de l'organisation.
const TemplateFile = "modele-facture.html"

// Template renvoie le modèle figé, ou "" si aucune facture n'a encore fixé le
// modèle de ce dossier.
func (o *Org) Template() (string, error) {
	raw, err := os.ReadFile(filepath.Join(o.Path, TemplateFile))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ReplaceTemplate remplace délibérément le modèle de référence. La mise en page
// n'est pas gravée dans le marbre — c'est la numérotation qui l'est — mais un
// changement de modèle doit être un choix, pas un accident : d'où une méthode
// distincte de FreezeTemplate.
func (o *Org) ReplaceTemplate(html string) error {
	if err := os.WriteFile(filepath.Join(o.Path, TemplateFile), []byte(html), 0o644); err != nil {
		return err
	}
	return o.Journal("template_replaced", map[string]any{"fingerprint": Fingerprint(html)})
}

// FreezeTemplate fige html comme modèle de référence si le dossier n'en a pas
// encore. Ne remplace jamais un modèle existant par accident : le changement
// délibéré passe par ReplaceTemplate.
func (o *Org) FreezeTemplate(html string) (frozen bool, err error) {
	path := filepath.Join(o.Path, TemplateFile)
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// TemplateDrift compare la structure de html au modèle figé. Renvoie "" si le
// dossier n'a pas de modèle ou si la structure correspond, sinon un
// avertissement destiné à être relayé (par une IA notamment) à l'utilisateur.
func (o *Org) TemplateDrift(html string) (string, error) {
	ref, err := o.Template()
	if err != nil || ref == "" {
		return "", err
	}
	if Fingerprint(ref) == Fingerprint(html) {
		return "", nil
	}
	return fmt.Sprintf("la structure de ce HTML diffère du modèle de référence (%s). "+
		"Si c'est voulu — la mise en page peut évoluer librement — refaire l'appel avec "+
		"update_template pour en faire le nouveau modèle ; sinon, repartir du modèle.", TemplateFile), nil
}

var (
	classAttr = regexp.MustCompile(`class="([^"]*)"`)
	tagOpen   = regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9]*)[\s>]`)
	styleTag  = regexp.MustCompile(`(?s)<style[^>]*>(.*?)</style>`)
)

// Fingerprint résume la STRUCTURE d'un HTML de facture : l'ensemble des classes
// CSS, l'inventaire des balises et le contenu des feuilles de style — mais pas
// le texte, qui change légitimement d'une facture à l'autre.
func Fingerprint(html string) string {
	classes := map[string]bool{}
	for _, m := range classAttr.FindAllStringSubmatch(html, -1) {
		for _, c := range strings.Fields(m[1]) {
			classes[c] = true
		}
	}
	tags := map[string]int{}
	for _, m := range tagOpen.FindAllStringSubmatch(html, -1) {
		tags[strings.ToLower(m[1])]++
	}

	var parts []string
	for c := range classes {
		parts = append(parts, "c:"+c)
	}
	for t, n := range tags {
		parts = append(parts, fmt.Sprintf("t:%s=%d", t, n))
	}
	sort.Strings(parts)
	for _, m := range styleTag.FindAllStringSubmatch(html, -1) {
		parts = append(parts, "s:"+strings.Join(strings.Fields(m[1]), " "))
	}

	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

//go:embed modele-defaut.html
var defaultTemplateHTML string

// DefaultTemplate rend le modèle de facture livré avec gofact, renseigné de
// l'identité de l'organisation. Il existe pour qu'une première facture ne parte
// jamais d'une page blanche : les mentions légales françaises obligatoires y
// sont, quel que soit le modèle de langage qui compose la facture. Le jeton
// {{NUMERO}} y est préservé — c'est le serveur qui inscrit le numéro.
func (o *Org) DefaultTemplate() (string, error) {
	cfg := o.Config()
	seller := cfg.Seller
	if strings.TrimSpace(seller.Name) == "" {
		seller.Name = o.Name()
	}
	tpl, err := template.New("modele").Delims("[[", "]]").Parse(defaultTemplateHTML)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tpl.Execute(&buf, struct {
		Seller     facturx.PartySpec
		IBAN       string
		VATMention string
	}{Seller: seller, IBAN: formatIBAN(cfg.IBAN), VATMention: cfg.VATExemptMention})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// formatIBAN regroupe l'IBAN par quatre caractères — la forme imprimée usuelle.
func formatIBAN(iban string) string {
	iban = strings.ReplaceAll(iban, " ", "")
	if iban == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range iban {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
