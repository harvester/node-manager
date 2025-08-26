package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/harvester/go-common/sys"
	ctlnode "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/utils"
)

const (
	Disabled = "disabled"
	Unsynced = "unsynced"
	Synced   = "synced"
)

// NTPSyncTimeout is 45 minutes because the NTPUpdate should be triggered around 30 minutes
var NTPSyncTimeout = 45 * time.Minute
var DefaultNTPCheckInterval = 15 * time.Minute
var MAXNTPCheckInterval = 24 * 365 * time.Hour

type NTPStatusAnnotation utils.NTPStatusAnnotation

type NTPMonitor struct {
	Context           context.Context
	MonitorName       string
	NodeName          string
	NodeNTPAnnotation *NTPStatusAnnotation

	NodeClient    ctlnode.NodeClient
	NodeConfigCtl ctlv1.NodeConfigController
	mtx           *sync.Mutex
	ticker        *time.Ticker
}

type NTPMessage struct {
	Leap                 uint32
	Version              uint32
	Mode                 uint32
	Stratum              uint32
	Precision            int32
	RootDelay            uint64
	RootDispersion       uint64
	Reference            interface{}
	OriginateTimestamp   uint64
	ReceiveTimestamp     uint64
	TransmitTimestamp    uint64
	DestinationTimestamp uint64
	Ignored              bool
	PacketCount          uint64
	Jitter               uint64
}

func NewNTPMonitor(ctx context.Context, mtx *sync.Mutex, nodecfg ctlv1.NodeConfigController, nodes ctlnode.NodeController, nodeName, monitorName string) *NTPMonitor {
	ticker := time.NewTicker(DefaultNTPCheckInterval)
	return &NTPMonitor{
		Context:       ctx,
		MonitorName:   monitorName,
		NodeConfigCtl: nodecfg,
		NodeClient:    nodes,
		NodeName:      nodeName,
		NodeNTPAnnotation: &NTPStatusAnnotation{
			NTPSyncStatus:     "",
			CurrentNTPServers: "",
		},
		mtx:    mtx,
		ticker: ticker,
	}
}

func (monitor *NTPMonitor) startMonitor() {
	logrus.Infof("Manually update when bootup...")
	if err := monitor.updateAnnotation(); err != nil {
		// just log the error and continue, we could update the annotation later
		logrus.Errorf("Update annotation failed on bootup stage err: %v", err)
	}
	logrus.Infof("Start WatchDbus Signal...")

	go func() {
		sys.WatchDBusSignal(monitor.Context, utils.DbusPropertiesIface, utils.DbusTimedate1ObjectPath, monitor.handleTimedate1Signal)
	}()
	go func() {
		sys.WatchDBusSignal(monitor.Context, utils.DbusPropertiesIface, utils.DbusTimesync1ObjectPath, monitor.handleTimesync1Signal)
	}()
	go func() {
		defer monitor.ticker.Stop()
		for {
			select {
			case <-monitor.ticker.C:
				if err := monitor.updateNTPSyncStatus(); err != nil {
					logrus.Errorf("Failed to update NTPSyncStatus: %v", err)
				}
			case <-monitor.Context.Done():
				return
			}
		}
	}()
}

func getNTPServersOnNode() string {
	timesyncdConf, err := utils.GetTimesyncdConf()
	if err != nil {
		return err.Error()
	}
	ntpServersString := ""
	if slices.Contains(timesyncdConf.AllKeys(), "time.ntp") {
		ntpServersString = timesyncdConf.Get("time.ntp").(string)
	}
	return ntpServersString
}

func checkNTPSyncStatus() string {
	if !checkNTPEnable() {
		logrus.Debugf("NTP is not enabled.")
		return Disabled
	}

	if !checkNTPSynced() {
		logrus.Debugf("NTP was not synced.")
		return Unsynced
	}
	return Synced
}

func checkNTPEnable() bool {
	logrus.Debugf("Checking NTP Status on node ...")
	output, err := utils.GetTimeDate1PropertiesNTP()
	if err != nil {
		logrus.Warnf("Command failed with err: %v, skip this round.", err)
		return false
	}

	return output
}

func checkNTPSynced() bool {
	logrus.Debugf("Checking NTP NTPSynchronized on node ...")
	output, err := utils.GetTimeDate1PropertiesNTPSynchronized()
	if err != nil {
		logrus.Warnf("Command failed with err: %v, skip this round.", err)
		return false
	}

	return output
}

