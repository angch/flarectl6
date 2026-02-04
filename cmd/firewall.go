package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/accounts"
	"github.com/cloudflare/cloudflare-go/v6/firewall"
	"github.com/spf13/cobra"
)

var firewallCmd = &cobra.Command{
	Use:              "firewall",
	Short:            "Firewall",
	TraverseChildren: true,
}

var firewallRulesCmd = &cobra.Command{
	Use:              "rules",
	Short:            "Access Rules",
	TraverseChildren: true,
}

var firewallAccessRulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List firewall access rules",
	RunE:  firewallAccessRulesList,
}

var firewallAccessRuleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a firewall access rule",
	RunE:  firewallAccessRuleCreate,
}

var firewallAccessRuleUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a firewall access rule",
	RunE:  firewallAccessRuleUpdate,
}

var firewallAccessRuleCreateOrUpdateCmd = &cobra.Command{
	Use:   "create-or-update",
	Short: "Create a firewall access rule, or update it if it exists",
	RunE:  firewallAccessRuleCreateOrUpdate,
}

var firewallAccessRuleDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a firewall access rule",
	RunE:  firewallAccessRuleDelete,
}

func init() {
	rootCmd.AddCommand(firewallCmd)
	firewallCmd.AddCommand(firewallRulesCmd)
	firewallRulesCmd.AddCommand(firewallAccessRulesListCmd)
	firewallRulesCmd.AddCommand(firewallAccessRuleCreateCmd)
	firewallRulesCmd.AddCommand(firewallAccessRuleUpdateCmd)
	firewallRulesCmd.AddCommand(firewallAccessRuleCreateOrUpdateCmd)
	firewallRulesCmd.AddCommand(firewallAccessRuleDeleteCmd)

	// Flags for list
	firewallAccessRulesListCmd.Flags().String("zone", "", "zone name")
	firewallAccessRulesListCmd.Flags().String("account", "", "account name") // Legacy uses name, we need to resolve or use ID if passed as generic arg? Legacy resolves name to ID.
	firewallAccessRulesListCmd.Flags().String("value", "", "rule value")
	firewallAccessRulesListCmd.Flags().String("scope-type", "", "rule scope") // 'user', 'organization', etc.
	firewallAccessRulesListCmd.Flags().String("mode", "", "rule mode")
	firewallAccessRulesListCmd.Flags().String("notes", "", "rule notes")

	// Flags for create
	firewallAccessRuleCreateCmd.Flags().String("zone", "", "zone name")
	firewallAccessRuleCreateCmd.Flags().String("account", "", "account name")
	firewallAccessRuleCreateCmd.Flags().String("value", "", "rule value")
	firewallAccessRuleCreateCmd.Flags().String("mode", "", "rule mode")
	firewallAccessRuleCreateCmd.Flags().String("notes", "", "rule notes")

	// Flags for update
	firewallAccessRuleUpdateCmd.Flags().String("id", "", "rule id")
	firewallAccessRuleUpdateCmd.Flags().String("zone", "", "zone name")
	firewallAccessRuleUpdateCmd.Flags().String("account", "", "account name")
	firewallAccessRuleUpdateCmd.Flags().String("mode", "", "rule mode")
	firewallAccessRuleUpdateCmd.Flags().String("notes", "", "rule notes")

	// Flags for create-or-update
	firewallAccessRuleCreateOrUpdateCmd.Flags().String("zone", "", "zone name")
	firewallAccessRuleCreateOrUpdateCmd.Flags().String("account", "", "account name")
	firewallAccessRuleCreateOrUpdateCmd.Flags().String("value", "", "rule value")
	firewallAccessRuleCreateOrUpdateCmd.Flags().String("mode", "", "rule mode")
	firewallAccessRuleCreateOrUpdateCmd.Flags().String("notes", "", "rule notes")

	// Flags for delete
	firewallAccessRuleDeleteCmd.Flags().String("id", "", "rule id")
	firewallAccessRuleDeleteCmd.Flags().String("zone", "", "zone name")
	firewallAccessRuleDeleteCmd.Flags().String("account", "", "account name")
}

