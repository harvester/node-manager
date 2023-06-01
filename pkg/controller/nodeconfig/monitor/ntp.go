package monitor

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"

	"github.com/godbus/dbus/v5"
	gocommon "github.com/harvester/go-common"
	ctlnode "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
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

type NTPStatusAnnotation utils.NTPStatusAnnotation

type NTPMonitor struct {
	Context           context.Context
	MonitorName       string
	NodeName          string
	NodeNTPAnnotation *NTPStatusAnnotation

	NodeClient    ctlnode.NodeClient
	NodeConfigCtl ctlv1.NodeConfigController
	mtx           *sync.Mutex
}

func NewNTPMonitor(ctx context.Context, mtx *sync.Mutex, nodecfg ctlv1.NodeConfigController, nodes ctlnode.NodeController, nodeName, monitorName string) *NTPMonitor {
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
		mtx: mtx,
	}
}

func (monitor *NTPMonitor) startMonitor() {
	logrus.Infof("Manually update when bootup...")
	if err := monitor.updateAnnotation(); err != nil {
		// just log the error and continue, we could update the annotation later
		logrus.Errorf("Update annotation failed on bootup stage err: %v", err)
	}
	logrus.Infof("Start WatchDbus Signal...")

	iface := "org.freedesktop.DBus.Properties"
	objectPath := "/org/freedesktop/timedate1"
	go func() {
		gocommon.WatchDBusSignal(monitor.Context, iface, objectPath, monitor.handleTimedate1Signal)
	}()
}

func getNTPServersOnNode() string {
	if timesyncdConf := utils.GetTimesyncdConf(); timesyncdConf != nil {
		return timesyncdConf.Get("time.ntp").(string)
	}
	return ""
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

func (monitor *NTPMonitor) handleTimedate1Signal(signal *dbus.Signal) {
	if signal.Name == "org.freedesktop.DBus.Properties.PropertiesChanged" {
		logrus.Debugf("Debug Body: %+v", signal.Body)
		// check the signal.Body should at least have 2 elements.
		if len(signal.Body) < 2 || signal.Body[0].(string) != "org.freedesktop.timedate1" {
			logrus.Debugf("Do not handle this signal: %+v", signal.Body)
			return
		}
		for k, v := range signal.Body[1].(map[string]dbus.Variant) {
			switch k {
			case "NTP":
				if v.Value().(bool) {
					// from disable to enable, we could not wait another signal to update the sync status
					ntpSyncStatus := checkNTPSyncStatus()
					monitor.updateAnnotationNTPStatus(ntpSyncStatus)
				} else {
					monitor.updateAnnotationNTPStatus(Disabled)
				}
			case "NTPSynchronized":
				if v.Value().(bool) {
					monitor.updateAnnotationNTPStatus(Synced)
				} else {
					monitor.updateAnnotationNTPStatus(Unsynced)
				}
			default:
				logrus.Warnf("Do Not handle the un-supported key: %v, val: %v", k, v)
			}
		}

		err := monitor.updateAnnotation()
		if err != nil {
			logrus.Errorf("Update annotation failed with err: %v", err)
		}
	}
}

func (monitor *NTPMonitor) doAnnotationUpdate(annoValue *NTPStatusAnnotation) error {
	logrus.Infof("Node: %s, annotation update: %+v", monitor.NodeName, annoValue)
	monitor.mtx.Lock()
	defer monitor.mtx.Unlock()
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
		monitor.NodeClient.Update(nodeCpy)
	}
	return nil
}

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

func nodeNTPAnnotationEmpty(anno *NTPStatusAnnotation) bool {
	return anno.NTPSyncStatus == "" && anno.CurrentNTPServers == ""
}
