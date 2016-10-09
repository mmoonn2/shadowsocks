package cmd

import (
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	// FlagServer server flag
	FlagServer = "Server"
	// FlagClient client flag
	FlagClient = "Client"
)

var (
	cfgFile     string
	currentFlag string
)

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".shadowsocks") // name of config file (without extension)
	viper.AddConfigPath("$HOME")        // adding home directory as first search path
	viper.AutomaticEnv()                // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		log.Warnf("Read config file error:%v, will use default config", err)
	}
}
