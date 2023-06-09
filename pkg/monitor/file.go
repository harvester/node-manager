package monitor

import (
	"context"

	gocommon "github.com/harvester/go-common"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/utils"
)

var monitorTargets = []string{utils.SystemdConfigPath}
var timesyncdConfigPath = utils.SystemdConfigPath + utils.TimesyncdConfigName

type ConfigFileMonitor struct {
	Context        context.Context
	MonitorName    string
	NodeName       string
	MonitorTargets []string

	NodeConfigCtl ctlv1.NodeConfigController
}

func NewConfigFileMonitor(ctx context.Context, nodecfg ctlv1.NodeConfigController, nodeName, monitorName string) *ConfigFileMonitor {
	return &ConfigFileMonitor{
		Context:        ctx,
		MonitorName:    monitorName,
		NodeConfigCtl:  nodecfg,
		NodeName:       nodeName,
		MonitorTargets: []string{},
	}
}

func (monitor *ConfigFileMonitor) startMonitor() {
	go func() {
		gocommon.WatchFileChange(monitor.Context, monitor.genericHandler, monitorTargets)
	}()
}

func (monitor *ConfigFileMonitor) handleNTPConfigChange() {
	wantedNTPServers := ""
	nodeconfig, err := monitor.NodeConfigCtl.Get(HarvesterNS, monitor.NodeName, metav1.GetOptions{})
	if err != nil {
		logrus.Warnf("Get NodeConfig fail, err: %v", err)
	} else {
		wantedNTPServers = nodeconfig.Spec.NTPConfig.NTPServers
		logrus.Debugf("Get the wanted NTP Servers: %s", wantedNTPServers)
	}
	currentNTPServers := getNTPServersOnNode()
	logrus.Debugf("Current NTP Servers: %s, Config NTP Servers %s", currentNTPServers, wantedNTPServers)

	if wantedNTPServers != "" && wantedNTPServers != currentNTPServers {
		logrus.Infof("Enqueue to make controller to update NTP Servers")
		monitor.NodeConfigCtl.Enqueue(nodeconfig.Namespace, nodeconfig.Name)
	}

}

func (monitor *ConfigFileMonitor) genericHandler(eventName string) {
	logrus.Debugf("Prepare to handle the event: %s", eventName)
	eventType := parserEventType(eventName)
	if eventType != "" {
		monitor.doGenericHandler(eventType)
	}
}

func (monitor *ConfigFileMonitor) doGenericHandler(eventType string) {
	switch eventType {
	case "NTP":
		monitor.handleNTPConfigChange()
	default:
		logrus.Errorf("unknown event type: %s", eventType)
	}
}

func parserEventType(path string) string {
	switch path {
	case timesyncdConfigPath:
		return "NTP"
	default:
		logrus.Errorf("unknown supported path: %s", path)
	}
	return ""
}
