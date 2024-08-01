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
	gocommon "github.com/harvester/go-common"
	ctlnode "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/utils"
)

const (
	AnnotationNTP = "harvesterhci.io/ntp-service"
	Disabled      = "disabled"
	Unsynced      = "unsynced"
	Synced        = "synced"
	Unknown       = "unknown"
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
		gocommon.WatchDBusSignal(monitor.Context, utils.DbusPropertiesIface, utils.DbusTimedate1ObjectPath, monitor.handleTimedate1Signal)
	}()
	go func() {
		gocommon.WatchDBusSignal(monitor.Context, utils.DbusPropertiesIface, utils.DbusTimesync1ObjectPath, monitor.handleTimesync1Signal)
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
		logrus.Debugf("NTP does not enable.")
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
	if monitor.NodeNTPAnnotation.NTPSyncStatus != checkNTPSyncStatus() {
		return nil
	}
	logrus.Infof("Prepare update the NTPSync Status...")
	monitor.prepareUpdateAnnotation(true)
	return nil
}

func (monitor *NTPMonitor) handleTimesync1Signal(signal *dbus.Signal) {
	if signal.Name == "org.freedesktop.DBus.Properties.PropertiesChanged" {
		logrus.Debugf("Debug Body: %+v", signal.Body)
		// check the signal.Body should at least have 2 elements.
		if len(signal.Body) < 2 || signal.Body[0].(string) != utils.DbusTimesync1Name {
			logrus.Debugf("Do not handle this signal: %+v", signal.Body)
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
		logrus.Debugf("Debug Body: %+v", signal.Body)
		// check the signal.Body should at least have 2 elements.
		if len(signal.Body) < 2 || signal.Body[0].(string) != utils.DbusTimedate1Name {
			logrus.Debugf("Do not handle this signal: %+v", signal.Body)
			return
		}
		for k, v := range signal.Body[1].(map[string]dbus.Variant) {
			switch k {
			case "NTP":
				ntpEnable := v.Value().(bool)
				if err := monitor.prepareUpdateAnnotation(ntpEnable); err != nil {
					logrus.Errorf("Failed to update annotation with err: %v", err)
				}
				if ntpEnable {
					monitor.ticker.Reset(DefaultNTPCheckInterval)
				} else {
					monitor.ticker.Reset(MAXNTPCheckInterval)
				}
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
		logrus.Warnf("Get Node fail, skip this round NTP check...")
		return err
	}

	bytes, err := json.Marshal(annoValue)
	if err != nil {
		logrus.Errorf("Marshal annotation value fail, skip this round NTP check...")
		return err
	}

	nodeCpy := node.DeepCopy()
	nodeCpy.Annotations[utils.AnnotationNTP] = string(bytes)
	if !reflect.DeepEqual(node, nodeCpy) {
		logrus.Infof("Try to update with Node: %s, annotation update: %+v", monitor.NodeName, annoValue)
		monitor.NodeClient.Update(nodeCpy)
	}
	return nil
}

func (monitor *NTPMonitor) updateLatestNodeNTPAnnotation() error {
	node, err := monitor.NodeClient.Get(monitor.NodeName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("Get node %s failed. err: %v", monitor.NodeName, err)
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

	monitor.NodeNTPAnnotation.CurrentNTPServers = ntpValue.CurrentNTPServers
	monitor.NodeNTPAnnotation.NTPSyncStatus = ntpValue.NTPSyncStatus
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
		return monitor.doAnnotationUpdate(monitor.NodeNTPAnnotation)
	}
	return monitor.doAnnotationUpdate(monitor.NodeNTPAnnotation)
}

func (monitor *NTPMonitor) updateAnnotationNTPStatus(status string) {
	monitor.NodeNTPAnnotation.NTPSyncStatus = status
}

func (monitor *NTPMonitor) postponeTheNTPSyncStatusPolling(message NTPMessage) {
	logrus.Debugf("NTPMessage: %+v", message)
	now := time.Now()
	// microsecond * 1000 to nanosecond
	lastTimestamp := time.Unix(0, int64(message.DestinationTimestamp)*1000)
	if now.Sub(lastTimestamp) > NTPSyncTimeout {
		logrus.Warnf("NTP Server looks not responsible, let's running the NTPSyncStatus check.")
		return
	}
	monitor.ticker.Reset(DefaultNTPCheckInterval)
}

func nodeNTPAnnotationEmpty(anno *NTPStatusAnnotation) bool {
	return anno.NTPSyncStatus == "" && anno.CurrentNTPServers == ""
}
