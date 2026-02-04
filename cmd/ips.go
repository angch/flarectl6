package cmd

import (
	"fmt"

	"github.com/cloudflare/cloudflare-go/v6/ips"
	"github.com/spf13/cobra"
)

var ipsCmd = &cobra.Command{
	Use:     "ips",
	Short:   "Print Cloudflare IP ranges",
	Aliases: []string{"i"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return listIPs(cmd)
	},
}

func init() {
	rootCmd.AddCommand(ipsCmd)
	ipsCmd.Flags().String("ip-type", "all", "type of IPs ( ipv4 | ipv6 | all )")
	ipsCmd.Flags().Bool("ip-only", false, "show only addresses")
}

func listIPs(c *cobra.Command) error {
	var ipService *ips.IPService
	if client != nil {
		ipService = client.IPs
	} else {
		ipService = ips.NewIPService()
	}

	res, err := ipService.List(c.Context(), ips.IPListParams{})
	if err != nil {
		return err
	}

	ipType, _ := c.Flags().GetString("ip-type")
	ipOnly, _ := c.Flags().GetBool("ip-only")

	if ipType == "all" {
		printIPs("ipv4", ipOnly, res)
		printIPs("ipv6", ipOnly, res)
	} else {
		printIPs(ipType, ipOnly, res)
	}

	return nil
}

func printIPs(ipType string, showMsgType bool, res *ips.IPListResponse) {
	var ipv4, ipv6 []string

	union := res.AsUnion()
	switch u := union.(type) {
	case ips.IPListResponsePublicIPIPs:
		ipv4 = u.IPV4CIDRs
		ipv6 = u.IPV6CIDRs
	case ips.IPListResponsePublicIPIPsJDCloud:
		ipv4 = u.IPV4CIDRs
		ipv6 = u.IPV6CIDRs
	}

	var ranges []string
	switch ipType {
	case "ipv4":
		ranges = ipv4
		if showMsgType {
			fmt.Println("IPv4 ranges:")
		}
	case "ipv6":
		ranges = ipv6
		if showMsgType {
			fmt.Println("IPv6 ranges:")
		}
	}

	for _, r := range ranges {
		fmt.Println(" ", r)
	}
}
