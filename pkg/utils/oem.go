package utils

import (
	"fmt"

	"github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
	govfs "github.com/twpayne/go-vfs"
	"gopkg.in/yaml.v1"
)

var validateStages = []string{
	"rootfs",
	"initramfs",
	"boot",
	"fs",
	"network",
	"reconcile",
	"post-install",
	"after-install-chroot",
	"after-install",
	"post-upgrade",
	"after-upgrade-chroot",
	"after-upgrade",
	"post-reset",
	"after-reset-chroot",
	"after-reset",
	"before-install",
	"before-upgrade",
	"before-reset",
}

func validateStage(stageName string) bool {
	if stageName == "" {
		return false
	}
	for _, stage := range validateStages {
		if stageName == stage {
			return true
		}
	}
	return false
}

// return the empty OEM template for later use
func GenerateOEMTemplate() *schema.YipConfig {
	return &schema.YipConfig{
		Name: "oem_settings",
	}
}

func UpdateToOEMTemplate(stageName string, stage schema.Stage, oem *schema.YipConfig) (*schema.YipConfig, error) {
	if !validateStage(stageName) {
		return nil, fmt.Errorf("invalid stage name: %s", stageName)
	}
	oem.Stages = map[string][]schema.Stage{
		stageName: {stage},
	}
	return oem, nil
}

func LoadYipConfig(path string) (*schema.YipConfig, error) {
	yipConfig := GenerateOEMTemplate()
	yipConfig.Stages = make(map[string][]schema.Stage)
	err := LoadYipConfigToTarget(path, yipConfig)
	if err != nil {
		return nil, err
	}
	return yipConfig, nil
}

func LoadYipConfigToTarget(path string, config *schema.YipConfig) error {
	osfs := govfs.OSFS
	bytes, err := osfs.ReadFile(path)
	if err != nil {
		logrus.Errorf("osfs readfile failed. err: %v", err)
		return err
	}
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		logrus.Errorf("Unmarshal failed. err: %v", err)
		return err
	}
	logrus.Debugf("In function loaded settings: %+v", config)
	return nil
}