func generateAnnotationValue(syncStatus, current string) *NTPStatusAnnotation {
	return &NTPStatusAnnotation{
		NTPSyncStatus:     syncStatus,
		CurrentNTPServers: current,
	}
}

func (monitor *NTPMonitor) updateNTPSyncStatus() error {
	if monitor.NodeNTPAnnotation.NTPSyncStatus == checkNTPSyncStatus() {
		return nil
	}
	logrus.Infof("Prepare update the NTPSync Status...")
	return monitor.prepareUpdateAnnotation(true)
}

func (monitor *NTPMonitor) handleTimesync1Signal(signal *dbus.Signal) {
	if signal.Name == "org.freedesktop.DBus.Properties.PropertiesChanged" {
		logrus.WithFields(logrus.Fields{
			"msgLength": len(signal.Body),
		}).Debugf("Signal body: %+v", signal.Body)
		// check the signal.Body should at least have 2 elements.
		// Example:
		// [org.freedesktop.timesync1.Manager map[NTPMessage:@(uuuuittayttttbtt) [0, 4, 4, 2, -24, 4730, 122, [0x83, 0xbc, 0x3, 0xde], 1740471296747002, 1740471573952418, 1740471573952551, 1740471296760460, false, 1, 0]] ]]
		if len(signal.Body) < 2 || signal.Body[0].(string) != utils.DbusTimesync1Name {
			logrus.WithFields(logrus.Fields{
				"interface":         signal.Body[0].(string),
				"expectedInterface": utils.DbusTimesync1Name,
			}).Debug("Do not handle this signal")
			return
		}
		for k, v := range signal.Body[1].(map[string]dbus.Variant) {
			switch k {
			case "NTPMessage":
				var ntpMessage NTPMessage
				err := dbus.Store(v.Value().([]interface{}), &ntpMessage.Leap, &ntpMessage.Version, &ntpMessage.Mode, &ntpMessage.Stratum, &ntpMessage.Precision,
					&ntpMessage.RootDelay, &ntpMessage.RootDispersion, &ntpMessage.Reference, &ntpMessage.OriginateTimestamp, &ntpMessage.ReceiveTimestamp,
					&ntpMessage.TransmitTimestamp, &ntpMessage.DestinationTimestamp, &ntpMessage.Ignored, &ntpMessage.PacketCount, &ntpMessage.Jitter)
				if err != nil {
					logrus.Errorf("Failed to convert the dbus.Variant to NTPMessage: %v", err)
				}
				monitor.postponeTheNTPSyncStatusPolling(ntpMessage)
				if err := monitor.prepareUpdateAnnotation(true); err != nil {
					logrus.Errorf("Failed to update annotation with err: %v", err)
				}
			default:
				logrus.Warnf("Do Not handle the un-supported key: %v, val: %v", k, v)
			}
		}
	}
}

func (monitor *NTPMonitor) handleTimedate1Signal(signal *dbus.Signal) {
	if signal.Name == "org.freedesktop.DBus.Properties.PropertiesChanged" {
		logrus.WithFields(logrus.Fields{
			"msgLength": len(signal.Body),
		}).Debugf("Signal body: %+v", signal.Body)
		// check the signal.Body should at least have 2 elements.
		// Example:
		// [org.freedesktop.timedate1 map[NTP:false]]
		if len(signal.Body) < 2 || signal.Body[0].(string) != utils.DbusTimedate1Name {
			logrus.WithFields(logrus.Fields{
				"interface":         signal.Body[0].(string),
				"expectedInterface": utils.DbusTimedate1Name,
			}).Debug("Do not handle this signal")
			return
		}
		for k, v := range signal.Body[1].(map[string]dbus.Variant) {
			switch k {
			case "NTP":
				ntpEnable := v.Value().(bool)
				if err := monitor.prepareUpdateAnnotation(ntpEnable); err != nil {
					logrus.Errorf("Failed to update annotation with err: %v", err)
				}
				duration := MAXNTPCheckInterval
				if ntpEnable {
					duration = DefaultNTPCheckInterval
				}
				logrus.WithFields(logrus.Fields{
					"ntpEnable": ntpEnable,
					"duration":  duration.String(),
				}).Debug("Reset the ticker")
				monitor.ticker.Reset(duration)
			default:
				logrus.Warnf("Do Not handle the un-supported key: %v, val: %v", k, v)
			}
		}
	}
}

