package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints version of the app",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}
