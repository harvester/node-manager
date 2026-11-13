package hugepage

import (
	"context"
	"fmt"
	"reflect"
	"time"

	ctlnode "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/harvester/node-manager/pkg/hugepage"

	"github.com/harvester/go-common/sys"

	"github.com/harvester/go-common/common"
	nodev1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	ctlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
)

const (
	HugepageHandlerName     = "harvester-hugepage-handler"
	HugepageNodeHandlerName = "harvester-hugepage-node-handler"
	MonitorInterval         = 30 * time.Second
)

type Controller struct {
	ctx context.Context

	Name string

	HugepageCache  ctlv1.HugepageCache
	HugepageClient ctlv1.HugepageController

	Nodes ctlnode.NodeController

	NodeConfigCache ctlv1.NodeConfigCache

	HugepageManager *hugepage.Manager
}

func Register(ctx context.Context, name string, hugepagectl ctlv1.HugepageController, nodes ctlnode.NodeController, nodecfg ctlv1.NodeConfigController) (*Controller, error) {
	mgr, err := hugepage.NewHugepageManager(ctx, hugepage.THPPath, hugepage.SHPPath)
	if err != nil {
		return nil, err
	}
	c := &Controller{
		ctx:             ctx,
		Name:            name,
		HugepageCache:   hugepagectl.Cache(),
		HugepageClient:  hugepagectl,
		Nodes:           nodes,
		NodeConfigCache: nodecfg.Cache(),
		HugepageManager: mgr,
	}

	c.HugepageClient.OnChange(ctx, HugepageHandlerName, c.OnChange)

	c.Nodes.OnChange(ctx, HugepageNodeHandlerName, c.NodeOnChange)

	return c, nil
}

func (c *Controller) OnChange(key string, hugetlb *nodev1beta1.Hugepage) (*nodev1beta1.Hugepage, error) {
	if hugetlb == nil || hugetlb.DeletionTimestamp != nil || key != c.Name {
		return hugetlb, nil
	}

	logrus.WithField("name", key).Debug("reconciling hugepages object")

	observedStatus, err := c.HugepageManager.GenerateStatus()
	if err != nil {
		return hugetlb, fmt.Errorf("error generating hugepage status: %w", err)
	}

	if hugetlb.Spec.Static.Hugepages == 0 && observedStatus.Static.Hugepages > 0 {
		// Special case: either the user is deliberately setting hugepages to zero,
		// (there were some allocated before, but now we want them to go away),
		// or we're in a weird upgrade corner case, where the nodeconfig controller
		// allocated hugepages already (before upgrade), but hasn't yet bumped this
		// hugepage object's Hugepages up to whatever Longhorn needs (after upgrade).
		nodeConfig, err := c.NodeConfigCache.Get(common.HarvesterSystemNamespaceName, key)
		if err == nil {
			if nodeConfig.Spec.LonghornConfig != nil &&
				nodeConfig.Spec.LonghornConfig.EnableV2DataEngine &&
				nodeConfig.Spec.LonghornConfig.HugepagesToAllocate > 0 {
				// The LHv2 data engine is enabled, and we want hugepages allocated, but
				// this hugepage object doesn't yet reflect that, because the nodeconfig
				// controller's onchange event hasn't yet run to update it.
				// In this case, we just requeue so we pick it up next time
				logrus.WithField("name", key).Info("waiting to pick up Longhorn V2 hugepage allocation")
				c.HugepageClient.EnqueueAfter(key, MonitorInterval)
				return hugetlb, nil
			}
		} else if !apierrors.IsNotFound(err) {
			return hugetlb, err
		}
	}

	// Always persist the hugepage config, to ensure whatever is set is applied on boot
	if err := c.HugepageManager.UpdateHugepageConfigPersistence(&hugetlb.Spec); err != nil {
		return hugetlb, err
	}

	origHugepages := observedStatus.Static.Hugepages

	// if observedConfig is not the same as defined config, we need to bring it in sync
	if !reflect.DeepEqual(hugetlb.Spec.Transparent, observedStatus.Transparent) ||
		!reflect.DeepEqual(hugetlb.Spec.Static, observedStatus.Static) {

		logrus.WithField("name", key).Debugf("attempting to apply hugepages configuration")
		if err = c.HugepageManager.ApplyConfig(&hugetlb.Spec); err != nil {
			return hugetlb, err
		}
		// When allocating static hugepages, we might not actually get the full
		// allocation yet, so refresh the observed status
		observedStatus, err = c.HugepageManager.GenerateStatus()
		if err != nil {
			return hugetlb, fmt.Errorf("error generating hugepage status: %w", err)
		}
	}

	// Always update the status (this will pick up either the original observed status,
	// or the updated observed status after applying the updated hugepages settings).
	// Note: have to do DeepEqual against *observedStatus (as opposed to observedStatus)
	// or it always returns false due to a type mismatch, which causes unncessary updates.
	if !reflect.DeepEqual(hugetlb.Status, *observedStatus) {
		hugetlbCopy := hugetlb.DeepCopy()
		hugetlbCopy.Status = *observedStatus
		if _, err := c.HugepageClient.UpdateStatus(hugetlbCopy); err != nil {
			// Logging update error rather than returning, so that we fall through
			// to restart the kubelet if hugepage allocation has changed.  Not
			// returning on status update failure here is safe, because we enqueue
			// after the monitor interval at the end of this function anyway, which
			// gives us another retry in 30 seconds time.
			logrus.WithFields(logrus.Fields{
				"name":  key,
				"error": err.Error(),
			}).Warn("failed to update hugepage status")
		}
	}

	if observedStatus.Static.Hugepages != origHugepages {
		// The number of allocated hugepages has changed
		logrus.Debugf("origHugepages: %d, now %d", origHugepages, observedStatus.Static.Hugepages)
		if observedStatus.Static.Hugepages == hugetlb.Spec.Static.Hugepages {
			// And it has changed _to_ the desired allocation, so we can restart the kubelet
			logrus.Infof("Restarting kubelet to set hugepages=%d", hugetlb.Spec.Static.Hugepages)
			restartKubelet()
		} else {
			logrus.Debugf("hugepages %d not yet at target %d", observedStatus.Static.Hugepages, hugetlb.Spec.Static.Hugepages)
		}
	}

	// requeue object to recheck after every MonitorInterval
	c.HugepageClient.EnqueueAfter(key, MonitorInterval)
	return hugetlb, nil
}

func restartKubelet() {
	go func() {
		// Kubelet restart needs to be slightly delayed, because if we do it immediately
		// inside the OnChange event, then when editing hugepage settings in the GUI,
		// we see the following error:
		//   Internal error occurred: failed calling webhook "validator.harvesterhci.io":
		//   failed to call webhook: Post "https://harvester-webhook.harvester-system.svc:443/v1/webhook/validation?timeout=10s":
		//   dial tcp 127.0.0.1:9345: connect: connection refused
		// Injecting a tiny lag here before the restart means we don't break that validation.
		time.Sleep(2 * time.Second)

		// This is safe because TryRestartService will only restart
		// services that are already running, i.e. this will restart
		// whichever of rke2-server or rke2-agent happens to be active
		// on this host
		for _, service := range []string{"rke2-server.service", "rke2-agent.service"} {
			if err := sys.TryRestartService(service); err != nil {
				logrus.WithFields(logrus.Fields{
					"name":  service,
					"error": err.Error(),
				}).Error("failed to restart service")
			}
		}
	}()
}
