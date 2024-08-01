package ksmtuned

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rancher/wrangler/v3/pkg/ticker"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/sirupsen/logrus"

	ksmtunedv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

const (
	MonitorInterval = time.Second * 60
	MinThres        = 5
)

var (
	ksmReg = regexp.MustCompile(`\d+`)

	mergeAcrossNodesLocker sync.Mutex

	// single instance
	once     sync.Once
	instance *Ksmtuned
)

type Ksmtuned struct {
	ctx      context.Context
	nodeName string
	ksmd     *ksmd

	// statusCh monitoring ksm status and ksmtuned applied parameters.
	statusCh chan *ksmtunedv1.KsmtunedStatus
	running  bool

	// ksmtuend parameters
	sleepMsec uint64
	boost     uint
	decay     uint
	minPages  uint
	maxPages  uint
	thresCoef uint64

	// memTotal get host memory total size.
	memTotal uint64
	// curPage write to pages_to_scan file.
	curPage uint

	ksmdPhase ksmtunedv1.KsmdPhase

	// ksmdUtilization expose ksmd the cpu utilization metrics.
	ksmdUtilization *prometheus.GaugeVec
}

func NewKsmtuned(ctx context.Context, nodeName string) (*Ksmtuned, error) {
	var err error

	once.Do(func() {
		var (
			memTotal uint64
			ksmd     *ksmd
		)
		memTotal, err = totalMemory()
		if err != nil {
			err = fmt.Errorf("failed to get memory info: %s", err)
			return
		}

		ksmd, err = newKsmd()
		if err != nil {
			err = fmt.Errorf("get ksmd process: %s", err)
			return
		}

		instance = &Ksmtuned{
			ctx:      ctx,
			nodeName: nodeName,
			ksmd:     ksmd,
			statusCh: make(chan *ksmtunedv1.KsmtunedStatus, 10),
			memTotal: memTotal,
		}
		go instance.run()
	})
	return instance, err
}

func (k *Ksmtuned) SetKsmdUtilization(gv *prometheus.GaugeVec) {
	k.ksmdUtilization = gv
}

func (k *Ksmtuned) Apply(thresCoef uint, param ksmtunedv1.KsmtunedParameters) {
	k.apply(thresCoef, param)
	k.running = true
}

func (k *Ksmtuned) apply(thresCoef uint, param ksmtunedv1.KsmtunedParameters) {
	if thresCoef > 100 {
		thresCoef = 100
	}

	k.sleepMsec = uint64(param.SleepMsec) * 16 * 1024 * 1024 / k.memTotal
	if k.sleepMsec < 10 {
		k.sleepMsec = 10
	}

	k.thresCoef = uint64(thresCoef) * k.memTotal / 100
	if k.thresCoef < MinThres {
		k.thresCoef = MinThres
	}

	k.boost = param.Boost
	k.decay = param.Decay

	if k.curPage < 1 || k.minPages != param.MinPages {
		k.curPage = k.minPages
	}
	k.minPages = param.MinPages
	k.maxPages = param.MaxPages
}

func (k *Ksmtuned) Stop() error {
	k.running = false
	return k.ksmd.stop(0)
}

func (k *Ksmtuned) Prune() error {
	k.running = false
	return k.ksmd.prune(0)
}

func (k *Ksmtuned) run() {
	t := ticker.Context(k.ctx, MonitorInterval)
	for {
		select {
		case <-t:
			if k.running {
				if err := k.adjust(); err != nil {
					logrus.Errorf("failed to adjust: %s", err)
				}
			}
		case <-k.ctx.Done():
			if err := k.Stop(); err != nil {
				logrus.Errorf("failed to stop: %s", err)
			}
			time.Sleep(time.Second * 10)
			return
		}

		if err := k.status(); err != nil {
			logrus.Error("failed to get status:", err)
		}
		if err := k.metrics(); err != nil {
			logrus.Error("failed to set metrics:", err)
		}
	}
}

