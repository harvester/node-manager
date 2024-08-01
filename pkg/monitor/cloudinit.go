package monitor

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	gocommon "github.com/harvester/go-common"
	ctlnodev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	cloudinitv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/cloudinit"
	ctrlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
)

type CloudInitMonitor struct {
	ctx             context.Context
	monitorName     string
	nodeName        string
	cloudinits      ctrlv1.CloudInitController
	cloudinitsCache ctrlv1.CloudInitCache
	nodeCache       ctlnodev1.NodeCache
}

func NewCloudInitMonitor(ctx context.Context, monitorName, nodeName string, cloudinits ctrlv1.CloudInitController, nodeCache ctlnodev1.NodeCache) *CloudInitMonitor {
	return &CloudInitMonitor{
		ctx:             ctx,
		monitorName:     monitorName,
		nodeName:        nodeName,
		cloudinits:      cloudinits,
		cloudinitsCache: cloudinits.Cache(),
		nodeCache:       nodeCache,
	}
}

func (m *CloudInitMonitor) startMonitor() {
	go func() {
		gocommon.WatchFileChange(m.ctx, gocommon.FSNotifyHandlerFunc(m.handleFSNotify), []string{cloudinit.Directory})
	}()
}

func (m *CloudInitMonitor) handleFSNotify(event fsnotify.Event) {
	logrus.Debugf("Handling fsnotify Event %+v", event)

	path := event.Name

	retryAfter := func(event fsnotify.Event, wait time.Duration, cause error) {
		logrus.WithFields(logrus.Fields{
			"path":  path,
			"cause": cause,
		}).Info("Rescheduled file for possible CloudInit reconciliation")

		select {
		case <-m.ctx.Done():
			return
		case <-time.After(wait):
			m.handleFSNotify(event)
		}
	}

	cloudinits, err := m.cloudinitsCache.List(labels.Everything())
	if err != nil {
		logrus.Warnf("Fetch CloudInits failed: %v", err)
		go retryAfter(event, 5*time.Second, fmt.Errorf("list cloudinits: %w", err))
		return
	}

	node, err := m.nodeCache.Get(m.nodeName)
	if err != nil {
		logrus.Warnf("Fetch node %q from cache failed: %v", m.nodeName, err)
		go retryAfter(event, 5*time.Second, fmt.Errorf("get node %q: %w", m.nodeName, err))
		return
	}

	matching := make([]*cloudinitv1.CloudInit, 0, len(cloudinits))
	for _, c := range cloudinits {
		if !cloudinit.MatchesNode(node, c) {
			continue
		}
		matching = append(matching, c)
	}

	logrus.Debugf("Fetched %d CloudInits for %s", len(matching), m.nodeName)

	managedFiles := make(map[string]*cloudinitv1.CloudInit)

	for _, cloudinit := range matching {
		managedFiles[cloudinit.Spec.Filename] = cloudinit
	}

	filename := filepath.Base(path)

	match, ok := managedFiles[filename]
	if !ok {
		logrus.Debugf("%q is not a managed file", filename)
		return
	}

	logrus.Infof("Managed file %s has changed on disk, enqueueing for overwrite", event.Name)
	m.cloudinits.Enqueue(match.Name)
}
