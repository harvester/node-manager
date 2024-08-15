package config

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/harvester/go-common/sys"
	"github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
)

const (
	spdkStageName       = "Runtime SPDK Prerequisites"
	hugepagesPath       = "/sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages"
	hugepagesToAllocate = 1024
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

func setNrHugepages(n uint64) error {
	if err := os.WriteFile(hugepagesPath, []byte(strconv.FormatUint(n, 10)), 0644); err != nil {
		return fmt.Errorf("unable to write %d to %s: %v", n, hugepagesPath, err)
	}
	return nil
}

func getNrHugepages() (uint64, error) {
	data, err := os.ReadFile(hugepagesPath)
	if err != nil {
		return 0, fmt.Errorf("unable to read %s: %v", hugepagesPath, err)
	}
	n, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func restartKubelet() error {
	// This is safe because TryRestartService will only restart
	// services that are already running, i.e. this will restart
	// whichever of rke2-server or rke2-agent happens to be active
	// on this host
	for _, service := range []string{"rke2-server.service", "rke2-agent.service"} {
		if err := sys.TryRestartService(service); err != nil {
			return err
		}
	}
	return nil
}

func EnableV2DataEngine() error {
	origHugepages, err := getNrHugepages()
	if err != nil {
		return err
	}

	// Write the persistent config first, so we know it's saved...
	if err := UpdatePersistentOEMSettings(schema.Stage{
		Name: spdkStageName,
		Sysctl: map[string]string{
			"vm.nr_hugepages": fmt.Sprintf("%d", hugepagesToAllocate),
		},
		Commands: []string{
			"modprobe vfio_pci",
			"modprobe uio_pci_generic",
			"modprobe nvme_tcp",
		},
	}); err != nil {
		return err
	}

	// ...then try to do the runtime activation (which may not succeed)
	if err := modprobe(modulesToLoad, true); err != nil {
		return fmt.Errorf("unable to load kernel modules %v: %v", modulesToLoad, err)
	}

	if origHugepages >= hugepagesToAllocate {
		// We've already got enough hugepages, and don't want to unnecessarily
		// restart the kubelet, so no further action required
		return nil
	}

	if err := setNrHugepages(hugepagesToAllocate); err != nil {
		return err
	}

	nrHugepages, err := getNrHugepages()
	if err != nil {
		return err
	}
	if nrHugepages == hugepagesToAllocate {
		// We've successfully allocated the hugepages, but still need to restart
		// the kubelet in order for Longhorn to see the allocation.
		// TODO: handle possible corner case where setNrHugepages() succeeds but
		// getNrHugepages() fails, in which case the kubelet is never restarted.
		// One option we investigated was:
		// - Add a NodeConfigStatus.KubeletNeedsRestart flag.
		// - Set that flag to true if the kubelet needs restarting.
		// - Make OnNodeConfigChange() restart the kubelet and clear the flag.
		// Unfortunately this results in a restart loop in the single master case
		// (you can't clear the flag if the kubelet is currently restarting...)
		// Another possible corner case is where kubelet restart just fails for
		// some reason, but in this case the best (or least worst) choice
		// so far is to let the admin figure out what is causing the kubelet
		// restart to fail, fix that thing, and restart it manually.
		logrus.Infof("Restarting kubelet to set nr_hugepages=%d", hugepagesToAllocate)
		return restartKubelet()
	}

	// We didn't get enough hugepages (not enough available unfragmented memory)
	// but the system is now configured correctly so that if it's rebooted we should
	// get the required allocation.
	// TODO: record this somewhere (an event?) so that it can be picked up in the GUI
	// Note that if there aren't enough hugepages, when harvester tries to enable the
	// v2 data engine setting in Longhorn, the validator.longhorn.io admission webhook
	// will pick up the failure and an error will be displayed on the harvester settings
	// page, so we might not need to separately record this.
	logrus.Errorf("Unable to allocate %d hugepages (only got %d)", hugepagesToAllocate, nrHugepages)

	return nil
}

func DisableV2DataEngine() error {
	origHugepages, err := getNrHugepages()
	if err != nil {
		return err
	}

	// Write the persistent config first, so we know it's saved...
	if err := RemovePersistentOEMSettings(spdkStageName); err != nil {
		return err
	}

	// ...then try to do the runtime deactivation
	if err := modprobe(modulesToLoad, false); err != nil {
		return fmt.Errorf("unable to unload kernel modules %v: %v", modulesToLoad, err)
	}

	if origHugepages == 0 {
		// We already don't have any hugepages, and don't want to unnecessarily
		// restart the kubelet, so no further action required
		return nil
	}

	if err := setNrHugepages(0); err != nil {
		return err
	}

	logrus.Info("Restarting kubelet to set nr_hugepages=0")
	// TODO: see comment in EnableV2DataEngine() about possible kubectl restart failure corner case
	return restartKubelet()
}
