package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:     "user",
	Short:   "User information",
	Aliases: []string{"u"},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureClient()
	},
}

var userInfoCmd = &cobra.Command{
	Use:     "info",
	Short:   "User details",
	Aliases: []string{"i"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return userInfo(cmd)
	},
}

var userUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update user details",
	Aliases: []string{"u"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return userUpdate(cmd)
	},
}

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userInfoCmd)
	userCmd.AddCommand(userUpdateCmd)
}

func userInfo(c *cobra.Command) error {
	user, err := client.User.Get(c.Context())
	if err != nil {
		return err
	}

	output := [][]string{
		{
			user.ID,
			"", // Email not available in v6
			"", // Username not available in v6
			strings.TrimSpace(user.FirstName + " " + user.LastName),
			formatBool(user.TwoFactorAuthenticationEnabled),
		},
	}

	writeTable(output, "ID", "Email", "Username", "Name", "2FA")
	return nil
}

func userUpdate(c *cobra.Command) error {
	// This command is a no-op in the legacy flarectl tool.
	// We preserve this behavior for compatibility.
	return nil
}
