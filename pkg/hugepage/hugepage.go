package hugepage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mudler/yip/pkg/schema"
	"github.com/prometheus/procfs"
	"github.com/sirupsen/logrus"

	"github.com/harvester/node-manager/pkg/controller/nodeconfig/config"

	nodev1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

const (
	THPPath             = "/sys/kernel/mm/transparent_hugepage/"
	THPEnabledFile      = "enabled"
	THPShmemEnabledFile = "shmem_enabled"
	THPDefragFile       = "defrag"

	SHPPath         = "/sys/kernel/mm/hugepages/hugepages-2048kB/"
	NrHugepagesFile = "nr_hugepages"

	hugepageStageName = "Hugepage Configuration"
)

var (
	manager *Manager
)

type Manager struct {
	ctx     context.Context
	procFs  procfs.FS
	thpPath string
	shpPath string
}

func NewHugepageManager(ctx context.Context, thpPath, shpPath string) (*Manager, error) {
	procFs, err := procfs.NewFS("/proc")
	if err != nil {
		return nil, fmt.Errorf("error initialising hugepage manager: %w", err)
	}
	manager = &Manager{
		ctx:     ctx,
		procFs:  procFs,
		thpPath: thpPath,
		shpPath: shpPath,
	}
	return manager, nil
}

func (h *Manager) GenerateStatus() (*nodev1beta1.HugepageStatus, error) {
	meminfo, err := h.readProcMeminfo()
	if err != nil {
		return nil, err
	}

	thpConfig, err := h.readTHPConfig()
	if err != nil {
		return nil, err
	}

	shpConfig, err := h.readSHPConfig()
	if err != nil {
		return nil, err
	}

	return &nodev1beta1.HugepageStatus{
		Transparent: *thpConfig,
		Static:      *shpConfig,
		Meminfo: nodev1beta1.Meminfo{
			AnonHugePages:  *meminfo.AnonHugePagesBytes,
			ShmemHugePages: *meminfo.ShmemHugePagesBytes,
			HugePagesTotal: *meminfo.HugePagesTotal,
			HugePagesFree:  *meminfo.HugePagesFree,
			HugePagesRsvd:  *meminfo.HugePagesRsvd,
			HugePagesSurp:  *meminfo.HugePagesSurp,
			HugepageSize:   *meminfo.HugepagesizeBytes,
		},
	}, nil
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
	enabledLine, err := h.read(filepath.Join(h.thpPath, THPEnabledFile))
	if err != nil {
		return nil, err
	}
	enabled, err := parse(enabledLine)
	if err != nil {
		return nil, err
	}

	shmemEnabledLine, err := h.read(filepath.Join(h.thpPath, THPShmemEnabledFile))
	if err != nil {
		return nil, err
	}
	shmemEnabled, err := parse(shmemEnabledLine)
	if err != nil {
		return nil, err
	}

	defragLine, err := h.read(filepath.Join(h.thpPath, THPDefragFile))
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

func (h *Manager) readSHPConfig() (*nodev1beta1.SHPConfig, error) {
	nrHugepagesLine, err := h.read(filepath.Join(h.shpPath, NrHugepagesFile))
	if err != nil {
		return nil, err
	}
	nrHugepages, err := strconv.ParseUint(strings.TrimSpace(nrHugepagesLine), 10, 64)
	if err != nil {
		return nil, err
	}
	return &nodev1beta1.SHPConfig{
		NrHugepages: nrHugepages,
	}, nil
}

func (h *Manager) UpdateHugepageConfigPersistence(hp *nodev1beta1.HugepageSpec) error {
	stage := schema.Stage{
		Name: hugepageStageName,
		Commands: []string{
			fmt.Sprintf("echo %s > %s", string(hp.Transparent.Enabled), filepath.Join(h.thpPath, THPEnabledFile)),
			fmt.Sprintf("echo %s > %s", string(hp.Transparent.ShmemEnabled), filepath.Join(h.thpPath, THPShmemEnabledFile)),
			fmt.Sprintf("echo %s > %s", string(hp.Transparent.Defrag), filepath.Join(h.thpPath, THPDefragFile)),
		},
		Sysctl: map[string]string{"vm.nr_hugepages": fmt.Sprintf("%d", hp.Static.NrHugepages)},
	}
	return config.UpdatePersistentOEMSettings(stage)
}

func (h *Manager) ApplyConfig(hp *nodev1beta1.HugepageSpec) error {
	if err := h.write(filepath.Join(h.shpPath, NrHugepagesFile), strconv.FormatUint(hp.Static.NrHugepages, 10)); err != nil {
		return err
	}
	if err := h.write(filepath.Join(h.thpPath, THPEnabledFile), string(hp.Transparent.Enabled)); err != nil {
		return err
	}
	if err := h.write(filepath.Join(h.thpPath, THPShmemEnabledFile), string(hp.Transparent.ShmemEnabled)); err != nil {
		return err
	}
	if err := h.write(filepath.Join(h.thpPath, THPDefragFile), string(hp.Transparent.Defrag)); err != nil {
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
	return f.Sync()
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
