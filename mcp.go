package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/kolapsis/gofact/internal/dotenv"
	"github.com/kolapsis/gofact/internal/mcpsrv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version du binaire, injectable au build (-ldflags "-X main.version=…").
var version = "dev"

// runMCP démarre le serveur MCP sur stdio. À partir d'ici, stdout appartient au
// protocole JSON-RPC : toute sortie humaine passe par stderr.
func runMCP(argv []string) {
	fs := flag.NewFlagSet("gofact mcp", flag.ExitOnError)
	envPath := fs.String("env", "", "fichier .env (vendeur, IBAN, identifiants PDP, GOFACT_CHROME) ; défaut ./.env puis ~/.config/gofact/.env")
	_ = fs.Parse(argv)

	// Un client MCP lance le serveur sans l'environnement du terminal. Sans ce
	// chargement, aucune variable du .env n'atteint le serveur — et le recours
	// GOFACT_CHROME que proposent les messages d'erreur reste sans effet, alors
	// que c'est le seul dont dispose l'utilisateur en mode MCP. Un .env absent
	// n'est pas une erreur : on démarre quand même.
	if err := dotenv.LoadDefault(*envPath); err != nil {
		fmt.Fprintln(os.Stderr, "gofact mcp — .env ignoré :", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srv := mcpsrv.New(version)
	fmt.Fprintln(os.Stderr, "gofact mcp — serveur MCP (stdio) démarré")
	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil && ctx.Err() == nil {
		fail(err)
	}
}
