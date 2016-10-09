package cmd

import (
	"os"
	"strings"

	"shadowsocks/utils"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func setLog(cmd *cobra.Command) {
	logFilePath := viper.GetString("log_file")
	if len(strings.TrimSpace(logFilePath)) > 0 {
		if err := utils.CreateFileDir(logFilePath); err != nil {
			log.Fatalf("Create dir of log file error:%v", err)
		}

		f, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Fatalf("Open log file error:%v", err)
		}
		log.SetOutput(f)

	}

	logLevel, err := cmd.Flags().GetString("log_level")
	if err != nil {
		log.Fatal(err)
	}

	switch logLevel {
	case "debug":
		log.Infoln("Start as debug mode")
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}

}