func (monitor *NTPMonitor) doAnnotationUpdate(annoValue *NTPStatusAnnotation) error {
	logrus.Debugf("Node: %s, annotation update: %+v", monitor.NodeName, annoValue)
	node, err := monitor.NodeClient.Get(monitor.NodeName, metav1.GetOptions{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"node": monitor.NodeName,
		}).Errorf("Failed to get node, skip this round NTP check: %+v", err)
		return err
	}

	bytes, err := json.Marshal(annoValue)
	if err != nil {
		logrus.Errorf("Marshal annotation value fail, skip this round NTP check: %+v", err)
		return err
	}

	nodeCpy := node.DeepCopy()
	nodeCpy.Annotations[utils.AnnotationNTP] = string(bytes)
	if !reflect.DeepEqual(node, nodeCpy) {
		logrus.Infof("Try to update with Node: %s, annotation update: %+v", monitor.NodeName, annoValue)
		if _, err := monitor.NodeClient.Update(nodeCpy); err != nil {
			return err
		}
	}
	return nil
}

func (monitor *NTPMonitor) updateLatestNodeNTPAnnotation() error {
	node, err := monitor.NodeClient.Get(monitor.NodeName, metav1.GetOptions{})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"node": monitor.NodeName,
		}).Errorf("Failed to get node: %+v", err)
		return err
	}

	if _, found := node.Annotations[utils.AnnotationNTP]; !found {
		logrus.Debugf("Should not failure here, we only failed on first round!")
		return fmt.Errorf("ntp annotation not found")
	}
	annoNTPValue := node.Annotations[utils.AnnotationNTP]
	var ntpValue NTPStatusAnnotation
	if err := json.Unmarshal([]byte(annoNTPValue), &ntpValue); err != nil {
		logrus.Errorf("Unmarshal annotation value failed. err: %v", err)
		return err
	}
	logrus.Debugf("Current annotation value: %+v", ntpValue)

	monitor.updateAnnotationNTPServers(ntpValue.CurrentNTPServers)
	monitor.updateAnnotationNTPStatus(ntpValue.NTPSyncStatus)

	return nil
}

func (monitor *NTPMonitor) prepareUpdateAnnotation(ntpEnable bool) error {
	monitor.mtx.Lock()
	defer monitor.mtx.Unlock()
	if err := monitor.updateLatestNodeNTPAnnotation(); err != nil {
		logrus.Errorf("Failed to update latest node NTP annotation with err: %v", err)
		return err
	}

	if ntpEnable {
		ntpSyncStatus := checkNTPSyncStatus()
		monitor.updateAnnotationNTPStatus(ntpSyncStatus)
	} else {
		monitor.updateAnnotationNTPStatus(Disabled)
	}
	err := monitor.updateAnnotation()
	if err != nil {
		logrus.Errorf("Update annotation failed with err: %v", err)
		return err
	}
	return nil
}

// updateAnnotation only called directly on the init, we need lock with other caller.
func (monitor *NTPMonitor) updateAnnotation() error {
	if nodeNTPAnnotationEmpty(monitor.NodeNTPAnnotation) {
		logrus.Debugf("First update due to empty annotation.")
		ntpSyncStatus := checkNTPSyncStatus()
		currentNtpServers := getNTPServersOnNode()
		monitor.NodeNTPAnnotation = generateAnnotationValue(ntpSyncStatus, currentNtpServers)
	}
	return monitor.doAnnotationUpdate(monitor.NodeNTPAnnotation)
}

func (monitor *NTPMonitor) updateAnnotationNTPStatus(status string) {
	monitor.NodeNTPAnnotation.NTPSyncStatus = status
}

func (monitor *NTPMonitor) updateAnnotationNTPServers(servers string) {
	monitor.NodeNTPAnnotation.CurrentNTPServers = servers
}

func (monitor *NTPMonitor) postponeTheNTPSyncStatusPolling(message NTPMessage) {
	logrus.Debugf("NTPMessage: %+v", message)
	now := time.Now()
	// microsecond * 1000 to nanosecond
	lastTimestamp := time.Unix(0, int64(message.DestinationTimestamp)*1000) //nolint:gosec
	if now.Sub(lastTimestamp) > NTPSyncTimeout {
		logrus.Warnf("NTP Server looks not responsible, let's running the NTPSyncStatus check.")
		return
	}
	logrus.WithFields(logrus.Fields{
		"duration": DefaultNTPCheckInterval.String(),
	}).Debug("Reset the ticker")
	monitor.ticker.Reset(DefaultNTPCheckInterval)
}

func nodeNTPAnnotationEmpty(anno *NTPStatusAnnotation) bool {
	return anno.NTPSyncStatus == "" && anno.CurrentNTPServers == ""
}
