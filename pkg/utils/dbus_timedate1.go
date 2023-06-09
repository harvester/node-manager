package utils

import (
	"github.com/godbus/dbus/v5"
	"github.com/sirupsen/logrus"
)

func GetTimeDate1PropertiesNTP() (bool, error) {
	conn, err := generateDBUSConnection()
	if err != nil {
		return false, err
	}

	obj := conn.Object(dbusTimedate1Iface, dbusTimedate1ObjectPath)

	var output bool
	err = obj.Call(DbusPropertiesGet(), 0, dbusTimedate1Iface, "NTP").Store(&output)
	if err != nil {
		logrus.Warnf("Get timedate1 properties failed. err: %v", err)
		return false, err
	}
	return output, nil
}

func GetTimeDate1PropertiesNTPSynchronized() (bool, error) {
	conn, err := generateDBUSConnection()
	if err != nil {
		return false, err
	}

	obj := conn.Object(dbusTimedate1Iface, dbusTimedate1ObjectPath)

	var output bool
	err = obj.Call(DbusPropertiesGet(), 0, dbusTimedate1Iface, "NTPSynchronized").Store(&output)
	if err != nil {
		logrus.Warnf("Get timedate1 properties failed. err: %v", err)
		return false, err
	}
	return output, nil
}

// Do not close this return connection because SystemBus() will return a shared connection
func generateDBUSConnection() (*dbus.Conn, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		logrus.Warnf("Init DBus connection failed. err: %v", err)
		return nil, err
	}

	return conn, nil
}
