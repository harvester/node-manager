package ksmtuned

import (
	"context"
	"reflect"

	ctlnode "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"

	ksmtunedv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	ctlksmtuned "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/ksmtuned"
	"github.com/harvester/node-manager/pkg/metrics"
)

const (
	HandlerName     = "harvester-ksmtuned-handler"
	NodeHandlerName = "harvester-ksmtuned-node-handler"
)

var (
	modes = map[ksmtunedv1.KsmtunedMode]ksmtunedv1.KsmtunedParameters{
		ksmtunedv1.StandardMode: {
			SleepMsec: 20,
			Boost:     0,
			Decay:     0,
			MinPages:  100,
			MaxPages:  100,
		},
		ksmtunedv1.HighMode: {
			SleepMsec: 20,
			Boost:     200,
			Decay:     50,
			MinPages:  100,
			MaxPages:  10000,
		},
	}
)

type Controller struct {
	ctx      context.Context
	NodeName string

	KsmtunedCache ctlksmtuned.KsmtunedCache
	Ksmtuneds     ctlksmtuned.KsmtunedController

	NodeCache ctlnode.NodeCache
	Nodes     ctlnode.NodeController

	Ksmtuned *ksmtuned.Ksmtuned
}

func Register(ctx context.Context, nodeName string, kts ctlksmtuned.KsmtunedController, nodes ctlnode.NodeController) (*Controller, error) {
	k, err := ksmtuned.NewKsmtuned(ctx, nodeName)
	if err != nil {
		return nil, err
	}

	k.SetKsmdUtilization(metrics.KsmdUtilizationGV)
	c := &Controller{
		ctx:           ctx,
		NodeName:      nodeName,
		KsmtunedCache: kts.Cache(),
		Ksmtuneds:     kts,
		NodeCache:     nodes.Cache(),
		Nodes:         nodes,
		Ksmtuned:      k,
	}

	c.Ksmtuneds.OnChange(ctx, HandlerName, c.OnChange)
	c.Ksmtuneds.OnRemove(ctx, HandlerName, c.OnRemove)

	c.Nodes.OnChange(ctx, NodeHandlerName, c.NodeOnChange)

	go c.watchStatus(ctx, nodeName)

	return c, nil
}

func (c *Controller) OnChange(key string, kt *ksmtunedv1.Ksmtuned) (*ksmtunedv1.Ksmtuned, error) {
	if kt == nil || kt.DeletionTimestamp != nil || key != c.NodeName {
		return kt, nil
	}

	// process merge across nodes changed
	if err := c.Ksmtuned.ToggleMergeAcrossNodes(kt.Spec.MergeAcrossNodes); err != nil {
		return kt, err
	}

	var (
		parameters ksmtunedv1.KsmtunedParameters
		ok         bool
	)

	switch kt.Spec.Run {
	case ksmtunedv1.Stop:
		return kt, c.Ksmtuned.Stop()
	case ksmtunedv1.Prune:
		return kt, c.Ksmtuned.Prune()
	default:
		if parameters, ok = modes[kt.Spec.Mode]; !ok {
			c.Ksmtuned.Apply(kt.Spec.ThresCoef, kt.Spec.KsmtunedParameters)
			return kt, nil
		}
	}

	c.Ksmtuned.Apply(kt.Spec.ThresCoef, parameters)

	if !reflect.DeepEqual(kt.Spec.KsmtunedParameters, parameters) {
		newObj := kt.DeepCopy()

		newObj.Spec.KsmtunedParameters = parameters
		return c.Ksmtuneds.Update(newObj)
	}
	return kt, nil
}

func (c *Controller) OnRemove(_ string, kt *ksmtunedv1.Ksmtuned) (*ksmtunedv1.Ksmtuned, error) {
	if kt.Name != c.NodeName {
		return kt, nil
	}
	return kt, c.Ksmtuned.Stop()
}

func (c *Controller) watchStatus(ctx context.Context, name string) {
	ch := c.Ksmtuned.Status()
	for {
		select {
		case s := <-ch:
			oldObj, err := c.KsmtunedCache.Get(name)
			if err != nil {
				logrus.Errorf("failed to get Ksmtuned %s: %s", name, err)
				continue
			} else if reflect.DeepEqual(oldObj.Status, s) {
				continue
			}

			newObj := oldObj.DeepCopy()
			newObj.Status = *s

			phase, err := c.Ksmtuned.RunStatus()
			if err != nil {
				logrus.Errorf("failed to get ksmd run status: %s", err)
				continue
			}
			newObj.Status.KsmdPhase = phase

			if !reflect.DeepEqual(newObj.Status, oldObj.Status) {
				if _, err := c.Ksmtuneds.UpdateStatus(newObj); err != nil {
					logrus.Errorf("failed to update Ksmtuned %s: %s", name, err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
