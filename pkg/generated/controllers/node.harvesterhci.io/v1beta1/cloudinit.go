/*
Copyright 2024 Rancher Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by main. DO NOT EDIT.

package v1beta1

import (
	"context"
	"time"

	v1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/condition"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CloudInitController interface for managing CloudInit resources.
type CloudInitController interface {
	generic.NonNamespacedControllerInterface[*v1beta1.CloudInit, *v1beta1.CloudInitList]
}

// CloudInitClient interface for managing CloudInit resources in Kubernetes.
type CloudInitClient interface {
	generic.NonNamespacedClientInterface[*v1beta1.CloudInit, *v1beta1.CloudInitList]
}

// CloudInitCache interface for retrieving CloudInit resources in memory.
type CloudInitCache interface {
	generic.NonNamespacedCacheInterface[*v1beta1.CloudInit]
}

type CloudInitStatusHandler func(obj *v1beta1.CloudInit, status v1beta1.CloudInitStatus) (v1beta1.CloudInitStatus, error)

type CloudInitGeneratingHandler func(obj *v1beta1.CloudInit, status v1beta1.CloudInitStatus) ([]runtime.Object, v1beta1.CloudInitStatus, error)

func RegisterCloudInitStatusHandler(ctx context.Context, controller CloudInitController, condition condition.Cond, name string, handler CloudInitStatusHandler) {
	statusHandler := &cloudInitStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, generic.FromObjectHandlerToHandler(statusHandler.sync))
}

func RegisterCloudInitGeneratingHandler(ctx context.Context, controller CloudInitController, apply apply.Apply,
	condition condition.Cond, name string, handler CloudInitGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &cloudInitGeneratingHandler{
		CloudInitGeneratingHandler: handler,
		apply:                      apply,
		name:                       name,
		gvk:                        controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterCloudInitStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type cloudInitStatusHandler struct {
	client    CloudInitClient
	condition condition.Cond
	handler   CloudInitStatusHandler
}

func (a *cloudInitStatusHandler) sync(key string, obj *v1beta1.CloudInit) (*v1beta1.CloudInit, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		if a.condition != "" {
			// Since status has changed, update the lastUpdatedTime
			a.condition.LastUpdated(&newStatus, time.Now().UTC().Format(time.RFC3339))
		}

		var newErr error
		obj.Status = newStatus
		newObj, newErr := a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type cloudInitGeneratingHandler struct {
	CloudInitGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *cloudInitGeneratingHandler) Remove(key string, obj *v1beta1.CloudInit) (*v1beta1.CloudInit, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1beta1.CloudInit{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *cloudInitGeneratingHandler) Handle(obj *v1beta1.CloudInit, status v1beta1.CloudInitStatus) (v1beta1.CloudInitStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.CloudInitGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
