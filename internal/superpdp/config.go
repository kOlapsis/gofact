package superpdp

import (
	"fmt"
	"os"
)

// Identifiants de la PDP : uniquement lus dans l'environnement (ou un fichier
// .env non versionné). Aucun secret n'est jamais codé en dur ni journalisé.
const (
	envBase         = "SUPERPDP_BASE"
	envClientID     = "SUPERPDP_CLIENT_ID"
	envClientSecret = "SUPERPDP_CLIENT_SECRET"
)

// ConfigFromEnv lit les identifiants OAuth2 depuis l'environnement.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		Base:         os.Getenv(envBase),
		ClientID:     os.Getenv(envClientID),
		ClientSecret: os.Getenv(envClientSecret),
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return cfg, fmt.Errorf("superpdp: %s / %s manquants (environnement ou -env <fichier>)", envClientID, envClientSecret)
	}
	return cfg, nil
}
