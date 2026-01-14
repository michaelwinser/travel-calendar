package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/user/travel-calendar/cli/internal/client"
	"github.com/user/travel-calendar/cli/internal/output"
)

var documentsCmd = &cobra.Command{
	Use:     "documents",
	Aliases: []string{"docs"},
	Short:   "Manage documents",
	Long:    `List travel documents.`,
}

// documents list
var documentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List documents",
	Long:  `List all documents with optional filters.`,
	Run: func(cmd *cobra.Command, args []string) {
		tripID, _ := cmd.Flags().GetString("trip")
		unassociated, _ := cmd.Flags().GetBool("unassociated")

		params := &client.ListDocumentsParams{}
		if tripID != "" {
			id, err := parseUUID(tripID)
			if err != nil {
				output.PrintError("Invalid trip ID", err)
			}
			params.TripId = &id
		}
		if unassociated {
			params.Unassociated = &unassociated
		}

		resp, err := getClient().ListDocumentsWithResponse(getContext(), params)
		if err != nil {
			output.PrintError("Failed to list documents", err)
		}
		if resp.StatusCode() != http.StatusOK {
			output.PrintError(fmt.Sprintf("API returned %d", resp.StatusCode()), nil)
		}

		output.PrintDocuments(*resp.JSON200)
	},
}

func init() {
	rootCmd.AddCommand(documentsCmd)

	// documents list
	documentsCmd.AddCommand(documentsListCmd)
	documentsListCmd.Flags().String("trip", "", "Filter by trip ID")
	documentsListCmd.Flags().Bool("unassociated", false, "Show only unassociated documents")
}
