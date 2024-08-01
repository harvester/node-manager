package cloudinit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	ctlnodev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cloudinitv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/cloudinit"
	ctrlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
)

const (
	handlerName = "harvester-node-cloud-init-controller"

	// CloudInit objects are Cluster-scoped, but Kubernetes really wants
	// a Namespace associated with the Event objects. It rejects an Event
	// object with a Namespace of `harvester-system` since the CloudInit
	// object is Cluster-scoped and hence not in `harvester-system` or any
	// Namespace. Kubernetes is fine with `default` though.
	eventNamespace = "default"

	eventActionReconcile = "ReconcileContents"
	eventReasonReconcile = "CloudInitFileModified"
	eventActionRemove    = "RemoveFile"
	eventReasonRemove    = "CloudInitNotApplicable"

	// This is mainly used for detecting a "zero value" for timestamps
	// in a call to stat, where 0 means it has been 0 seconds since the
	// unix epoch.
	epoch int64 = 0
)

type controller struct {
	nodeName   string
	cloudinits ctrlv1.CloudInitClient
	events     ctlnodev1.EventClient
	nodeCache  ctlnodev1.NodeCache
}

func Register(ctx context.Context, nodeName string, cloudinits ctrlv1.CloudInitController, nodeCache ctlnodev1.NodeCache, events ctlnodev1.EventClient) {
	ctl := &controller{
		nodeName:   nodeName,
		cloudinits: cloudinits,
		events:     events,
		nodeCache:  nodeCache,
	}

	cloudinits.OnChange(ctx, handlerName, ctl.OnCloudInitChange)
	cloudinits.OnRemove(ctx, handlerName, ctl.OnCloudInitRemove)
}

func (c *controller) OnCloudInitChange(_ string, cloudInitObj *cloudinitv1.CloudInit) (*cloudinitv1.CloudInit, error) {
	if cloudInitObj == nil || cloudInitObj.DeletionTimestamp != nil {
		return cloudInitObj, nil
	}

	cloudInitCopy := cloudInitObj.DeepCopy()

	if cloudInitCopy.Annotations == nil {
		cloudInitCopy.Annotations = make(map[string]string)
	}

	checksum, err := cloudinit.Measure(strings.NewReader(cloudInitCopy.Spec.Contents))
	if err != nil {
		return cloudInitCopy, err
	}

	checksumString := fmt.Sprintf("%x", checksum)
	persistedChecksum := strings.ToLower(cloudInitCopy.Annotations[cloudinit.AnnotationHash])

	if checksumString != persistedChecksum {
		cloudInitCopy.Annotations[cloudinit.AnnotationHash] = checksumString
		return c.cloudinits.Update(cloudInitCopy)
	}

	node, err := c.nodeCache.Get(c.nodeName)
	if err != nil {
		return cloudInitCopy, err
	}

	if cloudInitCopy.Spec.Paused {
		return c.updateStatus(node, cloudInitCopy)
	}

	if cloudinit.MatchesNode(node, cloudInitCopy) {
		changed, err := cloudinit.RequireLocal(cloudInitCopy)
		_, _ = c.updateStatus(node, cloudInitCopy)
		if err != nil {
			return cloudInitCopy, err
		}

		if !changed {
			return cloudInitCopy, nil
		}

		err = c.emitOverwriteEvent(cloudInitCopy)
		if err != nil {
			logrus.WithError(err).
				WithField("cloudinit_name", cloudInitCopy.Name).
				Warn("Failed to emit overwrite event for CloudInit")
		}

		return cloudInitCopy, err
	}

	err = os.Remove(filepath.Join(cloudinit.Directory, cloudInitCopy.Spec.Filename))
	if os.IsNotExist(err) {
		return c.updateStatus(node, cloudInitCopy)
	}

	err = c.emitRemoveEvent(cloudInitCopy)
	if err != nil {
		logrus.WithError(err).
			WithField("cloudinit_name", cloudInitCopy.Name).
			Warn("Failed to emit removal event for CloudInit")
	}

	return c.updateStatus(node, cloudInitCopy)
}

func (c *controller) OnCloudInitRemove(_ string, cloudInitObj *cloudinitv1.CloudInit) (*cloudinitv1.CloudInit, error) {
	err := os.Remove(filepath.Join(cloudinit.Directory, cloudInitObj.Spec.Filename))
	if os.IsNotExist(err) {
		return cloudInitObj, nil
	}

	if err != nil {
		return cloudInitObj, err
	}

	return cloudInitObj, nil
}

type byCondType []metav1.Condition

