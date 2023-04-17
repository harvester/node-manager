package monitor

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// commonMonitor is the template for others monitoring
type moduleMonitor interface {
	startMonitor()
}

const (
	defaultInterval = 30 * time.Second
)

type Monitor struct {
	Context     context.Context
	MonitorName string
}

func InitServiceMonitor(ctx context.Context, monitorName string) interface{} {
	// Implement service monitor here
	commonMonitor := NewCommonMonitor(ctx, monitorName)
	return commonMonitor
}

func NewCommonMonitor(ctx context.Context, name string) *Monitor {
	return &Monitor{
		Context:     ctx,
		MonitorName: name,
	}
}

func (monitor *Monitor) startMonitor() {
	logrus.Warnf("Do not use Default monitoring, do no-op here.")
	go func() {
		ticker := time.NewTicker(defaultInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := monitor.runMonitor(); err != nil {
					logrus.Errorf("Failed to rescan block devices on node %s: %v", monitor.MonitorName, err)
				}
			case <-monitor.Context.Done():
				return
			}
		}
	}()
}

func (monitor *Monitor) runMonitor() error {
	// no-op here
	logrus.Infof("Default monitor would do no-op.")
	return nil
}

func StartsAllMonitors(monitors []interface{}) {
	for _, rawMonitor := range monitors {
		monitor := rawMonitor.(moduleMonitor)
		monitor.startMonitor()
	}
}
