package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/dns"
	"github.com/spf13/cobra"
)

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "DNS records",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureClient()
	},
}

func init() {
	rootCmd.AddCommand(dnsCmd)
	dnsCmd.AddCommand(dnsListCmd)

	dnsListCmd.Flags().String("zone", "", "zone name")
	dnsListCmd.Flags().String("id", "", "record id")
	dnsListCmd.Flags().String("type", "", "record type")
	dnsListCmd.Flags().String("name", "", "record name")
	dnsListCmd.Flags().String("content", "", "record content")

	dnsCmd.AddCommand(dnsCreateCmd)
	dnsCreateCmd.Flags().String("zone", "", "zone name")
	dnsCreateCmd.Flags().String("name", "", "record name")
	dnsCreateCmd.Flags().String("type", "", "record type")
	dnsCreateCmd.Flags().String("content", "", "record content")
	dnsCreateCmd.Flags().Int("ttl", 1, "TTL (1 = automatic)")
	dnsCreateCmd.Flags().Bool("proxy", false, "proxy through Cloudflare (orange cloud)")
	dnsCreateCmd.Flags().Uint("priority", 0, "priority for an MX record. Only used for MX")

	dnsCmd.AddCommand(dnsUpdateCmd)
	dnsUpdateCmd.Flags().String("zone", "", "zone name")
	dnsUpdateCmd.Flags().String("id", "", "record id")
	dnsUpdateCmd.Flags().String("name", "", "record name")
	dnsUpdateCmd.Flags().String("type", "", "record type")
	dnsUpdateCmd.Flags().String("content", "", "record content")
	dnsUpdateCmd.Flags().Int("ttl", 1, "TTL (1 = automatic)")
	dnsUpdateCmd.Flags().Bool("proxy", false, "proxy through Cloudflare (orange cloud)")
	dnsUpdateCmd.Flags().Uint("priority", 0, "priority for an MX record. Only used for MX")

	dnsCmd.AddCommand(dnsDeleteCmd)
	dnsDeleteCmd.Flags().String("zone", "", "zone name")
	dnsDeleteCmd.Flags().String("id", "", "record id")

	dnsCmd.AddCommand(dnsCreateOrUpdateCmd)
	dnsCreateOrUpdateCmd.Flags().String("zone", "", "zone name")
	dnsCreateOrUpdateCmd.Flags().String("name", "", "record name")
	dnsCreateOrUpdateCmd.Flags().String("type", "", "record type")
	dnsCreateOrUpdateCmd.Flags().String("content", "", "record content")
	dnsCreateOrUpdateCmd.Flags().Int("ttl", 1, "TTL (1 = automatic)")
	dnsCreateOrUpdateCmd.Flags().Bool("proxy", false, "proxy through Cloudflare (orange cloud)")
	dnsCreateOrUpdateCmd.Flags().Uint("priority", 0, "priority for an MX record. Only used for MX")
}

var dnsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List DNS records for a zone",
	RunE:  runDNSList,
}

func runDNSList(c *cobra.Command, args []string) error {
	zoneName, _ := c.Flags().GetString("zone")
	if zoneName == "" {
		if len(args) > 0 {
			zoneName = args[0]
		}
	}
	if zoneName == "" {
		return c.Help()
	}

	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	id, _ := c.Flags().GetString("id")
	name, _ := c.Flags().GetString("name")
	rtype, _ := c.Flags().GetString("type")
	content, _ := c.Flags().GetString("content")

	if id != "" {
		params := dns.RecordGetParams{
			ZoneID: cloudflare.F(zoneID),
		}
		res, err := client.DNS.Records.Get(c.Context(), id, params)
		if err != nil {
			return err
		}
		// List command output format: ID, Type, Name, Content, Proxied, TTL
		// Note: Legacy list used specialized output, not formatDNSRecord
		output := [][]string{{
			res.ID,
			string(res.Type),
			res.Name,
			formatDNSContent(*res),
			formatBool(res.Proxied),
			strconv.FormatFloat(float64(res.TTL), 'f', -1, 64),
		}}
		writeTable(output, "ID", "Type", "Name", "Content", "Proxied", "TTL")
		return nil
	}

	params := dns.RecordListParams{
		ZoneID: cloudflare.F(zoneID),
	}
	if name != "" {
		params.Name = cloudflare.F(dns.RecordListParamsName{Contains: cloudflare.F(name)})
	}
	if rtype != "" {
		params.Type = cloudflare.F(dns.RecordListParamsType(rtype))
	}
	if content != "" {
		params.Content = cloudflare.F(dns.RecordListParamsContent{Contains: cloudflare.F(content)})
	}

	pager := client.DNS.Records.ListAutoPaging(c.Context(), params)
	output := make([][]string, 0)
	for pager.Next() {
		r := pager.Current()
		output = append(output, []string{
			r.ID,
			string(r.Type),
			r.Name,
			formatDNSContent(r),
			formatBool(r.Proxied),
			strconv.FormatFloat(float64(r.TTL), 'f', -1, 64),
		})
	}
	if err := pager.Err(); err != nil {
		return err
	}

	writeTable(output, "ID", "Type", "Name", "Content", "Proxied", "TTL")
	return nil
}

