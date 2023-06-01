package utils

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	AnnotationNTP       = "harvesterhci.io/ntp-service"
	TimesyncdConfigName = "timesyncd.conf"
	SystemdConfigPath   = "/host/etc/systemd/"
)

type NTPStatusAnnotation struct {
	NTPSyncStatus     string `json:"ntpSyncStatus"`
	CurrentNTPServers string `json:"currentNtpServers"`
}

func GetTimesyncdConf() *viper.Viper {
	timesyncdConf := viper.New()
	timesyncdConf.SetConfigName(TimesyncdConfigName)
	timesyncdConf.SetConfigType("ini")
	timesyncdConf.AddConfigPath(SystemdConfigPath)
	err := timesyncdConf.ReadInConfig() // Find and read the config file
	if err != nil {                     // Handle errors reading the config file
		logrus.Errorf("Reading config file error: %v", err)
		return nil
	}
	return timesyncdConf
}
