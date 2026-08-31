package facturx

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Assemblage Factur-X en Go pur : on part du PDF rendu par Chrome et on y ajoute
// ce qui manque pour en faire un PDF/A-3b portant la facture. Aucun binaire
// externe — c'est ce qui permet de distribuer gofact sans rien demander à
// installer (voir README, « Dépendances »).
//
// Les quatre ajouts, mesurés comme suffisants contre veraPDF (via Mustang) :
//
//  1. le XML CII en fichier embarqué, inséré octet pour octet ;
//  2. un OutputIntent sRGB, sans lequel chaque usage de DeviceRGB est une
//     violation (à lui seul il règle l'essentiel des non-conformités) ;
//  3. un paquet XMP identifiant PDF/A-3b et déclarant l'extension Factur-X ;
//  4. un /ID dans le trailer, que Chrome n'écrit pas.
//
// Le PDF produit par Chrome n'est pas retouché autrement : ni reconversion
// colorimétrique, ni ré-encodage des polices. Les mesures montrent que Skia
// n'émet, pour une facture, que des constructions déjà admises en PDF/A-3.
const (
	facturxXMLName  = "factur-x.xml"
	facturxVersion  = "1.0"
	facturxDocType  = "INVOICE"
	facturxNS       = "urn:factur-x:pdfa:CrossIndustryDocument:invoice:1p0#"
	facturxRelation = "Alternative"
)

// embedFacturX produit le PDF/A-3 Factur-X final à partir du PDF rendu et du
// XML CII. Le XML est embarqué sans re-sérialisation : les champs étendus
// (adresses de routage PDP, BT-34/BT-49) sont préservés tels quels.
func embedFacturX(inPDF, xmlPath, out string, issue time.Time) error {
	xml, err := os.ReadFile(xmlPath)
	if err != nil {
		return fmt.Errorf("facturx: lecture XML: %w", err)
	}

	conf := model.NewDefaultConfiguration()
	// Le PDF de Chrome est valide mais pas pointilleux ; on ne veut pas qu'une
	// broutille de forme fasse échouer l'assemblage.
	conf.ValidationMode = model.ValidationRelaxed

	ctx, err := api.ReadContextFile(inPDF)
	if err != nil {
		return fmt.Errorf("facturx: lecture du PDF rendu: %w", err)
	}
	ctx.Configuration = conf
	xt := ctx.XRefTable

	root, err := xt.Catalog()
	if err != nil {
		return fmt.Errorf("facturx: catalogue PDF: %w", err)
	}

	if err := attachXML(xt, root, xml, issue); err != nil {
		return err
	}
	if err := addOutputIntent(xt, root); err != nil {
		return err
	}
	if err := addXMP(xt, root, docTitle(xt), issue); err != nil {
		return err
	}
	// PDF/A exige un identifiant de fichier ; Chrome n'en écrit pas.
	if xt.ID == nil {
		id := types.HexLiteral(fmt.Sprintf("%032X", issue.UnixNano()))
		xt.ID = types.Array{id, id}
	}

	if err := api.WriteContextFile(ctx, out); err != nil {
		return fmt.Errorf("facturx: écriture du Factur-X: %w", err)
	}
	return nil
}

// attachXML embarque le XML CII (arbre /EmbeddedFiles) et le déclare comme
// fichier associé du document (/AF), relation « Alternative » — la valeur
// qu'écrit Ghostscript et qu'attendent les validateurs allemands.
func attachXML(xt *model.XRefTable, root types.Dict, xml []byte, issue time.Time) error {
	sd, err := newStreamDict(xml, true)
	if err != nil {
		return fmt.Errorf("facturx: flux XML: %w", err)
	}
	sd.InsertName("Type", "EmbeddedFile")
	sd.InsertName("Subtype", "text/xml")
	params := types.NewDict()
	params.InsertInt("Size", len(xml))
	params.InsertString("ModDate", types.DateString(issue))
	sd.Insert("Params", params)

	sdRef, err := xt.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	spec, err := xt.NewFileSpecDict(facturxXMLName, facturxXMLName, "Factur-X Invoice", *sdRef)
	if err != nil {
		return fmt.Errorf("facturx: descripteur de fichier: %w", err)
	}
	spec.InsertName("AFRelationship", facturxRelation)
	specRef, err := xt.IndRefForNewObject(spec)
	if err != nil {
		return err
	}

	root.Insert("AF", types.Array{*specRef})

	namesRef, err := xt.IndRefForNewObject(types.Dict{
		"Names": types.Array{types.StringLiteral(facturxXMLName), *specRef},
	})
	if err != nil {
		return err
	}
	embeddedRef, err := xt.IndRefForNewObject(types.Dict{"EmbeddedFiles": *namesRef})
	if err != nil {
		return err
	}
	root.Insert("Names", *embeddedRef)
	return nil
}

// addOutputIntent déclare le profil sRGB embarqué dans le binaire. Sans lui,
// tout usage de DeviceRGB est une violation PDF/A. La clé /S vaut GTS_PDFA1
// y compris en PDF/A-3 : c'est la valeur prévue par la norme, pas une coquille.
func addOutputIntent(xt *model.XRefTable, root types.Dict) error {
	sd, err := newStreamDict(srgbICC, true)
	if err != nil {
		return fmt.Errorf("facturx: profil ICC: %w", err)
	}
	sd.InsertInt("N", 3)
	iccRef, err := xt.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	oi := types.NewDict()
	oi.InsertName("Type", "OutputIntent")
	oi.InsertName("S", "GTS_PDFA1")
	oi.InsertString("OutputConditionIdentifier", "sRGB")
	oi.InsertString("Info", "sRGB IEC61966-2.1")
	oi.Insert("DestOutputProfile", *iccRef)

	oiRef, err := xt.IndRefForNewObject(oi)
	if err != nil {
		return err
	}
	root.Insert("OutputIntents", types.Array{*oiRef})
	return nil
}