func getScope(c *cobra.Command) (string, string, error) {
	accountName, _ := c.Flags().GetString("account")
	zoneName, _ := c.Flags().GetString("zone")

	if accountName != "" && zoneName != "" {
		return "", "", errors.New("Cannot specify both --zone and --account")
	}

	if zoneName != "" {
		zoneID, err := getZoneIDByName(c, zoneName)
		if err != nil {
			return "", "", err
		}
		return "", zoneID, nil
	}

	if accountName != "" {
		// Legacy resolved account name to ID.
		// We need to list accounts and find the one with the name.
		if client == nil {
			if err := initClient(); err != nil {
				return "", "", err
			}
		}
		// Note: Legacy implementation iterates all accounts to find the name.
		// v6 List accounts params
		// accounts.AccountListParams
		// We can't filter by name in v6 AccountListParams?
		// Checking ref/cloudflare-go/accounts/account.go ...
		// AccountListParams has Name param?
		// I will check this separately. For now assuming we can iterate or filter.
		// Assuming we can iterate all accounts.
		// But wait, getScope in legacy code:
		/*
			accounts, _, err := api.Accounts(context.Background(), params)
			for _, acc := range accounts {
				if acc.Name == account {
					accountID = acc.ID
					break
				}
			}
		*/
		// I will use ListAutoPaging to find it.
		// For now, I will assume account flag is ID if I can't easily resolve name, but legacy says "account name".
		// Actually, I'll implement account name resolution.
		return resolveAccountIDByName(c, accountName), "", nil
	}

	// If neither, return error as User scope is not supported
	return "", "", errors.New("User-level access rules are not supported in this version. Please specify --zone or --account.")
}

func resolveAccountIDByName(c *cobra.Command, name string) string {
	iter := client.Accounts.ListAutoPaging(c.Context(), accounts.AccountListParams{})
	for iter.Next() {
		acc := iter.Current()
		if acc.Name == name {
			return acc.ID
		}
	}
	if err := iter.Err(); err != nil {
		// Log error or just return original name (assuming it might be an ID)
		fmt.Fprintf(os.Stderr, "Error listing accounts: %v\n", err)
	}
	// If not found by name, assume it's an ID
	return name
}

func getConfiguration(value string) (firewall.AccessRuleNewParamsConfigurationUnion, error) {
	// Target can be ip, ip_range, asn, country
	// Based on value format.
	if value == "" {
		return nil, nil
	}

	ip := net.ParseIP(value)
	_, cidr, cidrErr := net.ParseCIDR(value)
	_, asnErr := strconv.ParseInt(value, 10, 32)

	if ip != nil {
		return firewall.AccessRuleIPConfigurationParam{
			Target: cloudflare.F(firewall.AccessRuleIPConfigurationTargetIP),
			Value:  cloudflare.F(ip.String()),
		}, nil
	} else if cidrErr == nil {
		// Ensure correct masking
		cidr.IP = cidr.IP.Mask(cidr.Mask)
		return firewall.AccessRuleCIDRConfigurationParam{
			Target: cloudflare.F(firewall.AccessRuleCIDRConfigurationTargetIPRange),
			Value:  cloudflare.F(cidr.String()),
		}, nil
	} else if asnErr == nil {
		return firewall.ASNConfigurationParam{
			Target: cloudflare.F(firewall.ASNConfigurationTargetASN),
			Value:  cloudflare.F(value),
		}, nil
	} else {
		// Country
		return firewall.CountryConfigurationParam{
			Target: cloudflare.F(firewall.CountryConfigurationTargetCountry),
			Value:  cloudflare.F(value),
		}, nil
	}
}

func getListConfiguration(c *cobra.Command) (firewall.AccessRuleListParamsConfiguration, error) {
	value, _ := c.Flags().GetString("value")
	if value == "" {
		return firewall.AccessRuleListParamsConfiguration{}, nil
	}

	// Similar logic but returning ListParamsConfiguration
	// Note: ListParamsConfiguration has Target and Value.
	// Target enum: ip, ip_range, asn, country.

	ip := net.ParseIP(value)
	_, _, cidrErr := net.ParseCIDR(value)
	_, asnErr := strconv.ParseInt(value, 10, 32)

	var target firewall.AccessRuleListParamsConfigurationTarget

	if ip != nil {
		target = firewall.AccessRuleListParamsConfigurationTargetIP
	} else if cidrErr == nil {
		target = firewall.AccessRuleListParamsConfigurationTargetIPRange
	} else if asnErr == nil {
		target = firewall.AccessRuleListParamsConfigurationTargetASN
	} else {
		target = firewall.AccessRuleListParamsConfigurationTargetCountry
	}

	return firewall.AccessRuleListParamsConfiguration{
		Target: cloudflare.F(target),
		Value:  cloudflare.F(value),
	}, nil
}

func formatAccessRule(rule firewall.AccessRuleListResponse) []string {
	// Value depends on configuration type.
	// Helper to extract value from Union.
	val := rule.Configuration.Value

	return []string{
		rule.ID,
		val,
		string(rule.Scope.Type),
		string(rule.Mode),
		rule.Notes,
	}
}

