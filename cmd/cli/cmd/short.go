package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var shortCmd = &cobra.Command{
	Use:   "short [url]",
	Short: "Create a short link for a URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		longURL := args[0]
		ttl, _ := cmd.Flags().GetString("ttl")

		body := fmt.Sprintf(`{"long_url":"%s","ttl":"%s"}`, longURL, ttl)
		req, err := http.NewRequest("POST", ServerURL+"/api/short-links", strings.NewReader(body))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
			os.Exit(1)
		}

		if resp.StatusCode >= 400 {
			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Fprintf(os.Stderr, "Error: %s\n", string(out))
			os.Exit(1)
		}

		code, _ := result["code"].(string)
		fmt.Printf("Short link created!\n")
		fmt.Printf("Code:     %s\n", code)
		fmt.Printf("URL:      %s/s/%s\n", ServerURL, code)
		fmt.Printf("Long URL: %s\n", result["long_url"])
		if exp, ok := result["expires_at"]; ok && exp != nil {
			fmt.Printf("Expires:  %v\n", exp)
		}
	},
}

func init() {
	shortCmd.Flags().String("ttl", "", "Time to live (e.g., 1d, 7d, 30d)")
	RootCmd.AddCommand(shortCmd)
}
