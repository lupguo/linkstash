package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// ServerURL is the LinkStash server URL.
	ServerURL string
	// SecretKey is the authentication secret key.
	SecretKey string
	// Token is the JWT token obtained by exchanging the secret key.
	Token string
)

// RootCmd is the root command for the linkstash CLI.
var RootCmd = &cobra.Command{
	Use:   "linkstash",
	Short: "LinkStash CLI - manage your bookmarks and short links",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if ServerURL == "" {
			fmt.Fprintln(os.Stderr, "Error: server URL is not set. Use --server flag or LINKSTASH_SERVER env var.")
			os.Exit(1)
		}
		if SecretKey == "" {
			fmt.Fprintln(os.Stderr, "Error: secret key is not set. Use --secret-key flag or LINKSTASH_SECRET_KEY env var.")
			os.Exit(1)
		}

		// Exchange secret_key for JWT token
		jwt, err := exchangeToken(ServerURL, SecretKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: authentication failed: %v\n", err)
			os.Exit(1)
		}
		Token = jwt
	},
}

func init() {
	RootCmd.PersistentFlags().StringVar(&ServerURL, "server", os.Getenv("LINKSTASH_SERVER"), "LinkStash server URL (env: LINKSTASH_SERVER)")
	RootCmd.PersistentFlags().StringVar(&SecretKey, "secret-key", os.Getenv("LINKSTASH_SECRET_KEY"), "Authentication secret key (env: LINKSTASH_SECRET_KEY)")
}

// Execute runs the root command.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// exchangeToken calls /api/auth/token to exchange a secret_key for a JWT.
func exchangeToken(serverURL, secretKey string) (string, error) {
	body := fmt.Sprintf(`{"secret_key":"%s"}`, secretKey)
	resp, err := http.Post(serverURL+"/api/auth/token", "application/json", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if errObj, ok := result["error"].(map[string]interface{}); ok {
			return "", fmt.Errorf("%v", errObj["message"])
		}
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	token, ok := result["token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("no token in response")
	}
	return token, nil
}
