package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [id]",
	Short: "Get details of a URL by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		req, err := http.NewRequest("GET", ServerURL+"/api/urls/"+id, nil)
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

		if resp.StatusCode >= 400 {
			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Fprintf(os.Stderr, "Error: %s\n", string(out))
			os.Exit(1)
		}

		fmt.Printf("ID:          %v\n", result["ID"])
		fmt.Printf("Link:        %v\n", result["link"])
		fmt.Printf("Title:       %v\n", result["title"])
		fmt.Printf("Description: %v\n", result["description"])
		fmt.Printf("Category:    %v\n", result["category"])
		fmt.Printf("Tags:        %v\n", result["tags"])
		fmt.Printf("Status:      %v\n", result["status"])
		fmt.Printf("Visits:      %v\n", result["visit_count"])
		fmt.Printf("Created:     %v\n", result["CreatedAt"])
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
}
