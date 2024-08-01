package monitor

import (
	"context"
	"strings"
	"sync"
	"time"

	ctlnode "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	ctlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
)

// commonMonitor is the template for others monitoring
type moduleMonitor interface {
	startMonitor()
}

const (
	defaultInterval = 30 * time.Second
	HarvesterNS     = "harvester-system"
)

type Template struct {
	context      context.Context
	nodeName     string
	nodecfgctl   ctlv1.NodeConfigController
	nodesctl     ctlnode.NodeController
	cloudinitctl ctlv1.CloudInitController
	mtx          *sync.Mutex
}

func NewMonitorTemplate(ctx context.Context, mtx *sync.Mutex, nodecfg ctlv1.NodeConfigController, nodes ctlnode.NodeController, cloudinits ctlv1.CloudInitController, nodeName string) *Template {
	return &Template{
		context:      ctx,
		nodeName:     nodeName,
		nodecfgctl:   nodecfg,
		nodesctl:     nodes,
		cloudinitctl: cloudinits,
		mtx:          mtx,
	}
}

type Monitor struct {
	Context     context.Context
	MonitorName string
}

func InitServiceMonitor(template *Template, monitorName string) interface{} {
	// Implement service monitor here
	switch strings.ToLower(monitorName) {
	case "ntp":
		return NewNTPMonitor(template.context, template.mtx, template.nodecfgctl, template.nodesctl, template.nodeName, monitorName)
	case "configfile":
		return NewConfigFileMonitor(template.context, template.nodecfgctl, template.nodeName, monitorName)
	case "cloudinit":
		return NewCloudInitMonitor(template.context, monitorName, template.nodeName, template.cloudinitctl, template.nodesctl.Cache())
	default:
		return NewCommonMonitor(template.context, monitorName)
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
					logrus.Errorf("Failed to run the monitor %s: %v", monitor.MonitorName, err)
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
