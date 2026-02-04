package cmd

import (
	"strings"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/zones"
	"github.com/spf13/cobra"
)

// zoneCmd represents the zone command
var zoneCmd = &cobra.Command{
	Use:   "zone",
	Short: "Zone information",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureClient()
	},
}

var zoneListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all zones on an account",
	RunE: func(cmd *cobra.Command, args []string) error {
		return zoneList(cmd)
	},
}

var zoneCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new zone",
	RunE: func(cmd *cobra.Command, args []string) error {
		return zoneCreate(cmd)
	},
}

var zoneInfoCmd = &cobra.Command{
	Use:   "info [zone]",
	Short: "Information on one zone",
	RunE: func(cmd *cobra.Command, args []string) error {
		return zoneInfo(cmd, args)
	},
}

var zoneDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a zone",
	RunE: func(cmd *cobra.Command, args []string) error {
		return zoneDelete(cmd)
	},
}

func init() {
	rootCmd.AddCommand(zoneCmd)
	zoneCmd.AddCommand(zoneListCmd)
	zoneCmd.AddCommand(zoneCreateCmd)
	zoneCmd.AddCommand(zoneInfoCmd)
	zoneCmd.AddCommand(zoneDeleteCmd)

	zoneCreateCmd.Flags().String("zone", "", "zone name")
	_ = zoneCreateCmd.MarkFlagRequired("zone")
	zoneCreateCmd.Flags().Bool("jumpstart", false, "automatically fetch DNS records (ignored)")

	zoneInfoCmd.Flags().String("zone", "", "zone name")

	zoneDeleteCmd.Flags().String("zone", "", "zone name")
	_ = zoneDeleteCmd.MarkFlagRequired("zone")
}

func zoneList(c *cobra.Command) error {
	// ListAutoPaging to get all zones
	pager := client.Zones.ListAutoPaging(c.Context(), zones.ZoneListParams{})

	output := make([][]string, 0)
	for pager.Next() {
		z := pager.Current()
		output = append(output, []string{
			z.ID,
			z.Name,
			z.Plan.Name,
			string(z.Status),
		})
	}
	if err := pager.Err(); err != nil {
		return err
	}

	writeTable(output, "ID", "Name", "Plan", "Status")
	return nil
}

func zoneCreate(c *cobra.Command) error {
	zoneName, _ := c.Flags().GetString("zone")
	accountID, _ := c.Flags().GetString("account-id")

	params := zones.ZoneNewParams{
		Name: cloudflare.F(zoneName),
		Type: cloudflare.F(zones.TypeFull),
	}

	if accountID != "" {
		params.Account = cloudflare.F(zones.ZoneNewParamsAccount{
			ID: cloudflare.F(accountID),
		})
	}

	_, err := client.Zones.New(c.Context(), params)
	return err
}

func zoneInfo(c *cobra.Command, args []string) error {
	var zoneName string
	if len(args) > 0 {
		zoneName = args[0]
	} else {
		zoneName, _ = c.Flags().GetString("zone")
	}

	if zoneName == "" {
		return c.Help()
	}

	params := zones.ZoneListParams{
		Name: cloudflare.F(zoneName),
	}

	pager := client.Zones.ListAutoPaging(c.Context(), params)

	output := make([][]string, 0)
	for pager.Next() {
		z := pager.Current()

		var nameservers []string
		if len(z.VanityNameServers) > 0 {
			nameservers = z.VanityNameServers
		} else {
			nameservers = z.NameServers
		}

		output = append(output, []string{
			z.ID,
			z.Name,
			z.Plan.Name,
			string(z.Status),
			strings.Join(nameservers, ", "),
			formatBool(z.Paused),
			string(z.Type),
		})
	}
	if err := pager.Err(); err != nil {
		return err
	}

	writeTable(output, "ID", "Zone", "Plan", "Status", "Name Servers", "Paused", "Type")
	return nil
}

func zoneDelete(c *cobra.Command) error {
	zoneName, _ := c.Flags().GetString("zone")

	// Pass context to helper
	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	params := zones.ZoneDeleteParams{
		ZoneID: cloudflare.F(zoneID),
	}

	_, err = client.Zones.Delete(c.Context(), params)
	return err
}

