package cmd

import (
	"fmt"
	"os"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/option"
	"github.com/olekukonko/tablewriter"
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