// adjust calculate and handling the state before and after the free memory thresCoef,
// start ksm when greater than thresCoef, otherwise stop ksm.
func (k *Ksmtuned) adjust() error {
	free, err := freeMemory()
	if err != nil {
		return err
	}

	if free < k.thresCoef {
		k.increase()
		if err := k.ksmd.start(k.curPage, k.sleepMsec); err != nil {
			return err
		}
		k.ksmdPhase = ksmtunedv1.KsmdRunning
	} else {
		k.decrease()
		if err := k.ksmd.stop(k.curPage); err != nil {
			return err
		}
		k.ksmdPhase = ksmtunedv1.KsmdStopped
	}
	return nil
}

// increase processing efficiency
func (k *Ksmtuned) increase() {
	k.curPage += k.boost
	if k.curPage > k.maxPages {
		k.curPage = k.maxPages
	}
}

// decrease only when stopped
func (k *Ksmtuned) decrease() {
	if k.decay > k.curPage {
		k.curPage = k.minPages
	} else {
		k.curPage -= k.decay
		if k.curPage < k.minPages {
			k.curPage = k.minPages
		}
	}
}

func (k *Ksmtuned) Status() <-chan *ksmtunedv1.KsmtunedStatus {
	return k.statusCh
}

func (k *Ksmtuned) status() error {
	ks, err := k.ksmd.readKsmdStatus()
	if err != nil {
		return err
	}

	k.statusCh <- &ksmtunedv1.KsmtunedStatus{
		Shared:           ks.shared,
		Sharing:          ks.sharing,
		Unshared:         ks.unshared,
		Volatile:         ks.volatile,
		FullScans:        ks.fullScans,
		StableNodeDups:   ks.stableNodeDups,
		StableNodeChains: ks.stableNodeChains,
		KsmdPhase:        k.ksmdPhase,
	}
	return nil
}

func (k *Ksmtuned) metrics() error {
	if k.ksmdUtilization == nil {
		return nil
	}

	percent, err := k.ksmd.metrics()
	if err != nil {
		return err
	}

	k.ksmdUtilization.WithLabelValues(k.nodeName).Set(percent)
	return nil
}

func (k *Ksmtuned) RunStatus() (ksmtunedv1.KsmdPhase, error) {
	r, err := k.ksmd.getRunStatus()
	if err != nil {
		return ksmtunedv1.KsmdUndefined, err
	}

	var phase ksmtunedv1.KsmdPhase
	switch ksmdRun(r) {
	case ksmdStop:
		phase = ksmtunedv1.KsmdStopped
	case ksmdRunning:
		phase = ksmtunedv1.KsmdRunning
	case ksmdPrune:
		phase = ksmtunedv1.KsmdPruned
	}
	return phase, nil
}

// CompareMergeAcrossNodes compare Ksmtuned configure and /sys/kernel/mm/ksm/merge_across_nodes
func (k Ksmtuned) compareMergeAcrossNodes(toggle uint) (unchanged bool, err error) {
	s, err := k.ksmd.getMergeAcrossNodes()
	if err != nil {
		return false, err
	}

	if s == uint64(toggle) {
		return true, nil
	}
	return false, nil
}

// ToggleMergeAcrossNodes toggle /sys/kernel/mm/ksm/merge_across_nodes
func (k Ksmtuned) ToggleMergeAcrossNodes(toggle uint) error {
	mergeAcrossNodesLocker.Lock()
	defer mergeAcrossNodesLocker.Unlock()
	if unchanged, err := k.compareMergeAcrossNodes(toggle); err != nil {
		return err
	} else if unchanged {
		return nil
	}
	return k.ksmd.toggleMergeAcrossNodes(k.ctx, toggle)
}

func totalMemory() (uint64, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return memInfo.Total, nil
}

func freeMemory() (uint64, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return memInfo.Available, nil
}
