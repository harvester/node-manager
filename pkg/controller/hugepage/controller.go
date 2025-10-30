package hugepage

import (
	"context"
	"fmt"
	"reflect"
	"time"

	ctlnode "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	"github.com/harvester/node-manager/pkg/hugepage"

	nodev1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	ctlhugepage "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
)

const (
	HugepageHandlerName     = "harvester-hugepage-handler"
	HugepageNodeHandlerName = "harvester-hugepage-node-handler"
	MonitorInterval         = 30 * time.Second
)

type Controller struct {
	ctx context.Context

	Name string

	HugepageCache  ctlhugepage.HugepageCache
	HugepageClient ctlhugepage.HugepageController

	NodeCache ctlnode.NodeCache
	Nodes     ctlnode.NodeController

	HugepageManager *hugepage.Manager
}

func Register(ctx context.Context, name string, hugepagectl ctlhugepage.HugepageController, nodes ctlnode.NodeController) (*Controller, error) {
	mgr, err := hugepage.NewHugepageManager(ctx, hugepage.THPPath)
	if err != nil {
		return nil, err
	}
	c := &Controller{
		ctx:             ctx,
		Name:            name,
		HugepageCache:   hugepagectl.Cache(),
		HugepageClient:  hugepagectl,
		NodeCache:       nodes.Cache(),
		Nodes:           nodes,
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

	logrus.WithField("name", key).Debug("reconcilling hugepages object")
	observedStatus, err := c.HugepageManager.GenerateStatus()
	if err != nil {
		return hugetlb, fmt.Errorf("error generating hugepage status: %w", err)
	}

	// if observedConfig is not the same as defined config, we need to bring it in sync
	if !reflect.DeepEqual(hugetlb.Spec.Transparent, observedStatus.Transparent) {
		// apply config and return
		// if there is no error requeue object which will cause observedStatus
		// to be regenerated and applied to object
		logrus.WithField("name", key).Debugf("attempting to apply hugepages configuration")
		if err := c.HugepageManager.ApplyConfig(&hugetlb.Spec.Transparent); err == nil {
			c.HugepageClient.Enqueue(key)
		}
		return hugetlb, err
	}

	if !reflect.DeepEqual(hugetlb.Status, observedStatus) {
		hugetlbCopy := hugetlb.DeepCopy()
		hugetlbCopy.Status = *observedStatus
		if updatedObj, err := c.HugepageClient.UpdateStatus(hugetlbCopy); err != nil {
			return updatedObj, err
		}
	}

	// requeue object to recheck after every MonitorInterval
	c.HugepageClient.EnqueueAfter(key, MonitorInterval)
	return hugetlb, nil
}
