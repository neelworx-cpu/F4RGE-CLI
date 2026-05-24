package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/modelcatalog"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available F4RGE managed models",
	Long: `List effective F4RGE managed models for the signed-in user.

Local BYOK/provider models are not part of the managed F4RGE CLI product path.`,
	Example: `# List available managed models
4rged models

# Search models
4rged models gpt`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := f4rgesession.Load()
		if err != nil {
			return err
		}
		if !f4rgesession.IsUsable(session) {
			return fmt.Errorf("F4RGE sign-in is incomplete; run `4rged login --force`")
		}
		bundle, catalogErr := modelcatalog.Fetch(session)
		if catalogErr == nil && bundle != nil && len(bundle.Models) > 0 {
			term := strings.ToLower(strings.Join(args, " "))
			for _, model := range bundle.Models {
				haystack := strings.ToLower(strings.Join([]string{model.ID, model.Provider, model.ProviderModelID, model.Label, model.Description}, " "))
				if term != "" && !strings.Contains(haystack, term) {
					continue
				}
				if !isatty.IsTerminal(os.Stdout.Fd()) {
					fmt.Println(model.ID)
					continue
				}
				fmt.Printf("%-28s %-12s %-16s %s\n", model.ID, model.Provider, model.RequestProfile.APIFamily, model.Label)
			}
			return nil
		}

		if session == nil {
			fmt.Println("Sign in to F4RGE to load your managed model catalog.")
			fmt.Println()
			fmt.Println("Next step: run `4rged login`.")
			return nil
		}
		return fmt.Errorf("could not load managed model catalog: %w", catalogErr)
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
