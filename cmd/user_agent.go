package cmd

import (
	"fmt"
	"strconv"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/firewall"
	"github.com/spf13/cobra"
)

var userAgentCmd = &cobra.Command{
	Use:     "user-agents",
	Aliases: []string{"ua"},
	Short:   "User-Agent blocking",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureClient()
	},
}

var userAgentListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List User-Agent blocks for a zone",
	RunE: func(cmd *cobra.Command, args []string) error {
		return userAgentList(cmd)
	},
}

var userAgentCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"c"},
	Short:   "Create a User-Agent blocking rule",
	RunE: func(cmd *cobra.Command, args []string) error {
		return userAgentCreate(cmd)
	},
}

var userAgentUpdateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"u"},
	Short:   "Update an existing User-Agent block",
	RunE: func(cmd *cobra.Command, args []string) error {
		return userAgentUpdate(cmd)
	},
}

var userAgentDeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"d"},
	Short:   "Delete a User-Agent block",
	RunE: func(cmd *cobra.Command, args []string) error {
		return userAgentDelete(cmd)
	},
}

func init() {
	rootCmd.AddCommand(userAgentCmd)
	userAgentCmd.AddCommand(userAgentListCmd)
	userAgentCmd.AddCommand(userAgentCreateCmd)
	userAgentCmd.AddCommand(userAgentUpdateCmd)
	userAgentCmd.AddCommand(userAgentDeleteCmd)

	// List flags
	userAgentListCmd.Flags().String("zone", "", "zone name")
	userAgentListCmd.Flags().Int("page", 0, "result page to return")

	// Create flags
	userAgentCreateCmd.Flags().String("zone", "", "zone name")
	userAgentCreateCmd.Flags().String("mode", "", "the blocking mode: block, challenge, js_challenge, whitelist")
	userAgentCreateCmd.Flags().String("value", "", "the exact User-Agent to block")
	userAgentCreateCmd.Flags().Bool("paused", false, "whether the rule should be paused (default: false)")
	userAgentCreateCmd.Flags().String("description", "", "a description for the rule")

	// Update flags
	userAgentUpdateCmd.Flags().String("zone", "", "zone name")
	userAgentUpdateCmd.Flags().String("id", "", "User-Agent blocking rule ID")
	userAgentUpdateCmd.Flags().String("mode", "", "the blocking mode: block, challenge, js_challenge, whitelist")
	userAgentUpdateCmd.Flags().String("value", "", "the exact User-Agent to block")
	userAgentUpdateCmd.Flags().Bool("paused", false, "whether the rule should be paused (default: false)")
	userAgentUpdateCmd.Flags().String("description", "", "a description for the rule")

	// Delete flags
	userAgentDeleteCmd.Flags().String("zone", "", "zone name")
	userAgentDeleteCmd.Flags().String("id", "", "User-Agent blocking rule ID")
}

func formatUserAgentRule(id, description, mode, value string, paused bool) []string {
	return []string{
		id,
		description,
		mode,
		value,
		strconv.FormatBool(paused),
	}
}

func userAgentList(c *cobra.Command) error {
	if err := checkFlags(c, "zone"); err != nil {
		return err
	}
	zoneName, _ := c.Flags().GetString("zone")
	page, _ := c.Flags().GetInt("page")

	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	params := firewall.UARuleListParams{
		ZoneID: cloudflare.F(zoneID),
	}

	if page > 0 {
		params.Page = cloudflare.F(float64(page))
	} else {
		// Legacy behavior implicitly listed page 1 if not specified.
		// To match strict legacy behavior (which didn't list all by default), we default to page 1.
		params.Page = cloudflare.F(1.0)
	}

	// We use List, not ListAutoPaging, because we want to support specific page requests
	// and default to page 1 if not specified (matching legacy).
	resp, err := client.Firewall.UARules.List(c.Context(), params)
	if err != nil {
		return fmt.Errorf("Error listing User-Agent block rules: %w", err)
	}

	output := make([][]string, 0, len(resp.Result))
	for _, rule := range resp.Result {
		output = append(output, formatUserAgentRule(
			rule.ID,
			rule.Description,
			string(rule.Mode),
			rule.Configuration.Value,
			rule.Paused,
		))
	}

	writeTable(output, "ID", "Description", "Mode", "Value", "Paused")
	return nil
}

