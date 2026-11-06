package hugepage

import (
	"context"
	"fmt"
	"reflect"
	"time"

	ctlnode "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	"github.com/harvester/node-manager/pkg/hugepage"

	"github.com/harvester/go-common/sys"

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

	// Always persist the hugepage config, to ensure whatever is set is applied on boot
	if err := c.HugepageManager.UpdateHugepageConfigPersistence(&hugetlb.Spec); err != nil {
		return hugetlb, err
	}

	observedStatus, err := c.HugepageManager.GenerateStatus()
	if err != nil {
		return hugetlb, fmt.Errorf("error generating hugepage status: %w", err)
	}

	origNrHugepages := observedStatus.Static.NrHugepages

	// if observedConfig is not the same as defined config, we need to bring it in sync
	if !reflect.DeepEqual(hugetlb.Spec.Transparent, observedStatus.Transparent) ||
		!reflect.DeepEqual(hugetlb.Spec.Static, observedStatus.Static) {

		logrus.WithField("name", key).Debugf("attempting to apply hugepages configuration")
		if err = c.HugepageManager.ApplyConfig(&hugetlb.Spec); err != nil {
			return hugetlb, nil
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
		updatedObj, err := c.HugepageClient.UpdateStatus(hugetlbCopy)
		if err != nil {
			return updatedObj, err
		}
	}

	if observedStatus.Static.NrHugepages != origNrHugepages {
		// The number of allocated hugepages has changed
		logrus.Debugf("origNrHugepages: %d, now %d", origNrHugepages, observedStatus.Static.NrHugepages)
		if observedStatus.Static.NrHugepages == hugetlb.Spec.Static.NrHugepages {
			// And it has changed _to_ the desired allocation, so we can restart the kubelet
			logrus.Infof("Restarting kubelet to set nr_hugepages=%d", hugetlb.Spec.Static.NrHugepages)
			if err = restartKubelet(); err != nil {
				return hugetlb, err
			}
		} else {
			logrus.Debugf("nrHugepages %d not yet at target %d", observedStatus.Static.NrHugepages, hugetlb.Spec.Static.NrHugepages)
		}
	}

	// requeue object to recheck after every MonitorInterval
	c.HugepageClient.EnqueueAfter(key, MonitorInterval)
	return hugetlb, nil
}

func restartKubelet() error {
	// This is safe because TryRestartService will only restart
	// services that are already running, i.e. this will restart
	// whichever of rke2-server or rke2-agent happens to be active
	// on this host
	for _, service := range []string{"rke2-server.service", "rke2-agent.service"} {
		if err := sys.TryRestartService(service); err != nil {
			return err
		}
	}
	return nil
}
