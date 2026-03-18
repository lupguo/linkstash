package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search URLs in LinkStash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		searchType, _ := cmd.Flags().GetString("type")

		reqURL := fmt.Sprintf("%s/api/search?q=%s&type=%s", ServerURL, url.QueryEscape(query), url.QueryEscape(searchType))
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+Token)

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

		data, ok := result["data"].([]interface{})
		if !ok {
			fmt.Println("No results found.")
			return
		}

		for i, item := range data {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			// Results are {url: {...}, score: N}
			urlData, _ := m["url"].(map[string]interface{})
			score, _ := m["score"].(float64)
			if urlData == nil {
				continue
			}
			title := urlData["title"]
			link := urlData["link"]
			desc := urlData["description"]
			fmt.Printf("%d. %v (score: %.2f)\n", i+1, title, score)
			fmt.Printf("   %v\n", link)
			if desc != nil && desc != "" {
				fmt.Printf("   %v\n", desc)
			}
			fmt.Println()
		}
	},
}

func init() {
	searchCmd.Flags().String("type", "hybrid", "Search type: keyword, vector, or hybrid")
	RootCmd.AddCommand(searchCmd)
}
