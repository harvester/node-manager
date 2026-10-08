package utils

import (
	"fmt"
	"slices"
	"strings"

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

// NormalizeNTPServers splits a space-separated NTPServers value (hostnames
// and/or IPv4/IPv6 literals) and drops duplicate entries, preserving the
// order of first occurrence. systemd-timesyncd's NTP= contacts entries in
// the given order until one responds, so entries must never be reordered,
// only deduplicated.
func NormalizeNTPServers(servers string) string {
	if servers == "" {
		return ""
	}

	fields := strings.Split(servers, " ")
	deduped := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		if !slices.Contains(deduped, field) {
			deduped = append(deduped, field)
		}
	}
	return strings.Join(deduped, " ")
}