func formatDNSContent(r dns.RecordResponse) string {
	content := r.Content
	switch string(r.Type) {
	case "MX":
		content = fmt.Sprintf("%.f %s", r.Priority, content)
	case "SRV":
		// Legacy: r.Content = fmt.Sprintf("%.f %s", dp["priority"], r.Content)
		// r.Content = strings.Replace(r.Content, "\t", " ", -1)
		content = fmt.Sprintf("%.f %s", r.Priority, content)
		content = strings.Replace(content, "\t", " ", -1)
	}
	return content
}

func formatDNSRecord(record dns.RecordResponse) []string {
	return []string{
		record.ID,
		record.Name,
		string(record.Type),
		record.Content,
		strconv.FormatFloat(float64(record.TTL), 'f', -1, 64),
		formatBool(record.Proxiable),
		formatBool(record.Proxied),
	}
}

var dnsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a DNS record",
	RunE:  runDNSCreate,
}

func runDNSCreate(c *cobra.Command, args []string) error {
	zoneName, _ := c.Flags().GetString("zone")
	// Legacy checked flags "zone", "name", "type", "content"
	if err := checkFlags(c, "zone", "name", "type", "content"); err != nil {
		return err
	}

	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	name, _ := c.Flags().GetString("name")
	rtype, _ := c.Flags().GetString("type")
	content, _ := c.Flags().GetString("content")
	ttl, _ := c.Flags().GetInt("ttl")
	proxy, _ := c.Flags().GetBool("proxy")
	priority, _ := c.Flags().GetUint("priority")

	params := dns.RecordNewParams{
		ZoneID: cloudflare.F(zoneID),
		Body: dns.RecordNewParamsBody{
			Name:    cloudflare.F(name),
			Type:    cloudflare.F(dns.RecordNewParamsBodyType(rtype)),
			Content: cloudflare.F(content),
			TTL:     cloudflare.F(dns.TTL(ttl)),
			Proxied: cloudflare.F(proxy),
		},
	}

	if priority != 0 {
		// Need to set priority if provided.
		// Since Body is an interface, and we initialized it with RecordNewParamsBody struct,
		// we can access it if we used a variable or if we construct it properly.
		body := params.Body.(dns.RecordNewParamsBody)
		body.Priority = cloudflare.F(float64(priority))
		params.Body = body
	}

	res, err := client.DNS.Records.New(c.Context(), params)
	if err != nil {
		return err
	}

	output := [][]string{
		formatDNSRecord(*res),
	}

	writeTable(output, "ID", "Name", "Type", "Content", "TTL", "Proxiable", "Proxy")

	return nil
}

var dnsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a DNS record",
	RunE:  runDNSUpdate,
}

func runDNSUpdate(c *cobra.Command, args []string) error {
	if err := checkFlags(c, "zone", "id"); err != nil {
		return err
	}
	zoneName, _ := c.Flags().GetString("zone")
	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	recordID, _ := c.Flags().GetString("id")

	// Use Edit (PATCH). Check changed flags.
	name, _ := c.Flags().GetString("name")
	rtype, _ := c.Flags().GetString("type")
	content, _ := c.Flags().GetString("content")
	ttl, _ := c.Flags().GetInt("ttl")
	proxy, _ := c.Flags().GetBool("proxy")
	priority, _ := c.Flags().GetUint("priority")

	// RecordEditParamsBody
	body := dns.RecordEditParamsBody{}

	if c.Flags().Changed("name") {
		body.Name = cloudflare.F(name)
	}
	if c.Flags().Changed("type") {
		body.Type = cloudflare.F(dns.RecordEditParamsBodyType(rtype))
	}
	if c.Flags().Changed("content") {
		body.Content = cloudflare.F(content)
	}
	if c.Flags().Changed("ttl") {
		body.TTL = cloudflare.F(dns.TTL(ttl))
	}
	if c.Flags().Changed("proxy") {
		body.Proxied = cloudflare.F(proxy)
	}
	if c.Flags().Changed("priority") {
		body.Priority = cloudflare.F(float64(priority))
	}

	params := dns.RecordEditParams{
		ZoneID: cloudflare.F(zoneID),
		Body:   body,
	}

	_, err = client.DNS.Records.Edit(c.Context(), recordID, params)
	if err != nil {
		return err
	}
	return nil
}

var dnsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a DNS record",
	RunE:  runDNSDelete,
}

func runDNSDelete(c *cobra.Command, args []string) error {
	if err := checkFlags(c, "zone", "id"); err != nil {
		return err
	}
	zoneName, _ := c.Flags().GetString("zone")
	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	recordID, _ := c.Flags().GetString("id")
	params := dns.RecordDeleteParams{
		ZoneID: cloudflare.F(zoneID),
	}

	_, err = client.DNS.Records.Delete(c.Context(), recordID, params)
	return err
}

var dnsCreateOrUpdateCmd = &cobra.Command{
	Use:   "create-or-update",
	Short: "Create a DNS record, or update if it exists",
	RunE:  runDNSCreateOrUpdate,
}

func runDNSCreateOrUpdate(c *cobra.Command, args []string) error {
	if err := checkFlags(c, "zone", "name", "type", "content"); err != nil {
		return err
	}
	zoneName, _ := c.Flags().GetString("zone")
	zoneID, err := getZoneIDByName(c, zoneName)
	if err != nil {
		return err
	}

	name, _ := c.Flags().GetString("name")
	rtype, _ := c.Flags().GetString("type")
	content, _ := c.Flags().GetString("content")
	ttl, _ := c.Flags().GetInt("ttl")
	proxy, _ := c.Flags().GetBool("proxy")
	priority, _ := c.Flags().GetUint("priority")

	// Legacy behavior: search by FQDN constructed manually
	fqdn := name + "." + zoneName
	params := dns.RecordListParams{
		ZoneID: cloudflare.F(zoneID),
		Name:   cloudflare.F(dns.RecordListParamsName{Exact: cloudflare.F(fqdn)}),
	}

	pager := client.DNS.Records.ListAutoPaging(c.Context(), params)

	foundRecords := false
	var lastResult dns.RecordResponse

	for pager.Next() {
		r := pager.Current()
		foundRecords = true

		if string(r.Type) == rtype {
			body := dns.RecordEditParamsBody{
				Type:    cloudflare.F(dns.RecordEditParamsBodyType(rtype)),
				Content: cloudflare.F(content),
				TTL:     cloudflare.F(dns.TTL(ttl)),
				Proxied: cloudflare.F(proxy),
			}
			if priority != 0 {
				body.Priority = cloudflare.F(float64(priority))
			}

			editParams := dns.RecordEditParams{
				ZoneID: cloudflare.F(zoneID),
				Body:   body,
			}

			res, err := client.DNS.Records.Edit(c.Context(), r.ID, editParams)
			if err != nil {
				return err
			}
			lastResult = *res
		}
	}
	if err := pager.Err(); err != nil {
		return err
	}

	if !foundRecords {
		createParams := dns.RecordNewParams{
			ZoneID: cloudflare.F(zoneID),
			Body: dns.RecordNewParamsBody{
				Name:    cloudflare.F(name),
				Type:    cloudflare.F(dns.RecordNewParamsBodyType(rtype)),
				Content: cloudflare.F(content),
				TTL:     cloudflare.F(dns.TTL(ttl)),
				Proxied: cloudflare.F(proxy),
			},
		}
		if priority != 0 {
			body := createParams.Body.(dns.RecordNewParamsBody)
			body.Priority = cloudflare.F(float64(priority))
			createParams.Body = body
		}

		res, err := client.DNS.Records.New(c.Context(), createParams)
		if err != nil {
			return err
		}
		lastResult = *res
	}

	output := [][]string{
		formatDNSRecord(lastResult),
	}
	writeTable(output, "ID", "Name", "Type", "Content", "TTL", "Proxiable", "Proxy")

	return nil
}
