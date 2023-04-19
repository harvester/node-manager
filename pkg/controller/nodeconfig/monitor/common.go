package monitor

import (
	"context"
	"strings"
	"time"

	ctlnode "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	ctlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
)

// commonMonitor is the template for others monitoring
type moduleMonitor interface {
	startMonitor()
}

const (
	defaultInterval   = 30 * time.Second
	HarvesterNS       = "harvester-system"
	systemdConfigPath = "/host/etc/systemd/"
)

type Monitor struct {
	Context     context.Context
	MonitorName string
}

func InitServiceMonitor(ctx context.Context, nodecfg ctlv1.NodeConfigController, nodes ctlnode.NodeController, name, monitorName string) interface{} {
	// Implement service monitor here
	switch strings.ToLower(monitorName) {
	case "ntp":
		return NewNTPMonitor(ctx, nodecfg, nodes, name, monitorName)
	case "configfile":
		return NewConfigFileMonitor(ctx, nodecfg, name, monitorName)
	default:
		return NewCommonMonitor(ctx, monitorName)
	}
}

func NewCommonMonitor(ctx context.Context, name string) *Monitor {
	return &Monitor{
		Context:     ctx,
		MonitorName: name,
	}
}

func (monitor *Monitor) startMonitor() {
	logrus.Warnf("Do not use Default monitoring, do no-op here.")
	go func() {
		ticker := time.NewTicker(defaultInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := monitor.runMonitor(); err != nil {
					logrus.Errorf("Failed to rescan block devices on node %s: %v", monitor.MonitorName, err)
				}
			case <-monitor.Context.Done():
				return
			}
		}
	}()
}

func (monitor *Monitor) runMonitor() error {
	// no-op here
	logrus.Infof("Default monitor would do no-op.")
	return nil
}

func StartsAllMonitors(monitors []interface{}) {
	for _, rawMonitor := range monitors {
		monitor := rawMonitor.(moduleMonitor)
		monitor.startMonitor()
	}
}
