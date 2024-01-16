package utils

import (
	"fmt"

	"github.com/spf13/viper"
)

const (
	AnnotationNTP           = "node.harvesterhci.io/ntp-service"
	TimesyncdConfigName     = "timesyncd.conf"
	SystemdConfigPath       = "/host/etc/systemd/"
	DbusPropertiesIface     = "org.freedesktop.DBus.Properties"
	DbusTimedate1Name       = "org.freedesktop.timedate1"
	DbusTimesync1Name       = "org.freedesktop.timesync1.Manager"
	DbusTimedate1ObjectPath = "/org/freedesktop/timedate1"
	DbusTimesync1ObjectPath = "/org/freedesktop/timesync1"
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
	return []string{"NTP", "configFile", "cloudinit"}
}

func DbusPropertiesGet() string {
	return DbusPropertiesIface + ".Get"
}