func userAgentCreate(c *cobra.Command) error {
	if err := checkFlags(c, "zone", "mode", "value"); err != nil {
		return err
	}

	zoneName, _ := c.Flags().GetString("zone")
	mode, _ := c.Flags().GetString("mode")
	value, _ := c.Flags().GetString("value")
	paused, _ := c.Flags().GetBool("paused")
	description, _ := c.Flags().GetString("description")

	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	params := firewall.UARuleNewParams{
		ZoneID: cloudflare.F(zoneID),
		Configuration: cloudflare.F(firewall.UARuleNewParamsConfiguration{
			Target: cloudflare.F(firewall.UARuleNewParamsConfigurationTargetUA),
			Value:  cloudflare.F(value),
		}),
		Mode:        cloudflare.F(firewall.UARuleNewParamsMode(mode)),
		Paused:      cloudflare.F(paused),
		Description: cloudflare.F(description),
	}

	resp, err := client.Firewall.UARules.New(c.Context(), params)
	if err != nil {
		return fmt.Errorf("Error creating User-Agent block rule: %w", err)
	}

	output := [][]string{
		formatUserAgentRule(
			resp.ID,
			resp.Description,
			string(resp.Mode),
			resp.Configuration.Value,
			resp.Paused,
		),
	}

	writeTable(output, "ID", "Description", "Mode", "Value", "Paused")
	return nil
}

func userAgentUpdate(c *cobra.Command) error {
	// Legacy required flags: zone, id, mode, value
	if err := checkFlags(c, "zone", "id", "mode", "value"); err != nil {
		return err
	}

	zoneName, _ := c.Flags().GetString("zone")
	id, _ := c.Flags().GetString("id")
	mode, _ := c.Flags().GetString("mode")
	value, _ := c.Flags().GetString("value")
	paused, _ := c.Flags().GetBool("paused")
	description, _ := c.Flags().GetString("description")

	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	params := firewall.UARuleUpdateParams{
		ZoneID: cloudflare.F(zoneID),
		Configuration: cloudflare.F[firewall.UARuleUpdateParamsConfigurationUnion](firewall.UARuleUpdateParamsConfiguration{
			// "ua" is not explicitly defined in enum for UpdateParamsConfigurationTarget in the library constants,
			// but it is required for User-Agent rules. Casting the string works.
			Target: cloudflare.F(firewall.UARuleUpdateParamsConfigurationTarget("ua")),
			Value:  cloudflare.F(value),
		}),
		Mode:        cloudflare.F(firewall.UARuleUpdateParamsMode(mode)),
		Paused:      cloudflare.F(paused),
		Description: cloudflare.F(description),
	}

	resp, err := client.Firewall.UARules.Update(c.Context(), id, params)
	if err != nil {
		return fmt.Errorf("Error updating User-Agent block rule: %w", err)
	}

	output := [][]string{
		formatUserAgentRule(
			resp.ID,
			resp.Description,
			string(resp.Mode),
			resp.Configuration.Value,
			resp.Paused,
		),
	}

	writeTable(output, "ID", "Description", "Mode", "Value", "Paused")
	return nil
}

func userAgentDelete(c *cobra.Command) error {
	if err := checkFlags(c, "zone", "id"); err != nil {
		return err
	}

	zoneName, _ := c.Flags().GetString("zone")
	id, _ := c.Flags().GetString("id")

	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	params := firewall.UARuleDeleteParams{
		ZoneID: cloudflare.F(zoneID),
	}

	resp, err := client.Firewall.UARules.Delete(c.Context(), id, params)
	if err != nil {
		return fmt.Errorf("Error deleting User-Agent block rule: %w", err)
	}

	output := [][]string{
		formatUserAgentRule(
			resp.ID,
			resp.Description,
			string(resp.Mode),
			resp.Configuration.Value,
			resp.Paused,
		),
	}

	writeTable(output, "ID", "Description", "Mode", "Value", "Paused")
	return nil
}
