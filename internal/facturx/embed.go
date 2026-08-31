package facturx

import _ "embed"

// srgbICC est le profil sRGB embarqué, écrit dans un fichier temporaire pour
// servir d'OutputIntent au PDF/A-3 généré par Ghostscript. Embarqué pour rendre
// le binaire autonome (indépendant du chemin d'installation de Ghostscript).
//
//go:embed srgb.icc
var srgbICC []byte
