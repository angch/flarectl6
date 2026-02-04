package cmd

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/page_rules"
	"github.com/goccy/go-json"
	"github.com/spf13/cobra"
)

var pageRulesCmd = &cobra.Command{
	Use:     "pagerules",
	Short:   "Page Rules",
	Aliases: []string{"p"},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureClient()
	},
}

var pageRulesListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List Page Rules for a zone",
	Aliases: []string{"l"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return listPageRules(cmd)
	},
}

func init() {
	rootCmd.AddCommand(pageRulesCmd)
	pageRulesCmd.AddCommand(pageRulesListCmd)
	pageRulesListCmd.Flags().String("zone", "", "zone name")
}

func listPageRules(c *cobra.Command) error {
	if err := checkFlags(c, "zone"); err != nil {
		return err
	}
	zoneName, _ := c.Flags().GetString("zone")
	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	rules, err := client.PageRules.List(c.Context(), page_rules.PageRuleListParams{
		ZoneID: cloudflare.F(zoneID),
	})
	if err != nil {
		return err
	}

	if rules == nil {
		return fmt.Errorf("no rules returned")
	}

	fmt.Printf("%3s %-32s %-8s %s\n", "Pri", "ID", "Status", "URL")
	for _, r := range *rules {
		var settings []string

		var urlPattern string
		if len(r.Targets) > 0 {
			urlPattern = r.Targets[0].Constraint.Value
		}

		fmt.Printf("%3d %s %-8s %s\n", r.Priority, r.ID, r.Status, urlPattern)
		for _, a := range r.Actions {
			s := formatAction(a)
			settings = append(settings, s)
		}
		fmt.Println("   ", strings.Join(settings, ", "))
	}

	return nil
}

func formatAction(a page_rules.PageRuleAction) string {
	idStr := formatID(string(a.ID))

	if a.Value == nil {
		return idStr
	}

	switch a.ID {
	case page_rules.PageRuleActionsIDForwardingURL:
		if v, ok := a.Value.(page_rules.PageRuleActionsForwardingURLValue); ok {
			return fmt.Sprintf("%s: %d - %s", idStr, int64(v.StatusCode), v.URL)
		}
		// Fallback if not mapped correctly
		if v, ok := a.Value.(map[string]interface{}); ok {
			if code, ok := v["status_code"]; ok {
				if url, ok := v["url"]; ok {
					return fmt.Sprintf("%s: %v - %v", idStr, code, url)
				}
			}
		}

	default:
		switch v := a.Value.(type) {
		case int, int64, float64:
			return fmt.Sprintf("%s: %v", idStr, v)
		case string:
			return fmt.Sprintf("%s: %s", idStr, title(strings.ReplaceAll(v, "_", " ")))
		case map[string]interface{}:
			if code, ok := v["status_code"]; ok {
				if url, ok := v["url"]; ok {
					return fmt.Sprintf("%s: %v - %v", idStr, code, url)
				}
			}
			b, _ := json.Marshal(v)
			return fmt.Sprintf("%s: %s", idStr, string(b))
		default:
			if val, ok := a.Value.(page_rules.PageRuleActionsForwardingURLValue); ok {
				return fmt.Sprintf("%s: %d - %s", idStr, int64(val.StatusCode), val.URL)
			}
			return fmt.Sprintf("%s: %v", idStr, v)
		}
	}
	return idStr
}

func formatID(id string) string {
	return title(strings.ReplaceAll(id, "_", " "))
}

// title is a replacement for strings.Title which is deprecated
func title(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		runes := []rune(w)
		if len(runes) > 0 {
			runes[0] = unicode.ToTitle(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