func (n byCondType) Len() int           { return len(n) }
func (n byCondType) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byCondType) Less(i, j int) bool { return n[i].Type < n[j].Type }

const (
	CloudInitReasonPresent          = "CloudInitPresentOnDisk"
	CloudInitReasonAbsent           = "CloudInitAbsentFromDisk"
	CloudInitReasonError            = "CloudInitError"
	CloudInitReasonApplicable       = "CloudInitApplicable"
	CloudInitReasonNotApplicable    = "CloudInitNotApplicable"
	CloudInitReasonChecksumMismatch = "CloudInitChecksumMismatch"
	CloudInitReasonChecksumMatch    = "CloudInitChecksumMatch"
)

func (c *controller) updateStatus(node *corev1.Node, cloudInitObj *cloudinitv1.CloudInit) (*cloudinitv1.CloudInit, error) {
	var st unix.Stat_t
	_ = unix.Stat(filepath.Join(cloudinit.Directory, cloudInitObj.Spec.Filename), &st)
	createdAt := time.Unix(st.Ctim.Sec, st.Ctim.Nsec)
	modifiedAt := time.Unix(st.Mtim.Sec, st.Mtim.Nsec)

	conds := []metav1.Condition{
		newApplicableCondition(node, cloudInitObj),
		newOutOfSyncCondition(cloudInitObj),
		newPresentCondition(cloudInitObj),
	}
	sort.Sort(byCondType(conds))

	rollout := cloudInitObj.Status.Rollouts[c.nodeName]

	oldConds := make(map[string]metav1.Condition)
	for _, cond := range rollout.Conditions {
		oldConds[cond.Type] = cond
	}

	now := time.Now()

	if createdAt.Unix() == epoch {
		createdAt = now
	}
	if modifiedAt.Unix() == epoch {
		modifiedAt = now
	}

	for i := range conds {
		cond := &conds[i]
		prev := oldConds[cond.Type]

		// Only retain the LastTransitionTime when it is not the first time
		// processing this metav1.Cond.
		var zero metav1.Condition
		if prev != zero &&
			cond.Status == prev.Status &&
			cond.Reason == prev.Reason &&
			cond.Message == prev.Message {
			cond.LastTransitionTime = prev.LastTransitionTime
			continue
		}

		switch cond.Type {
		case string(cloudinitv1.CloudInitFilePresent):
			if createdAt.After(cond.LastTransitionTime.Time) {
				cond.LastTransitionTime = metav1.NewTime(createdAt)
			}
		case string(cloudinitv1.CloudInitOutOfSync):
			if modifiedAt.After(cond.LastTransitionTime.Time) {
				cond.LastTransitionTime = metav1.NewTime(modifiedAt)
			}
		case string(cloudinitv1.CloudInitApplicable):
			cond.LastTransitionTime = metav1.NewTime(now)
		}
	}

	updated := cloudinitv1.Rollout{Conditions: conds}
	if reflect.DeepEqual(updated, cloudInitObj.Status.Rollouts[c.nodeName]) {
		return cloudInitObj, nil
	}

	if cloudInitObj.Status.Rollouts == nil {
		cloudInitObj.Status.Rollouts = make(map[string]cloudinitv1.Rollout)
	}

	cloudInitObj.Status.Rollouts[c.nodeName] = updated
	return c.cloudinits.UpdateStatus(cloudInitObj)
}

