package hugepage

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/procfs"
	"github.com/rancher/wrangler/v3/pkg/ticker"
	"github.com/sirupsen/logrus"

	nodev1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

const (
	MonitorInterval     = time.Second * 30
	THPEnabledPath      = "/sys/kernel/mm/transparent_hugepage/enabled"
	THPShmemEnabledPath = "/sys/kernel/mm/transparent_hugepage/shmem_enabled"
	THPDefragPath       = "/sys/kernel/mm/transparent_hugepage/defrag"
)

var (
	manager *Manager
)

type Manager struct {
	ctx context.Context

	statCh chan *nodev1beta1.HugepageStatus
	specCh chan *nodev1beta1.HugepageSpec

	procFs procfs.FS
}

func NewHugepageManager(ctx context.Context) *Manager {
	procFs, _ := procfs.NewFS("/proc")

	manager = &Manager{
		ctx:    ctx,
		statCh: make(chan *nodev1beta1.HugepageStatus, 10),
		specCh: make(chan *nodev1beta1.HugepageSpec, 1),
		procFs: procFs,
	}
	go manager.run()
	return manager
}

func (h *Manager) run() {
	t := ticker.Context(h.ctx, MonitorInterval)
	for {
		select {
		case s := <-h.specCh:
			logrus.Debug("updating hugepage settings")
			if err := h.writeTHPConfig(&s.Transparent); err != nil {
				logrus.Errorf("failed to update transparent hugepage config: %v", err)
			}
		case <-t:
			logrus.Debug("updating hugepage status")
		case <-h.ctx.Done():
			return
		}

		if err := h.updateStatus(); err != nil {
			logrus.Errorf("failed to update hugepage status: %v", err)
		}
	}
}

func (h *Manager) updateStatus() error {
	meminfo, err := h.readProcMeminfo()
	if err != nil {
		return err
	}

	thpConfig, err := h.readTHPConfig()
	if err != nil {
		return err
	}

	h.statCh <- &nodev1beta1.HugepageStatus{
		Transparent: *thpConfig,
		Meminfo: nodev1beta1.Meminfo{
			AnonHugePages:  *meminfo.AnonHugePagesBytes,
			ShmemHugePages: *meminfo.ShmemHugePagesBytes,
			HugePagesTotal: *meminfo.HugePagesTotal,
			HugePagesFree:  *meminfo.HugePagesFree,
			HugePagesRsvd:  *meminfo.HugePagesRsvd,
			HugePagesSurp:  *meminfo.HugePagesSurp,
			HugepageSize:   *meminfo.HugepagesizeBytes,
		},
	}
	return nil
}

func (h *Manager) GetStatusChan() <-chan *nodev1beta1.HugepageStatus {
	return h.statCh
}

func (h *Manager) GetSpecChan() chan<- *nodev1beta1.HugepageSpec {
	return h.specCh
}

// GetDefaultTHPConfig returns the system's current THP config, or if that
// fails, it returns sensible default values. This is to be used when first
// creating a Hugepage CR for a node so that the initial spec reflects the
// current state of the system.
func (h *Manager) GetDefaultTHPConfig() *nodev1beta1.THPConfig {
	config, err := h.readTHPConfig()
	if err != nil {
		logrus.Warnf("failed to read current THP config: %v, falling back to default settings", err)
		return &nodev1beta1.THPConfig{
			Enabled:      nodev1beta1.THPEnabledAlways,
			ShmemEnabled: nodev1beta1.THPShmemEnabledNever,
			Defrag:       nodev1beta1.THPDefragMadvise,
		}
	}
	return config
}

func (h *Manager) readProcMeminfo() (*procfs.Meminfo, error) {
	meminfo, err := h.procFs.Meminfo()
	if err != nil {
		logrus.Errorf("failed to read procfs: %v", err)
		return nil, err
	}

	return &meminfo, nil
}

func (h *Manager) readTHPConfig() (*nodev1beta1.THPConfig, error) {
	enabledLine, err := h.read(THPEnabledPath)
	if err != nil {
		return nil, err
	}
	enabled, err := parse(enabledLine)
	if err != nil {
		return nil, err
	}

	shmemEnabledLine, err := h.read(THPShmemEnabledPath)
	if err != nil {
		return nil, err
	}
	shmemEnabled, err := parse(shmemEnabledLine)
	if err != nil {
		return nil, err
	}

	defragLine, err := h.read(THPDefragPath)
	if err != nil {
		return nil, err
	}
	defrag, err := parse(defragLine)
	if err != nil {
		return nil, err
	}

	return &nodev1beta1.THPConfig{
		Enabled:      nodev1beta1.THPEnabled(enabled),
		ShmemEnabled: nodev1beta1.THPShmemEnabled(shmemEnabled),
		Defrag:       nodev1beta1.THPDefrag(defrag),
	}, nil
}

func (h *Manager) readSysfsUint64(path string) (uint64, error) {
	rawStr, err := h.read(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read path %v: %v", path, err)
	}

	num, err := strconv.ParseUint(strings.TrimSpace(rawStr), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %v: %v", rawStr, err)
	}
	return num, nil
}

func (h *Manager) writeTHPConfig(thp *nodev1beta1.THPConfig) error {
	if err := h.write(THPEnabledPath, string(thp.Enabled)); err != nil {
		return err
	}
	if err := h.write(THPShmemEnabledPath, string(thp.ShmemEnabled)); err != nil {
		return err
	}
	if err := h.write(THPDefragPath, string(thp.Defrag)); err != nil {
		return err
	}
	return nil
}

func (h *Manager) read(path string) (string, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (h *Manager) write(path, value string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(value)
	if err != nil {
		return err
	}
	f.Sync()
	return nil
}

func parse(line string) (string, error) {
	_, str1, fnd := strings.Cut(line, "[")
	if !fnd {
		return "", fmt.Errorf("could not parse sysfs setting: %v", str1)
	}
	str2, _, fnd := strings.Cut(str1, "]")
	if !fnd {
		return "", fmt.Errorf("could not parse sysfs setting: %v", str2)
	}
	return str2, nil
}
