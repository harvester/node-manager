package config

import (
	"fmt"
	"os"
	"slices"

	"github.com/harvester/go-common/files"
	"github.com/harvester/node-manager/pkg/utils"
	"github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
)

const (
	// we use `99_settings.yaml` because it needs to be run after `90_custom.yaml`
	// with elemental works, the later change would override the previous one
	yipStageInitramfs = "initramfs"
)

// The following would ordinarily be const, but we need to override them in unit tests

var (
	oemPath                   = "/host/oem/"
	settingsOEMPath           = "/host/oem/99_settings.yaml"
	settingsOEMPathBackupPath = "/host/oem/99_settings.yaml.bak"
)

type NTPConfigTemplate struct {
	NTPConfigKeyValuePairs map[string]string
}

func generateNTPConfigData() string {
	return `
[Time]
{{- range $key, $value := .NTPConfigKeyValuePairs }}
{{ $key }} = {{ $value }}
{{- end }}
`
}

func UpdatePersistentOEMSettings(stage schema.Stage) error {
	_, err := os.Stat(settingsOEMPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat %s failed: %v", settingsOEMPath, err)
	}

	settings := utils.GenerateOEMTemplate()
	doBackup := true
	if os.IsNotExist(err) {
		// New file, we can just set the stages to whatever was passed in.
		settings.Stages = make(map[string][]schema.Stage)
		settings.Stages[yipStageInitramfs] = []schema.Stage{stage}
		doBackup = false
	} else {
		// Existing file, we need to load it...
		err = utils.LoadYipConfigToTarget(settingsOEMPath, settings)
		if err != nil {
			return fmt.Errorf("load %s to YIP format failed: %v", settingsOEMPath, err)
		}
		logrus.Debugf("Loaded settings from file %s, content: %+v", settingsOEMPath, settings)
		// ...then merge the new stage into whatever stages are already present,
		// either overwriting or appending as necessary.
		existingStage := slices.IndexFunc(settings.Stages[yipStageInitramfs], func(s schema.Stage) bool {
			return s.Name == stage.Name
		})
		if existingStage == -1 {
			settings.Stages[yipStageInitramfs] = append(settings.Stages[yipStageInitramfs], stage)
		} else {
			settings.Stages[yipStageInitramfs][existingStage] = stage
		}
	}

	return writePersistentOEMSettings(settings, doBackup)
}

func RemovePersistentOEMSettings(stageName string) error {
	yipConfig, err := utils.LoadYipConfig(settingsOEMPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("load %s failed: %v", settingsOEMPath, err)
	}
	logrus.Debugf("Loaded yipConfig: %+v, %p", yipConfig, yipConfig)

	if _, found := yipConfig.Stages[yipStageInitramfs]; !found {
		// this moment, we only have `initramfs` stage, so we could remove all OEM settings files.
		logrus.Infof("No `initramfs` stage found, remove all OEM settings files.")
		return files.RemoveFiles(settingsOEMPath)
	}

	pos := slices.IndexFunc(yipConfig.Stages[yipStageInitramfs], func(s schema.Stage) bool {
		return s.Name == stageName
	})

	if pos >= 0 {
		stages := yipConfig.Stages[yipStageInitramfs]
		stages = append(stages[:pos], stages[pos+1:]...)
		if len(stages) == 0 {
			logrus.Infof("No other stages found, remove all OEM settings files.")
			return files.RemoveFiles(settingsOEMPath)
		}
		yipConfig.Stages[yipStageInitramfs] = stages
	}

	// we still have other stages, so we need to backup/update OEM settings files
	return writePersistentOEMSettings(yipConfig, true)
}

func writePersistentOEMSettings(yipConfig *schema.YipConfig, doBackup bool) error {
	if doBackup {
		if _, err := files.BackupFile(settingsOEMPath); err != nil {
			return fmt.Errorf("backup %s failed: %v", settingsOEMPath, err)
		}
	}
	logrus.Infof("Prepare to update new settings to persistent files: %+v", yipConfig)
	tmpFileName, err := files.GenerateYAMLTempFileWithDir(yipConfig, "settings", oemPath)
	if err != nil {
		return fmt.Errorf("generate temp YAML file failed: %v", err)
	}
	if err = os.Rename(tmpFileName, settingsOEMPath); err != nil {
		return fmt.Errorf("rename temp file to %s failed: %v", settingsOEMPath, err)
	}
	return nil
}
