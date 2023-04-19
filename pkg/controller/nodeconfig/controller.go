package nodeconfig

import (
	"context"
	"fmt"
	"strings"
	"sync"

	ctlnode "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	nodeconfigv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/controller/nodeconfig/monitor"
	ctlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
)

const (
	HandlerName = "harvester-node-config-controller"
	ScpoeGlobal = "Global"
)

var toMonitorServices = []string{"NTP", "configFile"}

// ensure every monitor shutdown safely
var WaitGroup sync.WaitGroup

type Controller struct {
	ctx      context.Context
	NodeName string

	NodeConfigs      ctlv1.NodeConfigController
	NodeConfigsCache ctlv1.NodeConfigCache

	WaitGroup *sync.WaitGroup
}

func Register(ctx context.Context, nodeName string, nodecfg ctlv1.NodeConfigController, nodes ctlnode.NodeController) (*Controller, error) {
	ctl := &Controller{
		ctx:              ctx,
		NodeName:         nodeName,
		NodeConfigs:      nodecfg,
		NodeConfigsCache: nodecfg.Cache(),
		WaitGroup:        &WaitGroup,
	}

	monitorNnumbers := len(toMonitorServices)

	monitorModules := make([]interface{}, 0, monitorNnumbers)
	for _, serviceName := range toMonitorServices {
		monitorModule := monitor.InitServiceMonitor(ctx, nodecfg, nodes, nodeName, serviceName)
		monitorModules = append(monitorModules, monitorModule)
	}
	monitor.StartsAllMonitors(monitorModules)

	ctl.NodeConfigs.OnChange(ctx, HandlerName, ctl.OnNodeConfigChange)
	ctl.NodeConfigs.OnRemove(ctx, HandlerName, ctl.OnNodeConfigRemove)

	return ctl, nil
}

func (c *Controller) OnNodeConfigChange(key string, nodecfg *nodeconfigv1.NodeConfig) (*nodeconfigv1.NodeConfig, error) {
	confName := strings.Split(key, "/")[1]
	if nodecfg == nil || confName != c.NodeName || nodecfg.DeletionTimestamp != nil {
		logrus.Infof("Skip this round (OnChange) with NodeConfigs (%s): %+v", confName, nodecfg)
		return nil, nil
	}

	return nil, nil
}

func (c *Controller) OnNodeConfigRemove(key string, nodecfg *nodeconfigv1.NodeConfig) (*nodeconfigv1.NodeConfig, error) {
	if nodecfg == nil || nodecfg.DeletionTimestamp == nil {
		logrus.Infof("Skip this round (OnRemove) with NodeConfigs :%+v", nodecfg)
		return nil, nil
	}

	confName := strings.Split(key, "/")[1]
	if confName != c.NodeName {
		return nil, fmt.Errorf("node name %s is not matched", confName)
	}

	return nil, nil
}
