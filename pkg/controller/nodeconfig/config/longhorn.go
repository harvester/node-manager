package config

import (
	"fmt"
	"os/exec"

	"github.com/mudler/yip/pkg/schema"
)

const (
	spdkStageName = "Runtime SPDK Prerequisites"
)

var (
	modulesToLoad = []string{"vfio_pci", "uio_pci_generic", "nvme_tcp"}
)

func modprobe(modules []string, load bool) error {
	args := []string{"-a"}
	if !load {
		args = append(args, "-r")
	}
	args = append(args, modules...)
	out, err := exec.Command("/usr/sbin/modprobe", args...).CombinedOutput()
	if err != nil {
		// This ensures we capture some helpful information if modules can't
		// be loaded.  For example, if /lib/modules isn't actually mounted in
		// the container, we'll see something like this:
		//   modprobe failed: exit status 1 (output: 'modprobe: WARNING: Module
		//   vfio_pci not found in directory /lib/modules/5.14.21-150500.55.68-default[...]')
		return fmt.Errorf("modprobe failed: %v (output: '%s')", err, out)
	}
	return nil
}

func updateLonghornConfigPersistence() error {
	stage := schema.Stage{
		Name:     spdkStageName,
		Commands: []string{},
	}

	for _, module := range modulesToLoad {
		stage.Commands = append(stage.Commands, fmt.Sprintf("modprobe %s", module))
	}

	return UpdatePersistentOEMSettings(stage)
}

func EnableV2DataEngine() error {
	// Write the persistent config first, so we know it's saved...
	if err := updateLonghornConfigPersistence(); err != nil {
		return err
	}

	// ...then try to do the runtime activation (which may not succeed)
	if err := modprobe(modulesToLoad, true); err != nil {
		return fmt.Errorf("unable to load kernel modules %v: %v", modulesToLoad, err)
	}

	return nil
}

func DisableV2DataEngine() error {
	return RemovePersistentOEMSettings(spdkStageName)
}
