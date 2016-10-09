package cmd

import (
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "reload config for service",
	Run: func(cmd *cobra.Command, args []string) {
		setLog(cmd)

		log.Infoln("Reload service config")
	},
}

func init() {
	RootServerCmd.AddCommand(reloadCmd)
}
