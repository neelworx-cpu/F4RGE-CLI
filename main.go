// Package main is the entry point for the 4rged CLI.
//
//	@title			4rged API
//	@version		1.0
//	@description	4rged is F4RGE's AI coding agent. This API is served over a Unix socket (or Windows named pipe) and provides programmatic access to workspaces, sessions, agents, LSP, MCP, and more.
//	@contact.name	F4RGE
//	@contact.url	https://4rged.app
//	@license.name	F4RGE Functional Source License
//	@license.url	https://github.com/neelworx-cpu/F4RGE-CLI/blob/main/LICENSE.md
//	@BasePath		/v1
package main

import (
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/cmd"
	_ "github.com/neelworx-cpu/F4RGE-CLI/internal/dns"
)

func main() {
	if os.Getenv("F4RGED_PROFILE") != "" {
		go func() {
			slog.Info("Serving pprof at localhost:6060")
			if httpErr := http.ListenAndServe("localhost:6060", nil); httpErr != nil {
				slog.Error("Failed to pprof listen", "error", httpErr)
			}
		}()
	}

	cmd.Execute()
}
