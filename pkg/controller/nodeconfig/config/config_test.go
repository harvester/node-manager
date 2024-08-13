package config

import (
	"os"
	"testing"

	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/utils"
	"github.com/mudler/yip/pkg/schema"

	"github.com/stretchr/testify/assert"
)

func TestNTPConfigPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	oemPath = tmpDir + "/host/oem/"
	settingsOEMPath = tmpDir + "/host/oem/99_settings.yaml"
	settingsOEMPathBackupPath = tmpDir + "/host/oem/99_settings.yaml.bak"
	if os.MkdirAll(oemPath, 0777) != nil {
		t.Errorf("Unable to create %s", oemPath)
	}

	ntpConfig := v1beta1.NTPConfig{
		NTPServers: "0.suse.pool.ntp.org 1.suse.pool.ntp.org",
	}

	// Create config for the first time
	ntpConfigHandler := NewNTPConfigHandler(nil, nil, "harvester-node-0", &ntpConfig, "")
	err := ntpConfigHandler.UpdateNTPConfigPersistence()
	assert.Nil(t, err)

	// Settings file should exist
	_, err = os.Stat(settingsOEMPath)
	assert.Nil(t, err)

	// Backup file should not exist
	_, err = os.Stat(settingsOEMPathBackupPath)
	assert.True(t, os.IsNotExist(err))

	// Should be able to load config
	yipConfig, err := utils.LoadYipConfig(settingsOEMPath)
	assert.Nil(t, err)

	// Config should be valid
	assert.Equal(t, "oem_settings", yipConfig.Name)
	// ...one top level stage ("initramfs"):
	assert.Equal(t, 1, len(yipConfig.Stages))
	assert.Contains(t, yipConfig.Stages, yipStageInitramfs)
	// ...which in turn has one stage inside ("ntp"):
	assert.Equal(t, 1, len(yipConfig.Stages[yipStageInitramfs]))
	assert.Equal(t, "ntp", yipConfig.Stages[yipStageInitramfs][0].Name)
	// ...and the NTP servers are set as we expect:
	assert.Equal(t, map[string]string{"NTP": ntpConfig.NTPServers}, yipConfig.Stages[yipStageInitramfs][0].TimeSyncd)

	// Update config with new servers
	newNtpConfig := v1beta1.NTPConfig{
		NTPServers: "something different set of servers",
	}
	ntpConfigHandler.NTPConfig = reGenerateNTPConfig(&newNtpConfig)
	err = ntpConfigHandler.UpdateNTPConfigPersistence()
	assert.Nil(t, err)

	// Backup file should exist
	_, err = os.Stat(settingsOEMPathBackupPath)
	assert.False(t, os.IsNotExist(err))

	// Backup config should be the same as the previous config
	backupConfig, err := utils.LoadYipConfig(settingsOEMPathBackupPath)
	assert.Nil(t, err)
	assert.Equal(t, yipConfig, backupConfig)

	// New config should have new NTP servers
	newConfig, err := utils.LoadYipConfig(settingsOEMPath)
	assert.Nil(t, err)
	assert.Equal(t, map[string]string{"NTP": newNtpConfig.NTPServers}, newConfig.Stages[yipStageInitramfs][0].TimeSyncd)

	// Remove the NTP settings
	err = RemovePersistentNTPConfig()
	assert.Nil(t, err)

	// Settings file should be gone
	_, err = os.Stat(settingsOEMPath)
	assert.True(t, os.IsNotExist(err))

	// Backup file should remain
	_, err = os.Stat(settingsOEMPathBackupPath)
	assert.False(t, os.IsNotExist(err))
}

func TestExtraConfigPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	oemPath = tmpDir + "/host/oem/"
	settingsOEMPath = tmpDir + "/host/oem/99_settings.yaml"
	settingsOEMPathBackupPath = tmpDir + "/host/oem/99_settings.yaml.bak"
	if os.MkdirAll(oemPath, 0777) != nil {
		t.Errorf("Unable to create %s", oemPath)
	}

	ntpConfig := v1beta1.NTPConfig{
		NTPServers: "0.suse.pool.ntp.org 1.suse.pool.ntp.org",
	}

	// Create config for the first time, with NTP as in TestNTPConfigPersistence()
	ntpConfigHandler := NewNTPConfigHandler(nil, nil, "harvester-node-0", &ntpConfig, "")
	err := ntpConfigHandler.UpdateNTPConfigPersistence()
	assert.Nil(t, err)

	// Settings file should exist
	_, err = os.Stat(settingsOEMPath)
	assert.Nil(t, err)

	// Backup file should not exist
	_, err = os.Stat(settingsOEMPathBackupPath)
	assert.True(t, os.IsNotExist(err))

	// Add an extra stage
	extraStage := schema.Stage{
		Name: "extra",
	}
	err = UpdatePersistentOEMSettings(extraStage)
	assert.Nil(t, err)

	// Backup file should exist
	_, err = os.Stat(settingsOEMPathBackupPath)
	assert.False(t, os.IsNotExist(err))

	// Should be able to load config
	yipConfig, err := utils.LoadYipConfig(settingsOEMPath)
	assert.Nil(t, err)

	// Config should be valid
	assert.Equal(t, "oem_settings", yipConfig.Name)
	// ...one top level stage ("initramfs"):
	assert.Equal(t, 1, len(yipConfig.Stages))
	assert.Contains(t, yipConfig.Stages, yipStageInitramfs)
	// ...which in turn has _two_ stages inside ("ntp" and "extra"):
	assert.Equal(t, 2, len(yipConfig.Stages[yipStageInitramfs]))
	assert.Equal(t, "ntp", yipConfig.Stages[yipStageInitramfs][0].Name)
	assert.Equal(t, map[string]string{"NTP": ntpConfig.NTPServers}, yipConfig.Stages[yipStageInitramfs][0].TimeSyncd)
	assert.Equal(t, "extra", yipConfig.Stages[yipStageInitramfs][1].Name)

	// Should be able to update the extra stage
	newExtraStage := schema.Stage{
		Name:     "extra",
		Commands: []string{"/bin/true"},
	}
	err = UpdatePersistentOEMSettings(newExtraStage)
	assert.Nil(t, err)

	// Should be able to load config and see the updated commands in the extra stage,
	// while the ntp stage should still be there unchanged
	yipConfig, err = utils.LoadYipConfig(settingsOEMPath)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(yipConfig.Stages))
	assert.Contains(t, yipConfig.Stages, yipStageInitramfs)
	assert.Equal(t, 2, len(yipConfig.Stages[yipStageInitramfs]))
	assert.Equal(t, "ntp", yipConfig.Stages[yipStageInitramfs][0].Name)
	assert.Equal(t, map[string]string{"NTP": ntpConfig.NTPServers}, yipConfig.Stages[yipStageInitramfs][0].TimeSyncd)
	assert.Equal(t, "extra", yipConfig.Stages[yipStageInitramfs][1].Name)
	assert.Equal(t, newExtraStage.Commands, yipConfig.Stages[yipStageInitramfs][1].Commands)

	// Remove the NTP settings
	err = RemovePersistentNTPConfig()
	assert.Nil(t, err)

	// Read the config again and we should only have the extra stage now
	yipConfig, err = utils.LoadYipConfig(settingsOEMPath)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(yipConfig.Stages))
	assert.Contains(t, yipConfig.Stages, yipStageInitramfs)
	assert.Equal(t, 1, len(yipConfig.Stages[yipStageInitramfs]))
	assert.Equal(t, "extra", yipConfig.Stages[yipStageInitramfs][0].Name)

	// Remove the extra stage
	err = RemovePersistentOEMSettings("extra")
	assert.Nil(t, err)

	// Settings file should be gone
	_, err = os.Stat(settingsOEMPath)
	assert.True(t, os.IsNotExist(err))

	// Backup file should remain
	_, err = os.Stat(settingsOEMPathBackupPath)
	assert.False(t, os.IsNotExist(err))
}
