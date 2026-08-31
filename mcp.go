package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/kolapsis/gofact/internal/mcpsrv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version du binaire, injectable au build (-ldflags "-X main.version=…").
var version = "dev"

// runMCP démarre le serveur MCP sur stdio. À partir d'ici, stdout appartient au
// protocole JSON-RPC : toute sortie humaine passe par stderr.
func runMCP(argv []string) {
	fs := flag.NewFlagSet("gofact mcp", flag.ExitOnError)
	_ = fs.Parse(argv)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srv := mcpsrv.New(version)
	fmt.Fprintln(os.Stderr, "gofact mcp — serveur MCP (stdio) démarré")
	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil && ctx.Err() == nil {
		fail(err)
	}
}
