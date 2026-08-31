package facturx

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Auto-contrôle du Factur-X produit. Ce n'est pas un validateur PDF/A générique
// — il n'a pas à l'être : on ne vérifie que notre propre sortie, dont on connaît
// la forme exacte. Il relit le fichier écrit sur le disque et confirme que les
// structures que l'assembleur devait poser y sont bien, et que le XML embarqué
// est celui qu'on a généré.
//
// La conformité PDF/A-3b de fond reste vérifiée par veraPDF (via Mustang) en
// intégration continue, où Java est disponible ; cf. validate_oracle_test.go.
// Ici l'objectif est différent : attraper une régression sans rien exiger de la
// machine de l'utilisateur.

// SelfCheck relit le PDF Factur-X écrit à path et vérifie sa structure ainsi que
// l'intégrité du XML embarqué face à wantXML. Renvoie la liste des anomalies ;
// vide si tout est conforme.
func SelfCheck(path string, wantXML []byte) ([]string, error) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	ctx, err := api.ReadContextFile(path)
	if err != nil {
		return nil, fmt.Errorf("facturx: relecture %s: %w", path, err)
	}
	ctx.Configuration = conf
	xt := ctx.XRefTable

	root, err := xt.Catalog()
	if err != nil {
		return nil, fmt.Errorf("facturx: catalogue: %w", err)
	}

	var bad []string
	report := func(format string, a ...any) { bad = append(bad, fmt.Sprintf(format, a...)) }

	if xt.ID == nil {
		report("/ID absent du trailer (exigé en PDF/A)")
	}

	spec := checkAssociatedFile(xt, root, report)
	checkEmbeddedXML(xt, spec, wantXML, report)
	checkOutputIntent(xt, root, report)
	checkXMP(xt, root, report)

	return bad, nil
}

// checkAssociatedFile vérifie l'entrée /AF du catalogue et renvoie le descripteur
// de fichier associé, ou nil s'il est absent ou mal formé.
func checkAssociatedFile(xt *model.XRefTable, root types.Dict, report func(string, ...any)) types.Dict {
	arr, err := xt.DereferenceArray(root["AF"])
	if err != nil || len(arr) == 0 {
		report("/AF absent du catalogue : le XML ne serait pas reconnu comme fichier associé")
		return nil
	}
	spec, err := xt.DereferenceDict(arr[0])
	if err != nil || spec == nil {
		report("/AF présent mais le descripteur de fichier est illisible")
		return nil
	}
	if n := spec.NameEntry("AFRelationship"); n == nil || *n != facturxRelation {
		got := "absent"
		if n != nil {
			got = *n
		}
		report("/AFRelationship = %s, attendu %s", got, facturxRelation)
	}
	return spec
}

// checkEmbeddedXML confirme que le flux embarqué est bien le XML généré, octet
// pour octet — c'est la garantie que les champs étendus (BT-34/BT-49) n'ont subi
// aucune re-sérialisation.
func checkEmbeddedXML(xt *model.XRefTable, spec types.Dict, want []byte, report func(string, ...any)) {
	if spec == nil {
		return
	}
	ef, err := xt.DereferenceDict(spec["EF"])
	if err != nil || ef == nil {
		report("descripteur de fichier sans entrée /EF")
		return
	}
	sd, _, err := xt.DereferenceStreamDict(ef["F"])
	if err != nil || sd == nil {
		report("fichier embarqué illisible")
		return
	}
	if n := sd.NameEntry("Subtype"); n == nil || *n != "text/xml" {
		report("sous-type du fichier embarqué inattendu")
	}
	if err := sd.Decode(); err != nil {
		report("fichier embarqué indécodable : %v", err)
		return
	}
	if !bytes.Equal(sd.Content, want) {
		report("le XML embarqué diffère du XML généré (%d octets contre %d)", len(sd.Content), len(want))
	}
}

func checkOutputIntent(xt *model.XRefTable, root types.Dict, report func(string, ...any)) {
	arr, err := xt.DereferenceArray(root["OutputIntents"])
	if err != nil || len(arr) == 0 {
		report("/OutputIntents absent : chaque usage de DeviceRGB serait une violation PDF/A")
		return
	}
	oi, err := xt.DereferenceDict(arr[0])
	if err != nil || oi == nil {
		report("/OutputIntents illisible")
		return
	}
	if n := oi.NameEntry("S"); n == nil || *n != "GTS_PDFA1" {
		report("/OutputIntent /S attendu GTS_PDFA1")
	}
	if _, _, err := xt.DereferenceStreamDict(oi["DestOutputProfile"]); err != nil {
		report("profil ICC de sortie absent ou illisible")
	}
}

func checkXMP(xt *model.XRefTable, root types.Dict, report func(string, ...any)) {
	sd, _, err := xt.DereferenceStreamDict(root["Metadata"])
	if err != nil || sd == nil {
		report("/Metadata absent : le PDF ne s'annonce pas comme PDF/A")
		return
	}
	if err := sd.Decode(); err != nil {
		report("paquet XMP indécodable : %v", err)
		return
	}
	xmp := string(sd.Content)
	for _, want := range []struct{ needle, label string }{
		{"<pdfaid:part>3</pdfaid:part>", "identification PDF/A partie 3"},
		{"<pdfaid:conformance>B</pdfaid:conformance>", "niveau de conformité B"},
		{facturxNS, "espace de noms Factur-X"},
		{"pdfaExtension:schemas", "schéma d'extension PDF/A"},
		{"<fx:DocumentFileName>" + facturxXMLName + "</fx:DocumentFileName>", "nom du fichier XML"},
		{"<fx:ConformanceLevel>" + facturxConformance + "</fx:ConformanceLevel>", "niveau Factur-X"},
	} {
		if !strings.Contains(xmp, want.needle) {
			report("XMP : %s manquant", want.label)
		}
	}
}
