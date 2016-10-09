package cmd

import (
	"fmt"
	"os"

	ss "shadowsocks/shadowsocks"
	"shadowsocks/utils"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootServerCmd represents the base command when called without any subcommands
var RootServerCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: fmt.Sprintf("server for %s service", os.Args[0]),
	Run: func(cmd *cobra.Command, args []string) {
		setLog(cmd)
		fmt.Printf("This is %s binary for %s \n", currentFlag, os.Args[0])
		fmt.Println("config:", cfgFile)
		fmt.Println("log_level:", viper.GetString("log_level"))

		pm := ss.NewServer(cfgFile)
		if err := pm.Start(); err != nil {
			log.Fatalln(err)
		}
		// move to outside
		utils.HandleSignal(pm.Reload, func() {
			log.Infoln("Close")
		})
	},
}

// ExecuteServer adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteServer() {
	currentFlag = FlagServer // mark this is server

	if err := RootServerCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}

func init() {

	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootServerCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.shadowsocks.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// RootServerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	RootServerCmd.PersistentFlags().String("bind_addr", "", "Server address for shadowsocks service")
	RootServerCmd.PersistentFlags().String("bind_port", "", "Server port for shadowsocks service")
	RootServerCmd.PersistentFlags().String("log_level", "info", "log level")

	// viper.BindPFlag("config", RootServerCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("bind_addr", RootServerCmd.PersistentFlags().Lookup("bind_addr"))
	viper.BindPFlag("bind_port", RootServerCmd.PersistentFlags().Lookup("bind_port"))
	viper.BindPFlag("log_level", RootServerCmd.PersistentFlags().Lookup("log_level"))

	RootServerCmd.AddCommand(versionCmd)
}
