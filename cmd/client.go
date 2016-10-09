package cmd

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	ss "shadowsocks/shadowsocks"
)

// RootClientCmd represents the base command when called without any subcommands
var RootClientCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: fmt.Sprintf("client for %s service", os.Args[0]),
	Run: func(cmd *cobra.Command, args []string) {
		setLog(cmd)
		log.Infof("This is %s binary for %s", currentFlag, os.Args[0])

		client := ss.NewClient()
		if err := client.ParseServerConfig(cfgFile); err != nil {
			log.Fatalln(err)
		}

		if err := client.Run(); err != nil {
			log.Fatalln(err)
		}
	},
}

// ExecuteClient adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteClient() {
	currentFlag = FlagClient // mark this is client

	if err := RootClientCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}

func init() {

	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootClientCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.shadowsocks.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootClientCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	RootClientCmd.PersistentFlags().String("server_addr", "", "Server address for shadowsocks service")
	RootClientCmd.PersistentFlags().String("server_port", "", "Server port for shadowsocks service")
	RootClientCmd.PersistentFlags().String("log_level", "info", "log level")

	viper.BindPFlag("server_addr", RootClientCmd.PersistentFlags().Lookup("server_addr"))
	viper.BindPFlag("server_port", RootClientCmd.PersistentFlags().Lookup("server_port"))
	viper.BindPFlag("log_level", RootClientCmd.PersistentFlags().Lookup("log_level"))

	// RootClientCmd.AddCommand(httpCmd)
	RootClientCmd.AddCommand(versionCmd)
	// RootClientCmd.AddCommand(shadowsocksCmd)
}
