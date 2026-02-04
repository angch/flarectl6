package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var originCARootCertCmd = &cobra.Command{
	Use:     "origin-ca-root-cert",
	Aliases: []string{"ocrc"},
	Short:   "Print Origin CA Root Certificate (in PEM format)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return originCARootCertificate(cmd)
	},
}

func init() {
	rootCmd.AddCommand(originCARootCertCmd)
	originCARootCertCmd.Flags().String("algorithm", "", "certificate algorithm ( ecc | rsa )")
	originCARootCertCmd.MarkFlagRequired("algorithm")
}

func originCARootCertificate(c *cobra.Command) error {
	if err := ensureClient(); err != nil {
		return err
	}

	alg, _ := c.Flags().GetString("algorithm")

	path := fmt.Sprintf("cert_req?certificate_chain_type=%s", alg)

	var res struct {
		Result struct {
			Certificate string `json:"certificate"`
		} `json:"result"`
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	err := client.Get(c.Context(), path, nil, &res)
	if err != nil {
		return err
	}

	if !res.Success {
		if len(res.Errors) > 0 {
			return fmt.Errorf("api error: %s", res.Errors[0].Message)
		}
		return fmt.Errorf("api error: unknown error")
	}

	fmt.Println(strings.TrimSpace(res.Result.Certificate))

	return nil
}
