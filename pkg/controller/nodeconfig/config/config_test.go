package config

import (
	"os"
	"testing"

	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/utils"

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

	// Backup file should be gone
	_, err = os.Stat(settingsOEMPathBackupPath)
	assert.True(t, os.IsNotExist(err))
}
