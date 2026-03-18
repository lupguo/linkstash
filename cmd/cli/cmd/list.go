package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List URLs from LinkStash",
	Run: func(cmd *cobra.Command, args []string) {
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")

		url := fmt.Sprintf("%s/api/urls?page=%d&size=%d", ServerURL, page, size)
		req, err := http.NewRequest("GET", url, nil)
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
			fmt.Println("No URLs found.")
			return
		}

		fmt.Printf("%-6s %-60s %-12s\n", "ID", "Link", "Status")
		fmt.Println(strings.Repeat("-", 80))
		for _, item := range data {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			id := m["ID"]
			link := m["link"]
			status := m["status"]
			fmt.Printf("%-6v %-60v %-12v\n", id, link, status)
		}

		fmt.Printf("\nTotal: %.0f | Page: %.0f\n", result["total"], result["page"])
	},
}

func init() {
	listCmd.Flags().Int("page", 1, "Page number")
	listCmd.Flags().Int("size", 20, "Page size")
	RootCmd.AddCommand(listCmd)
}
