package hugepage

import (
	"context"
	"reflect"

	"github.com/sirupsen/logrus"

	"github.com/harvester/node-manager/pkg/hugepage"

	nodev1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	ctlhugepage "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
)

const (
	HugepageHandlerName = "harvester-hugepage-handler"
)

type Controller struct {
	ctx context.Context

	Name string

	HugepageCache  ctlhugepage.HugepageCache
	HugepageClient ctlhugepage.HugepageController

	HugepageManager *hugepage.HugepageManager
}

func Register(ctx context.Context, name string, hugepagectl ctlhugepage.HugepageController) (*Controller, error) {
	man := hugepage.NewHugepageManager(ctx)

	c := &Controller{
		ctx:             ctx,
		Name:            name,
		HugepageCache:   hugepagectl.Cache(),
		HugepageClient:  hugepagectl,
		HugepageManager: man,
	}

	c.HugepageClient.OnChange(ctx, HugepageHandlerName, c.OnChange)

	go c.Watch(ctx, name)

	return c, nil
}

func (c *Controller) OnChange(key string, hugetlb *nodev1beta1.Hugepage) (*nodev1beta1.Hugepage, error) {
	if hugetlb == nil || hugetlb.DeletionTimestamp != nil || key != c.Name {
		return hugetlb, nil
	}

	ch := c.HugepageManager.GetSpecChan()
	ch <- &hugetlb.Spec
	return hugetlb, nil
}

func (c *Controller) Watch(ctx context.Context, name string) {
	ch := c.HugepageManager.GetStatusChan()
	for {
		select {
		case s := <-ch:
			oldObj, err := c.HugepageCache.Get(name)
			if err != nil {
				logrus.Errorf("failed to get hugepage %v: %v", name, err)
				continue
			}

			if !reflect.DeepEqual(oldObj.Status, s) {
				newObj := oldObj.DeepCopy()
				newObj.Status = *s

				if _, err := c.HugepageClient.UpdateStatus(newObj); err != nil {
					logrus.Errorf("failed to update hugepage status %v: %v", name, err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
