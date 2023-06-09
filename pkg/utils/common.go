package utils

import (
	"fmt"

	"github.com/spf13/viper"
)

const (
	AnnotationNTP           = "node.harvesterhci.io/ntp-service"
	TimesyncdConfigName     = "timesyncd.conf"
	SystemdConfigPath       = "/host/etc/systemd/"
	dbusPropertiesIface     = "org.freedesktop.DBus.Properties"
	dbusTimedate1Iface      = "org.freedesktop.timedate1"
	dbusTimedate1ObjectPath = "/org/freedesktop/timedate1"
)

type NTPStatusAnnotation struct {
	NTPSyncStatus     string `json:"ntpSyncStatus"`
	CurrentNTPServers string `json:"currentNtpServers"`
}

func GetTimesyncdConf() (*viper.Viper, error) {
	timesyncdConf := viper.New()
	timesyncdConf.SetConfigName(TimesyncdConfigName)
	timesyncdConf.SetConfigType("ini")
	timesyncdConf.AddConfigPath(SystemdConfigPath)
	err := timesyncdConf.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("reading config file error: %v", err)
	}
	return timesyncdConf, nil
}

func GetToMonitorServices() []string {
	return []string{"NTP", "configFile"}
}

func DbusPropertiesGet() string {
	return dbusPropertiesIface + ".Get"
}
