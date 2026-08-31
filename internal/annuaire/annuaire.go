// Package annuaire interroge les annuaires publics : l'API Recherche
// d'entreprises (SIRENE) pour résoudre un nom en identité légale, et
// l'annuaire Peppol pour l'adressabilité électronique.
//
// Vie privée — règles non négociables :
//
//   - appel sortant UNIQUEMENT sur invocation explicite d'un outil de
//     recherche, jamais en tâche de fond ;
//   - seule la chaîne recherchée part sur le réseau — jamais le contenu
//     d'une facture ;
//   - GOFACT_OFFLINE=1 coupe toutes les sources réseau ;
//   - les réponses sont mises en cache sur disque pour limiter les appels.
package annuaire

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// EnvOffline coupe toutes les sources réseau quand elle vaut une valeur non vide.
const EnvOffline = "GOFACT_OFFLINE"

// Offline dit si les sources réseau sont désactivées.
func Offline() bool { return os.Getenv(EnvOffline) != "" }

// Candidate est une entreprise résolue par un annuaire.
type Candidate struct {
	Source     string `json:"source"` // "sirene"
	Name       string `json:"name"`
	SIREN      string `json:"siren"`
	SIRET      string `json:"siret,omitempty"` // siège
	Address    string `json:"address,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	City       string `json:"city,omitempty"`
	Active     bool   `json:"active"`
}

// Routing est une adresse de routage électronique trouvée dans l'annuaire Peppol.
type Routing struct {
	Scheme string `json:"scheme"` // ex. "0225" (EAS FR) ou "0009" (SIRET Peppol)
	Value  string `json:"value"`
}

// client est le client HTTP partagé : court, l'utilisateur attend derrière.
var client = &http.Client{Timeout: 10 * time.Second}

// cacheTTL borne la fraîcheur du cache disque.
const cacheTTL = 24 * time.Hour

// cached exécute fetch en passant par le cache disque (~/.cache/gofact/annuaire).
// Une réponse en cache et fraîche évite tout appel réseau.
func cached(key string, out any, fetch func() ([]byte, error)) error {
	dir, err := os.UserCacheDir()
	if err != nil {
		raw, err := fetch()
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, out)
	}
	sum := sha256.Sum256([]byte(key))
	path := filepath.Join(dir, "gofact", "annuaire", hex.EncodeToString(sum[:8])+".json")

	if fi, err := os.Stat(path); err == nil && time.Since(fi.ModTime()) < cacheTTL {
		if raw, err := os.ReadFile(path); err == nil && json.Unmarshal(raw, out) == nil {
			return nil
		}
	}
	raw, err := fetch()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
		_ = os.WriteFile(path, raw, 0o644)
	}
	return nil
}
