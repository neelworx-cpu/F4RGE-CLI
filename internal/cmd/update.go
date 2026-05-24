package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update 4RGED",
	Long:  "Download and install the latest 4RGED binary while keeping local sessions and configuration.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("run this in PowerShell: iwr https://4rged.ai/install.ps1 -useb | iex")
		}
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "sh"
		}
		install := exec.CommandContext(cmd.Context(), shell, "-c", "curl https://4rged.ai/install -fLsS | bash")
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		install.Stdin = os.Stdin
		return install.Run()
	},
}