func firewallAccessRulesList(c *cobra.Command, args []string) error {
	if err := ensureClient(); err != nil {
		return err
	}

	accountID, zoneID, err := getScope(c)
	if err != nil {
		return err
	}

	notes, _ := c.Flags().GetString("notes")
	mode, _ := c.Flags().GetString("mode")

	config, _ := getListConfiguration(c)

	params := firewall.AccessRuleListParams{
		Notes: cloudflare.F(notes),
	}
	if accountID != "" {
		params.AccountID = cloudflare.F(accountID)
	}
	if zoneID != "" {
		params.ZoneID = cloudflare.F(zoneID)
	}
	if mode != "" {
		// Map string to AccessRuleListParamsMode
		// "block", "challenge", "whitelist", "js_challenge", "managed_challenge"
		params.Mode = cloudflare.F(firewall.AccessRuleListParamsMode(mode))
	}
	if config.Value.Value != "" {
		params.Configuration = cloudflare.F(config)
	}

	iter := client.Firewall.AccessRules.ListAutoPaging(context.Background(), params)

	var rules []firewall.AccessRuleListResponse
	for iter.Next() {
		rules = append(rules, iter.Current())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	output := make([][]string, 0, len(rules))
	for _, rule := range rules {
		output = append(output, formatAccessRule(rule))
	}
	writeTable(output, "ID", "Value", "Scope", "Mode", "Notes")

	return nil
}

func firewallAccessRuleCreate(c *cobra.Command, args []string) error {
	if err := ensureClient(); err != nil {
		return err
	}

	if err := checkFlags(c, "mode", "value"); err != nil {
		return err
	}

	accountID, zoneID, err := getScope(c)
	if err != nil {
		return err
	}

	value, _ := c.Flags().GetString("value")
	mode, _ := c.Flags().GetString("mode")
	notes, _ := c.Flags().GetString("notes")

	config, err := getConfiguration(value)
	if err != nil {
		return err
	}

	params := firewall.AccessRuleNewParams{
		Mode:          cloudflare.F(firewall.AccessRuleNewParamsMode(mode)),
		Configuration: cloudflare.F(config),
		Notes:         cloudflare.F(notes),
	}
	if accountID != "" {
		params.AccountID = cloudflare.F(accountID)
	}
	if zoneID != "" {
		params.ZoneID = cloudflare.F(zoneID)
	}

	resp, err := client.Firewall.AccessRules.New(context.Background(), params)
	if err != nil {
		return err
	}

	// Response is AccessRuleNewResponse. Need to map to table format.
	// FormatAccessRule expects ListResponse.
	// They are structurally similar but different types.
	// I'll create a helper or manually format.
	output := [][]string{{
		resp.ID,
		resp.Configuration.Value,
		string(resp.Scope.Type),
		string(resp.Mode),
		resp.Notes,
	}}
	writeTable(output, "ID", "Value", "Scope", "Mode", "Notes")

	return nil
}

func firewallAccessRuleUpdate(c *cobra.Command, args []string) error {
	if err := ensureClient(); err != nil {
		return err
	}

	if err := checkFlags(c, "id"); err != nil {
		return err
	}
	id, _ := c.Flags().GetString("id")

	accountID, zoneID, err := getScope(c)
	if err != nil {
		return err
	}

	mode, _ := c.Flags().GetString("mode")
	notes, _ := c.Flags().GetString("notes")

	params := firewall.AccessRuleEditParams{}
	if accountID != "" {
		params.AccountID = cloudflare.F(accountID)
	}
	if zoneID != "" {
		params.ZoneID = cloudflare.F(zoneID)
	}

	if mode != "" {
		params.Mode = cloudflare.F(firewall.AccessRuleEditParamsMode(mode))
	}
	if notes != "" {
		params.Notes = cloudflare.F(notes)
	}
	// Note: Configuration cannot be updated in v6 EditParams?
	// ref/cloudflare-go/firewall/accessrule.go: AccessRuleEditParams has Configuration.
	// But legacy only updated Mode and Notes.
	// "rule := cloudflare.AccessRule{ Mode: mode, Notes: notes }"

	resp, err := client.Firewall.AccessRules.Edit(context.Background(), id, params)
	if err != nil {
		return err
	}

	output := [][]string{{
		resp.ID,
		resp.Configuration.Value,
		string(resp.Scope.Type),
		string(resp.Mode),
		resp.Notes,
	}}
	writeTable(output, "ID", "Value", "Scope", "Mode", "Notes")

	return nil
}

func firewallAccessRuleCreateOrUpdate(c *cobra.Command, args []string) error {
	if err := ensureClient(); err != nil {
		return err
	}

	if err := checkFlags(c, "mode", "value"); err != nil {
		return err
	}

	accountID, zoneID, err := getScope(c)
	if err != nil {
		return err
	}

	value, _ := c.Flags().GetString("value")
	mode, _ := c.Flags().GetString("mode")
	notes, _ := c.Flags().GetString("notes")

	// 1. Search for existing rule
	listConfig, _ := getListConfiguration(c)
	listParams := firewall.AccessRuleListParams{
		Configuration: cloudflare.F(listConfig),
	}
	if accountID != "" {
		listParams.AccountID = cloudflare.F(accountID)
	}
	if zoneID != "" {
		listParams.ZoneID = cloudflare.F(zoneID)
	}

	// We only check the first page (legacy behavior implies just finding *a* match)
	iter := client.Firewall.AccessRules.ListAutoPaging(context.Background(), listParams)
	var existingRules []firewall.AccessRuleListResponse
	for iter.Next() {
		existingRules = append(existingRules, iter.Current())
	}

	if len(existingRules) > 0 {
		// Update existing
		var output [][]string
		for _, r := range existingRules {
			updateParams := firewall.AccessRuleEditParams{}
			if accountID != "" {
				updateParams.AccountID = cloudflare.F(accountID)
			}
			if zoneID != "" {
				updateParams.ZoneID = cloudflare.F(zoneID)
			}

			// If mode or notes are empty, keep existing?
			// Legacy: if mode == "", rule.Mode = r.Mode
			if mode != "" {
				updateParams.Mode = cloudflare.F(firewall.AccessRuleEditParamsMode(mode))
			} else {
				// Convert ListResponseMode to EditParamsMode
				updateParams.Mode = cloudflare.F(firewall.AccessRuleEditParamsMode(r.Mode))
			}

			if notes != "" {
				updateParams.Notes = cloudflare.F(notes)
			} else {
				updateParams.Notes = cloudflare.F(r.Notes)
			}

			resp, err := client.Firewall.AccessRules.Edit(context.Background(), r.ID, updateParams)
			if err != nil {
				fmt.Println("Error updating firewall access rule:", err)
				continue
			}
			output = append(output, []string{
				resp.ID,
				resp.Configuration.Value,
				string(resp.Scope.Type),
				string(resp.Mode),
				resp.Notes,
			})
		}
		if len(output) > 0 {
			writeTable(output, "ID", "Value", "Scope", "Mode", "Notes")
		}
	} else {
		// Create new
		config, err := getConfiguration(value)
		if err != nil {
			return err
		}
		createParams := firewall.AccessRuleNewParams{
			Mode:          cloudflare.F(firewall.AccessRuleNewParamsMode(mode)),
			Configuration: cloudflare.F(config),
			Notes:         cloudflare.F(notes),
		}
		if accountID != "" {
			createParams.AccountID = cloudflare.F(accountID)
		}
		if zoneID != "" {
			createParams.ZoneID = cloudflare.F(zoneID)
		}

		resp, err := client.Firewall.AccessRules.New(context.Background(), createParams)
		if err != nil {
			return err
		}
		output := [][]string{{
			resp.ID,
			resp.Configuration.Value,
			string(resp.Scope.Type),
			string(resp.Mode),
			resp.Notes,
		}}
		writeTable(output, "ID", "Value", "Scope", "Mode", "Notes")
	}

	return nil
}

func firewallAccessRuleDelete(c *cobra.Command, args []string) error {
	if err := ensureClient(); err != nil {
		return err
	}

	if err := checkFlags(c, "id"); err != nil {
		return err
	}
	id, _ := c.Flags().GetString("id")

	accountID, zoneID, err := getScope(c)
	if err != nil {
		return err
	}

	params := firewall.AccessRuleDeleteParams{}
	if accountID != "" {
		params.AccountID = cloudflare.F(accountID)
	}
	if zoneID != "" {
		params.ZoneID = cloudflare.F(zoneID)
	}

	resp, err := client.Firewall.AccessRules.Delete(context.Background(), id, params)
	if err != nil {
		return err
	}

	// Output deleted ID
	output := [][]string{{
		resp.ID,
		"", // Value not returned in delete response
		"", // Scope not returned
		"", // Mode not returned
		"", // Notes not returned
	}}
	// Legacy output listed the rule details after deletion?
	// Legacy:
	/*
		resp, err := api.DeleteUserAccessRule(context.Background(), ruleID)
		rules = append(rules, resp.Result) // Legacy returned the rule?
		// check ref/cloudflare-go/firewall/accessrule.go
		// Delete returns AccessRuleDeleteResponse { ID string }
	*/
	// v6 Delete response only has ID. So we can't show full details unless we fetched it before.
	// Legacy flarectl printed the table.
	// I'll just print the ID.
	writeTable(output, "ID", "Value", "Scope", "Mode", "Notes")

	return nil
}