func newPresentCondition(cloudInit *cloudinitv1.CloudInit) metav1.Condition {
	var status metav1.ConditionStatus
	var reason, message string

	f, err := os.Open(filepath.Join(cloudinit.Directory, cloudInit.Spec.Filename))
	switch {
	case err == nil:
		defer f.Close()
		status = metav1.ConditionTrue
		reason = CloudInitReasonPresent
		message = fmt.Sprintf("%s is present under /oem", cloudInit.Spec.Filename)
	case os.IsNotExist(err):
		status = metav1.ConditionFalse
		reason = CloudInitReasonAbsent
		message = fmt.Sprintf("%s is absent from /oem", cloudInit.Spec.Filename)
	default:
		status = metav1.ConditionUnknown
		reason = CloudInitReasonError
		message = fmt.Sprintf("Open file: %v", err)
	}

	return metav1.Condition{
		Type:    string(cloudinitv1.CloudInitFilePresent),
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

func (c *controller) emitOverwriteEvent(cloudInitObj *cloudinitv1.CloudInit) error {
	now := time.Now()

	eventName := fmt.Sprintf("cloudinit-overwrite-%s.%s", cloudInitObj.Name, c.nodeName)

	event, err := c.events.Get(eventNamespace, eventName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		event = &corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      eventName,
				Namespace: eventNamespace,
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: cloudInitObj.Kind,
				Name: cloudInitObj.Name,
				UID:  cloudInitObj.UID,
			},
			Action:  eventActionReconcile,
			Reason:  eventReasonReconcile,
			Message: fmt.Sprintf("%s has been overwritten on %s", cloudInitObj.Spec.Filename, c.nodeName),
			Source: corev1.EventSource{
				Component: "harvester-node-manager",
				Host:      c.nodeName,
			},
			Type:                "Normal",
			ReportingController: fmt.Sprintf("harvesterhci.io/%s", handlerName),
			ReportingInstance:   c.nodeName,
			EventTime:           metav1.NewMicroTime(time.Now()),
			FirstTimestamp:      metav1.NewTime(now),
			LastTimestamp:       metav1.NewTime(now),
			Count:               1,
		}

		if _, err := c.events.Create(event); err != nil {
			return err
		}

		return nil
	}
	if err != nil {
		return err
	}

	event.LastTimestamp = metav1.NewTime(now)
	event.Count++

	if _, err := c.events.Update(event); err != nil {
		return err
	}

	return nil
}

func (c *controller) emitRemoveEvent(cloudInitObj *cloudinitv1.CloudInit) error {
	now := time.Now()

	eventName := fmt.Sprintf("cloudinit-remove-%s.%s", cloudInitObj.Name, c.nodeName)

	event, err := c.events.Get(eventNamespace, eventName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		event = &corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      eventName,
				Namespace: eventNamespace,
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: cloudInitObj.Kind,
				Name: cloudInitObj.Name,
				UID:  cloudInitObj.UID,
			},
			Action:  eventActionRemove,
			Reason:  eventReasonRemove,
			Message: fmt.Sprintf("%s has been removed from %s", cloudInitObj.Spec.Filename, c.nodeName),
			Source: corev1.EventSource{
				Component: "harvester-node-manager",
				Host:      c.nodeName,
			},
			Type:                "Normal",
			ReportingController: fmt.Sprintf("harvesterhci.io/%s", handlerName),
			ReportingInstance:   c.nodeName,
			EventTime:           metav1.NewMicroTime(time.Now()),
			FirstTimestamp:      metav1.NewTime(now),
			LastTimestamp:       metav1.NewTime(now),
			Count:               1,
		}

		if _, err := c.events.Create(event); err != nil {
			return err
		}

		return nil
	}
	if err != nil {
		return err
	}

	event.LastTimestamp = metav1.NewTime(now)
	event.Count++

	if _, err := c.events.Update(event); err != nil {
		return err
	}

	return nil
}

func newApplicableCondition(node *corev1.Node, cloudInit *cloudinitv1.CloudInit) metav1.Condition {
	var (
		status  = metav1.ConditionFalse
		reason  = CloudInitReasonNotApplicable
		message = "MatchSelector does not match Node labels"
	)

	if cloudinit.MatchesNode(node, cloudInit) {
		status = metav1.ConditionTrue
		reason = CloudInitReasonApplicable
		message = ""
	}

	return metav1.Condition{
		Type:    string(cloudinitv1.CloudInitApplicable),
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

func newOutOfSyncCondition(cloudInit *cloudinitv1.CloudInit) metav1.Condition {
	var (
		status  = metav1.ConditionTrue
		reason  = CloudInitReasonChecksumMismatch
		message = "Local file checksum is different than the CloudInit checksum"
	)

	f, err := os.Open(filepath.Join(cloudinit.Directory, cloudInit.Spec.Filename))
	if err != nil {
		return metav1.Condition{
			Type:    string(cloudinitv1.CloudInitOutOfSync),
			Status:  metav1.ConditionUnknown,
			Reason:  CloudInitReasonError,
			Message: fmt.Sprintf("Open file: %v", err),
		}
	}
	defer f.Close()

	checksum, err := cloudinit.Measure(f)
	if err != nil {
		status = metav1.ConditionUnknown
		reason = CloudInitReasonError
		message = fmt.Sprintf("Calculate checksum: %v", err)
	} else if fmt.Sprintf("%x", checksum) == cloudInit.Annotations[cloudinit.AnnotationHash] {
		status = metav1.ConditionFalse
		reason = CloudInitReasonChecksumMatch
		message = "Local file checksum is the same as the CloudInit checksum"
	}

	return metav1.Condition{
		Type:    string(cloudinitv1.CloudInitOutOfSync),
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}
