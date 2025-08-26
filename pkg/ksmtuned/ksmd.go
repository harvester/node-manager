package ksmtuned

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

// relate to /sys/kernel/mm/ksm/run
type ksmdRun uint

const (
	ksmdStop    ksmdRun = 0
	ksmdRunning ksmdRun = 1
	ksmdPrune   ksmdRun = 2
)

type ksmKey string

const (
	shared           ksmKey = "shared"
	sharing          ksmKey = "sharing"
	unshared         ksmKey = "unshared"
	volatile         ksmKey = "volatile"
	fullScan         ksmKey = "fullScan"
	stableNodeChains ksmKey = "stableNodeChains"
	stableNodeDups   ksmKey = "stableNodeDups"
)

type ksmPath string

const (
	Ksmd = "ksmd"

	KSMPath            ksmPath = "/sys/kernel/mm/ksm"
	RunPath                    = KSMPath + "/run"
	PagesToScanPath            = KSMPath + "/pages_to_scan"
	SleepMillisecsPath         = KSMPath + "/sleep_millisecs"
	MergeAcrossNodes           = KSMPath + "/merge_across_nodes"
)

var (
	// The effectiveness of KSM and MADV_MERGEABLE is shown in /sys/kernel/mm/ksm/:
	ksmdStatusMap = map[ksmKey]ksmPath{
		shared:           KSMPath + "/pages_shared",
		sharing:          KSMPath + "/pages_sharing",
		unshared:         KSMPath + "/pages_unshared",
		volatile:         KSMPath + "/pages_volatile",
		fullScan:         KSMPath + "/full_scans",
		stableNodeChains: KSMPath + "/stable_node_chains",
		stableNodeDups:   KSMPath + "/stable_node_dups",
	}
)

type (
	ksmd struct {
		proc *process.Process
	}

	ksmdStatus struct {
		sharing          uint64
		shared           uint64
		unshared         uint64
		volatile         uint64
		fullScans        uint64
		stableNodeChains uint64
		stableNodeDups   uint64
	}
)

func newKsmd() (*ksmd, error) {
	p, err := getKsmd()
	if err != nil {
		return nil, fmt.Errorf("failed to get ksmd process: %s", err)
	}

	k := &ksmd{
		proc: p,
	}

	return k, nil
}

func (k *ksmd) toggleMergeAcrossNodes(ctx context.Context, toggle uint) (err error) {

	if err = k.prune(0); err != nil {
		return err
	}

	ctxCancel, cancel := context.WithCancel(ctx)
	wait.UntilWithContext(ctxCancel, func(_ context.Context) {
		var s *ksmdStatus
		s, err = k.readKsmdStatus()
		if err != nil || s.shared+s.sharing+s.unshared < 1 {
			cancel()
		}

	}, time.Second)

	if err != nil {
		return err
	}

	s := strconv.FormatUint(uint64(toggle), 10)

	return saveKsmPath(MergeAcrossNodes, []byte(s))
}

func (k *ksmd) start(pagesToScan uint, sleepMsec uint64) error {
	if err := k.save(ksmdRunning, pagesToScan); err != nil {
		return fmt.Errorf("failed to start ksmd: %s", err)
	}
	if err := saveKsmPathByUint64(SleepMillisecsPath, sleepMsec); err != nil {
		return fmt.Errorf("failed to set sleep_millisecs: %s", err)
	}
	return nil
}

func (k *ksmd) stop(pagesToScan uint) error {
	if err := k.save(ksmdStop, pagesToScan); err != nil {
		return fmt.Errorf("failed to stop ksmd: %s", err)
	}
	return nil
}

func (k *ksmd) prune(pagesToScan uint) error {
	if err := k.save(ksmdPrune, pagesToScan); err != nil {
		return fmt.Errorf("failed to prune ksmd: %s", err)
	}
	return nil
}

func (k *ksmd) save(r ksmdRun, pagesToScan uint) error {
	logrus.Debugf("run: %d, pages_to_scan: %d", r, pagesToScan)
	if err := saveRun(r); err != nil {
		return fmt.Errorf("failed to operate ksmd: %s", err)
	}
	if err := saveKsmPathByUint(PagesToScanPath, pagesToScan); err != nil {
		return fmt.Errorf("failed to write pages_to_scan : %s", err)
	}
	return nil
}

func (k *ksmd) readKsmdStatus() (*ksmdStatus, error) {
	ks := &ksmdStatus{}
	for key, path := range ksmdStatusMap {
		v, err := readKsmPath(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read ksm files: %s: %s", path, err)
		}
		switch key {
		case shared:
			ks.shared = v
		case sharing:
			ks.sharing = v
		case unshared:
			ks.unshared = v
		case volatile:
			ks.volatile = v
		case fullScan:
			ks.fullScans = v
		case stableNodeChains:
			ks.stableNodeChains = v
		case stableNodeDups:
			ks.stableNodeDups = v
		}
	}
	return ks, nil
}

func (k *ksmd) getRunStatus() (uint64, error) {
	return readKsmPath(RunPath)
}

func (k *ksmd) metrics() (float64, error) {
	percent, err := k.proc.Percent(time.Second)
	if err != nil {
		return 0, err
	}

	return percent, nil
}

func getKsmd() (*process.Process, error) {
	ps, err := process.Processes()
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		name, err := p.Name()
		if err != nil {
			return nil, err
		}
		if name == Ksmd {
			return p, nil
		}
	}
	return nil, fmt.Errorf("not found ksmd program")
}

func (k *ksmd) getMergeAcrossNodes() (uint64, error) {
	return readKsmPath(MergeAcrossNodes)
}

func saveKsmPath(p ksmPath, b []byte) error {
	return os.WriteFile(string(p), b, 0644)
}

func saveKsmPathByUint(p ksmPath, v uint) error {
	return saveKsmPathByUint64(p, uint64(v))
}

func saveKsmPathByUint64(p ksmPath, v uint64) error {
	return saveKsmPath(p, []byte(strconv.FormatUint(v, 10)))
}

func saveRun(v ksmdRun) error {
	return saveKsmPathByUint64(RunPath, uint64(v))
}

func readKsmPath(p ksmPath) (uint64, error) {
	b, err := os.ReadFile(string(p))
	if err != nil {
		return 0, err
	}

	d := ksmReg.Find(b)
	return strconv.ParseUint(string(d), 10, 64)
}
