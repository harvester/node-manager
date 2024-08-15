package nodeconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/harvester/go-common/common"
	ctlnode "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	nodeconfigv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/controller/nodeconfig/config"
	ctlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
)

const (
	HandlerName             = "harvester-node-config-controller"
	ConfigApplied           = "Applied"
	ConfigAppliedAnnotation = "AppliedConfig"
)

type Controller struct {
	ctx      context.Context
	NodeName string

	NodeConfigs      ctlv1.NodeConfigController
	NodeConfigsCache ctlv1.NodeConfigCache
	NodeClient       ctlnode.NodeController
	mtx              *sync.Mutex
}

func Register(ctx context.Context, nodeName string, nodecfg ctlv1.NodeConfigController, nodes ctlnode.NodeController, mtx *sync.Mutex) (*Controller, error) {
	ctl := &Controller{
		ctx:              ctx,
		NodeName:         nodeName,
		NodeConfigs:      nodecfg,
		NodeConfigsCache: nodecfg.Cache(),
		NodeClient:       nodes,
		mtx:              mtx,
	}

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

	// V2 Data Engine related handling.  This is intentionally not bothering
	// to check whether the engine is already enabled or not, it runs on any
	// change to the node config, even if that change wasn't related to the
	// longhorn settings.  This is mostly harmless, because if the engine is
	// already in the state about to be applied, re-applying that state is
	// effectively a no-op, and I'd rather keep the code simple than add
	// annotations for whether or not we already enabled the engine.
	// The one wrinkle is that when allocating (or deallocating) hugepages,
	// the kubelet needs to be restarted to pick up the change and reflect
	// that in node.status.capacity.hugepages-2Mi, so that Longhorn can
	// query that value when lhs/v2-data-engine is set to true.  This restart
	// logic is handled inside EnableV2DataEngine() and DisableV2DataEngine().
	if nodecfg.Spec.LonghornConfig != nil && nodecfg.Spec.LonghornConfig.EnableV2DataEngine {
		if err := config.EnableV2DataEngine(); err != nil {
			logrus.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("Failed to enable V2 Data Engine")
		}
	} else {
		if err := config.DisableV2DataEngine(); err != nil {
			logrus.WithFields(logrus.Fields{
				"err": err.Error(),
			}).Error("Failed to disable V2 Data Engine")
		}
	}

	// NTP related handling
	appliedConfig := nodecfg.ObjectMeta.Annotations[ConfigAppliedAnnotation]
	ntpConfigHandler := config.NewNTPConfigHandler(c.mtx, c.NodeClient, confName, nodecfg.Spec.NTPConfig, appliedConfig)
	updated, err := ntpConfigHandler.DoNTPUpdate(len(nodecfg.Status.NTPConditions) == 0)
	if err != nil {
		logrus.Errorf("Update NTP config fail. err: %v", err)
		return nil, err
	}
	if updated {
		if err := ntpConfigHandler.RestartService(); err != nil {
			logrus.Errorf("Restart systemd-timesyncd fail. err: %v", err)
			return nil, err
		}
		if err := ntpConfigHandler.UpdateNTPConfigPersistence(); err != nil {
			logrus.Errorf("Update NTP config to OEM fail. err: %v", err)
			return nil, err
		}

		if err := ntpConfigHandler.UpdateNodeNTPAnnotation(); err != nil {
			logrus.Errorf("Update Node NTP annotation fail. err: %v", err)
			return nil, err
		}
		annoValue := generateAnnotationValue(nodecfg.Spec.NTPConfig.NTPServers)
		bytes, err := json.Marshal(annoValue)
		if err != nil {
			logrus.Errorf("Marshal annotation value fail, err: %v", err)
			return nil, err
		}

		nodecfg, err := config.UpdateNTPConfigApplied(c.NodeConfigs, nodecfg)
		if err != nil {
			logrus.Errorf("Update NodeConfig Status fail, err: %v", err)
			return nil, err
		}

		nodecfgCpy := nodecfg.DeepCopy()
		if nodecfgCpy.ObjectMeta.Annotations == nil {
			nodecfgCpy.ObjectMeta.Annotations = make(map[string]string)
		}
		nodecfgCpy.ObjectMeta.Annotations[ConfigAppliedAnnotation] = string(bytes)
		if !reflect.DeepEqual(nodecfg, nodecfgCpy) {
			return c.NodeConfigs.Update(nodecfgCpy)
		}
	} else {
		logrus.Infof("NTP config is not changed")
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

	logrus.Infof("Node config is removed, rollback and remove persistent NTP config")
	if err := config.NTPConfigRollback(); err != nil {
		logrus.Errorf("Rollback NTP config fail. err: %v", err)
		c.NodeConfigs.EnqueueAfter(nodecfg.Namespace, nodecfg.Name, enqueueJitter())
		return nil, err
	}
	if err := config.RemovePersistentNTPConfig(); err != nil {
		logrus.Errorf("Remove persistent NTP config fail. err: %v", err)
		c.NodeConfigs.EnqueueAfter(nodecfg.Namespace, nodecfg.Name, enqueueJitter())
		return nil, err
	}
	if err := config.DisableV2DataEngine(); err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Error("Failed to disable V2 Data Engine")
		c.NodeConfigs.EnqueueAfter(nodecfg.Namespace, nodecfg.Name, enqueueJitter())
	}
	return nil, nil
}

func enqueueJitter() time.Duration {
	baseDelay := 7
	randNum, err := common.GenRandNumber(3)
	if err != nil {
		logrus.Errorf("Failed to generate random number, use `0` as randNumber: %v", err)
	}
	return time.Duration(int(randNum)+baseDelay) * time.Second
}

func generateAnnotationValue(ntpServers string) *nodeconfigv1.AppliedConfigAnnotation {
	return &nodeconfigv1.AppliedConfigAnnotation{
		NTPServers: ntpServers,
	}
}
