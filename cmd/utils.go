package cmd

import (
	"fmt"
	"os"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/option"
	"github.com/cloudflare/cloudflare-go/v6/zones"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var client *cloudflare.Client

func initClient() error {
	// Check for legacy environment variables
	apiToken := os.Getenv("CF_API_TOKEN")
	apiKey := os.Getenv("CF_API_KEY")
	apiEmail := os.Getenv("CF_API_EMAIL")

	var opts []option.RequestOption

	if apiToken != "" {
		opts = append(opts, option.WithAPIToken(apiToken))
	} else if apiKey != "" && apiEmail != "" {
		opts = append(opts, option.WithAPIKey(apiKey), option.WithAPIEmail(apiEmail))
	}
	// If neither is set, cloudflare-go will try to read CLOUDFLARE_API_TOKEN etc. from env

	c := cloudflare.NewClient(opts...)
	client = c
	return nil
}

func writeTable(data [][]string, cols ...string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(cols)
	table.SetBorder(false)
	table.AppendBulk(data)
	table.Render()
}

// ensureClient can be used by commands to make sure client is ready
func ensureClient() error {
	if client == nil {
		return initClient()
	}
	return nil
}

// formatBool converts boolean to string "true"/"false"
func formatBool(b bool) string {
	return fmt.Sprintf("%t", b)
}

func getZoneIDByName(c *cobra.Command, zoneName string) (string, error) {
	params := zones.ZoneListParams{
		Name: cloudflare.F(zoneName),
	}
	// List first page only is enough to check existence
	res, err := client.Zones.List(c.Context(), params)
	if err != nil {
		return "", err
	}

	if len(res.Result) == 0 {
		return "", fmt.Errorf("zone %q not found", zoneName)
	}

	return res.Result[0].ID, nil
}

func checkFlags(c *cobra.Command, flags ...string) error {
	for _, flag := range flags {
		val, err := c.Flags().GetString(flag)
		if err != nil {
			return err
		}
		if val == "" {
			return fmt.Errorf("error: the required flag %q was empty or not provided", flag)
		}
	}
	return nil
}
