package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// ServerURL is the LinkStash server URL.
	ServerURL string
	// Token is the authentication token.
	Token string
)

// RootCmd is the root command for the linkstash CLI.
var RootCmd = &cobra.Command{
	Use:   "linkstash",
	Short: "LinkStash CLI - manage your bookmarks and short links",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if ServerURL == "" {
			fmt.Fprintln(os.Stderr, "Error: LINKSTASH_SERVER is not set. Use --server flag or LINKSTASH_SERVER environment variable.")
			os.Exit(1)
		}
		if Token == "" {
			fmt.Fprintln(os.Stderr, "Error: LINKSTASH_TOKEN is not set. Use --token flag or LINKSTASH_TOKEN environment variable.")
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.PersistentFlags().StringVar(&ServerURL, "server", os.Getenv("LINKSTASH_SERVER"), "LinkStash server URL (env: LINKSTASH_SERVER)")
	RootCmd.PersistentFlags().StringVar(&Token, "token", os.Getenv("LINKSTASH_TOKEN"), "Authentication token (env: LINKSTASH_TOKEN)")
}

// Execute runs the root command.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