// addXMP écrit le paquet de métadonnées. Il porte trois choses indissociables :
// l'identification PDF/A-3b, le titre (qui doit correspondre au /Title du
// dictionnaire Info, sous peine de non-conformité) et le schéma d'extension
// Factur-X, obligatoire dès qu'on emploie un espace de noms hors norme PDF/A.
func addXMP(xt *model.XRefTable, root types.Dict, title string, issue time.Time) error {
	// Le paquet XMP ne doit pas être filtré : un validateur doit pouvoir le lire
	// sans décompresser quoi que ce soit.
	sd, err := newStreamDict([]byte(buildXMP(title, issue)), false)
	if err != nil {
		return fmt.Errorf("facturx: XMP: %w", err)
	}
	sd.InsertName("Type", "Metadata")
	sd.InsertName("Subtype", "XML")

	ref, err := xt.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}
	root.Insert("Metadata", *ref)
	return nil
}

// docTitle lit le /Title du dictionnaire Info, que Chrome renseigne depuis le
// <title> du HTML. Le XMP doit le reprendre à l'identique.
func docTitle(xt *model.XRefTable) string {
	if xt.Info == nil {
		return ""
	}
	d, err := xt.DereferenceDict(*xt.Info)
	if err != nil || d == nil {
		return ""
	}
	s, err := xt.DereferenceText(d["Title"])
	if err != nil {
		return ""
	}
	return s
}

// newStreamDict construit un flux, compressé ou non.
func newStreamDict(raw []byte, compress bool) (*types.StreamDict, error) {
	sd := types.StreamDict{Dict: types.NewDict(), Content: raw, Raw: raw}
	if !compress {
		n := int64(len(raw))
		sd.StreamLength = &n
		sd.InsertInt("Length", len(raw))
		return &sd, nil
	}
	sd.InsertName("Filter", "FlateDecode")
	sd.FilterPipeline = []types.PDFFilter{{Name: "FlateDecode"}}
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	return &sd, nil
}

func buildXMP(title string, issue time.Time) string {
	ts := issue.Format("2006-01-02T15:04:05-07:00")
	var b strings.Builder
	b.WriteString(`<?xpacket begin="` + "\uFEFF" + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about="" xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/">
   <pdfaid:part>3</pdfaid:part>
   <pdfaid:conformance>B</pdfaid:conformance>
  </rdf:Description>
  <rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">
   <dc:format>application/pdf</dc:format>`)
	if title != "" {
		b.WriteString(`
   <dc:title><rdf:Alt><rdf:li xml:lang="x-default">` + escapeXML(title) + `</rdf:li></rdf:Alt></dc:title>`)
	}
	b.WriteString(`
  </rdf:Description>
  <rdf:Description rdf:about="" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
   <xmp:CreateDate>` + ts + `</xmp:CreateDate>
   <xmp:ModifyDate>` + ts + `</xmp:ModifyDate>
  </rdf:Description>
  <rdf:Description rdf:about="" xmlns:fx="` + facturxNS + `">
   <fx:DocumentType>` + facturxDocType + `</fx:DocumentType>
   <fx:DocumentFileName>` + facturxXMLName + `</fx:DocumentFileName>
   <fx:Version>` + facturxVersion + `</fx:Version>
   <fx:ConformanceLevel>` + facturxConformance + `</fx:ConformanceLevel>
  </rdf:Description>
  <rdf:Description rdf:about=""
    xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/"
    xmlns:pdfaSchema="http://www.aiim.org/pdfa/ns/schema#"
    xmlns:pdfaProperty="http://www.aiim.org/pdfa/ns/property#">
   <pdfaExtension:schemas>
    <rdf:Bag>
     <rdf:li rdf:parseType="Resource">
      <pdfaSchema:schema>Factur-X PDFA Extension Schema</pdfaSchema:schema>
      <pdfaSchema:namespaceURI>` + facturxNS + `</pdfaSchema:namespaceURI>
      <pdfaSchema:prefix>fx</pdfaSchema:prefix>
      <pdfaSchema:property>
       <rdf:Seq>`)
	for _, p := range [][2]string{
		{"DocumentFileName", "name of the embedded XML invoice file"},
		{"DocumentType", "INVOICE"},
		{"Version", "The actual version of the Factur-X data"},
		{"ConformanceLevel", "The conformance level of the embedded data"},
	} {
		b.WriteString(`
        <rdf:li rdf:parseType="Resource">
         <pdfaProperty:name>` + p[0] + `</pdfaProperty:name>
         <pdfaProperty:valueType>Text</pdfaProperty:valueType>
         <pdfaProperty:category>external</pdfaProperty:category>
         <pdfaProperty:description>` + escapeXML(p[1]) + `</pdfaProperty:description>
        </rdf:li>`)
	}
	b.WriteString(`
       </rdf:Seq>
      </pdfaSchema:property>
     </rdf:li>
    </rdf:Bag>
   </pdfaExtension:schemas>
  </rdf:Description>
 </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`)
	return b.String()
}

func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
