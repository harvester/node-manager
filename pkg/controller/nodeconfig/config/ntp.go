package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"

	gocommon "github.com/harvester/go-common"
	"github.com/mudler/yip/pkg/schema"
	ctlnode "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nodeconfigv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	ctlv1 "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io/v1beta1"
	"github.com/harvester/node-manager/pkg/utils"
)

const (
	NTPName                   = "ntp"
	systemdTimesyncdService   = "systemd-timesyncd.service"
	timesyncdConfigPath       = "/host/etc/systemd/timesyncd.conf"
	timesyncdConfigBackupPath = "/host/etc/systemd/timesyncd.conf.bak"
	timesyncdConfigOriginPath = "/host/etc/systemd/timesyncd.conf.origin"
	timesyncdService          = "systemd-timesyncd"
	timeWaitSyncService       = "systemd-time-wait-sync"
	oemPath                   = "/host/oem/"
	configNTPServer           = "ntpServer"
)

type NTPStatusAnnotation utils.NTPStatusAnnotation

type NTPHandler struct {
	NTPConfig      *nodeconfigv1.NTPConfig
	AppliedConfigs string // AppliedConfigs is a json format string, you should unmarshal it to AppliedConfigAnnotation
	NodeClient     ctlnode.NodeClient
	ConfName       string
	mtx            *sync.Mutex
}

func NewNTPConfigHandler(mtx *sync.Mutex, nodes ctlnode.NodeController, confName string, ntpconfigs *nodeconfigv1.NTPConfig, appliedConfig string) *NTPHandler {
	newntpconfigs := reGenerateNTPConfig(ntpconfigs)
	return &NTPHandler{
		NTPConfig:      newntpconfigs,
		AppliedConfigs: appliedConfig,
		NodeClient:     nodes,
		ConfName:       confName,
		mtx:            mtx,
	}
}

// DoNTPUpdate will backup and update NTP to system, return bool for restart service and generic error
func (handler *NTPHandler) DoNTPUpdate(forceUpdate bool) (bool, error) {
	var content nodeconfigv1.AppliedConfigAnnotation
	if !forceUpdate && handler.AppliedConfigs != "" {
		logrus.Infof("Found applied config from annotation: %s", handler.AppliedConfigs)
		err := json.Unmarshal([]byte(handler.AppliedConfigs), &content)
		if err != nil {
			logrus.Warnf("Unmarshal applied config from annotation failed, assume that is empty err: %v", err)
		}

		if content.NTPServers == handler.NTPConfig.NTPServers {
			return false, nil
		}
	}

	// if the incoming NTPServers is empty but we have annotation, we should remove the NTP config
	if handler.NTPConfig.NTPServers == "" && handler.AppliedConfigs == "" {
		return false, nil
	}

	_, err := os.Stat(timesyncdConfigOriginPath)
	if os.IsNotExist(err) {
		logrus.Infof("Backup original ntp config ...")
		if _, err := gocommon.BackupFileToDirWithSuffix(timesyncdConfigPath, "", "origin"); err != nil {
			return false, fmt.Errorf("backup the original ntp config failed. err: %v", err)
		}
	}

	logrus.Infof("Backup current ntp config ...")
	if err := handler.backupNTPConfig(); err != nil {
		return false, fmt.Errorf("backup NTP config failed, skip this round. err: %v", err)
	}

	logrus.Infof("Prepare to update NTP server with: %s", handler.NTPConfig.NTPServers)
	if err := handler.updateNTPConfig(); err != nil {
		return false, fmt.Errorf("update NTP config failed, skip this round. err: %v", err)
	}

	return true, nil
}

func generateNTPConfigTemplate(servers string) *NTPConfigTemplate {
	return &NTPConfigTemplate{
		NTPConfigKeyValuePairs: map[string]string{
			"NTP": servers,
		},
	}
}

func (handler *NTPHandler) generateNTPConfigRawString() (string, error) {
	conf := generateNTPConfigTemplate(handler.NTPConfig.NTPServers)

	tmpl, err := template.New("ntp").Parse(generateNTPConfigData())
	if err != nil {
		return "", err
	}
	buf := bytes.NewBufferString("")
	err = tmpl.Execute(buf, conf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// updateNTPConfig write the tempfile first then rename to the target.
func (handler *NTPHandler) updateNTPConfig() error {
	raw, err := handler.generateNTPConfigRawString()
	if err != nil {
		return fmt.Errorf("generate NTP Config Raw Buffer failed. err: %v", err)
	}

	tempNTPConfigName, err := gocommon.GenerateTempFileWithDir([]byte(raw), "timesyncd.conf", utils.SystemdConfigPath)
	if err != nil {
		return fmt.Errorf("generate temp NTP config failed. err: %v", err)
	}

	if err := os.Rename(tempNTPConfigName, timesyncdConfigPath); err != nil {
		return fmt.Errorf("rename temp NTP config failed. err: %v", err)
	}

	return nil
}

func (handler *NTPHandler) UpdateNodeNTPAnnotation() error {
	logrus.Debugf("Prepare to update currentNTPServer for node annotation: %s", handler.ConfName)
	handler.mtx.Lock()
	defer handler.mtx.Unlock()
	node, err := handler.NodeClient.Get(handler.ConfName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("Get node %s failed. err: %v", handler.ConfName, err)
		return err
	}

	if _, found := node.Annotations[utils.AnnotationNTP]; !found {
		logrus.Debugf("First update should be done by monitor, skip!")
		return nil
	}
	annoNTPValue := node.Annotations[utils.AnnotationNTP]
	var ntpValue NTPStatusAnnotation
	if err := json.Unmarshal([]byte(annoNTPValue), &ntpValue); err != nil {
		logrus.Errorf("Unmarshal annotation value failed. err: %v", err)
		return err
	}
	logrus.Debugf("Current annotation value: %+v", ntpValue)

	ntpValue.CurrentNTPServers = handler.NTPConfig.NTPServers

	bytes, err := json.Marshal(ntpValue)
	if err != nil {
		logrus.Errorf("Marshal annotation value fail, skip this round NTP check. err: %v", err)
		return err
	}

	nodeCpy := node.DeepCopy()
	nodeCpy.Annotations[utils.AnnotationNTP] = string(bytes)
	if !reflect.DeepEqual(node, nodeCpy) {
		handler.NodeClient.Update(nodeCpy)
	}
	return nil
}

func (handler *NTPHandler) backupNTPConfig() error {
	if _, err := gocommon.BackupFile(timesyncdConfigPath); err != nil {
		return fmt.Errorf("backup NTP config failed. err: %v", err)
	}
	return nil
}

// Rollback the NTPConfig do not need to any related config and it may called by OnRemove
func NTPConfigRollback() error {
	if _, err := os.Stat(timesyncdConfigOriginPath); err != nil {
		return fmt.Errorf("check original NTP config error. Please ensure the original config exists. err: %v", err)
	}
	if _, err := os.Stat(timesyncdConfigPath); err != nil {
		return fmt.Errorf("check current NTP config error. Please ensure the current config exists. err: %v", err)
	}

	src, err := os.Open(timesyncdConfigOriginPath)
	if err != nil {
		return fmt.Errorf("open NTP config origin file failed. err: %v", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(timesyncdConfigPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open NTP config file failed. err: %v", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func (handler *NTPHandler) RestartService() error {
	logrus.Infof("Restart systemd-timesyncd service ...")
	return gocommon.RestartService(systemdTimesyncdService)
}

// make NTP configuration persistence, using 99_settings.yaml to make sure we are later than 99_oem.yaml
func (handler *NTPHandler) UpdateNTPConfigPersistence() error {
	logrus.Infof("Prepare to make NTP configuration persistence ...")
	ntpServer := handler.NTPConfig.NTPServers
	_, err := os.Stat(settingsOEMPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat NTP OEM file failed. err: %v", err)
	}

	ntpStages := generateNTPStages(ntpServer)
	settings := utils.GenerateOEMTemplate()
	settings.Stages = make(map[string][]schema.Stage)
	if os.IsNotExist(err) {
		if _, found := settings.Stages[yipStageInitramfs]; !found {
			settings.Stages[yipStageInitramfs] = []schema.Stage{ntpStages}
		} else {
			settings.Stages[yipStageInitramfs] = append(settings.Stages[yipStageInitramfs], ntpStages)
		}
	} else {
		// backup current config
		if _, err := gocommon.BackupFile(settingsOEMPath); err != nil {
			return fmt.Errorf("backup NTP OEM file failed. err: %v", err)
		}
		// load and overwrite
		err = utils.LoadYipConfigToTarget(settingsOEMPath, settings)
		if err != nil {
			return fmt.Errorf("load OEM file failed to YIP format. err: %v", err)
		}
		logrus.Debugf("Loaded settings from file %s, content: %+v", settingsOEMPath, settings)
		currentInitramfs := settings.Stages[yipStageInitramfs]
		for id, stage := range currentInitramfs {
			if stage.Name == NTPName {
				currentInitramfs[id] = ntpStages
				break
			}
		}
	}
	logrus.Infof("Prepare to update settings to persistent file: %+v", settings)
	tmpFileName, err := gocommon.GenerateYAMLTempFileWithDir(settings, "settings", oemPath)
	if err != nil {
		return fmt.Errorf("generate temp YAML file failed. err: %v", err)
	}
	if err = os.Rename(tmpFileName, settingsOEMPath); err != nil {
		return fmt.Errorf("rename temp file to OEM path failed. err: %v", err)
	}

	return nil
}

func generateNTPStages(ntpserver string) schema.Stage {
	return schema.Stage{
		Name: NTPName,
		TimeSyncd: map[string]string{
			"NTP": ntpserver,
		},
		Systemctl: schema.Systemctl{
			Enable: []string{"systemd-timesyncd", "systemd-time-wait-sync"},
		},
	}
}

func RemovePersistentNTPConfig() error {
	yipConfig, err := utils.LoadYipConfig(settingsOEMPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("load OEM settings failed. err: %v", err)
	}
	logrus.Debugf("Loaded yipConfig: %+v, %p", yipConfig, yipConfig)

	pos := -1
	if _, found := yipConfig.Stages[yipStageInitramfs]; !found {
		// this moment, we only have `initramfs` stage, so we could remove all OEM settings files.
		logrus.Infof("No `initramfs` stage found, remove all OEM settings files.")
		return gocommon.RemoveFiles(settingsOEMPath, settingsOEMPathBackupPath)
	}

	for id, stage := range yipConfig.Stages[yipStageInitramfs] {
		if stage.Name == NTPName {
			pos = id
			break
		}
	}

	if pos >= 0 {
		stages := yipConfig.Stages[yipStageInitramfs]
		stages = append(stages[:pos], stages[pos+1:]...)
		if len(stages) == 0 {
			logrus.Infof("No other stages found, remove all OEM settings files.")
			return gocommon.RemoveFiles(settingsOEMPath, settingsOEMPathBackupPath)
		}
		yipConfig.Stages[yipStageInitramfs] = stages
	}

	// we still have other stages, so we need to backup/update OEM settings files
	if _, err := gocommon.BackupFile(settingsOEMPath); err != nil {
		return fmt.Errorf("backup NTP OEM file failed. err: %v", err)
	}

	logrus.Infof("Prepare to update new settings to persistent files: %+v", yipConfig)
	tmpFileName, err := gocommon.GenerateYAMLTempFileWithDir(yipConfig, "settings", oemPath)
	if err != nil {
		return fmt.Errorf("generate temp YAML file failed. err: %v", err)
	}
	if err = os.Rename(tmpFileName, settingsOEMPath); err != nil {
		return fmt.Errorf("rename temp file to OEM path failed. err: %v", err)
	}
	return nil
}

func CheckConfigApplied(configName string, status nodeconfigv1.NodeConfigStatus) bool {
	switch configName {
	case "ntp":
		return ntpConfigWaitApplied(status)
	default:
		logrus.Warnf("Unknown config name: %s, we should not be blocked here", configName)
		return false
	}
}

func ntpConfigWaitApplied(status nodeconfigv1.NodeConfigStatus) bool {
	// apply NTP config first time
	var configModified, configApplied bool
	if status.NTPConditions == nil {
		return true
	}

	for _, cond := range status.NTPConditions {
		if cond.Type == nodeconfigv1.ConfigModified {
			configModified = cond.Status == corev1.ConditionTrue
		}
		if cond.Type == nodeconfigv1.ConfigApplied {
			configApplied = cond.Status == corev1.ConditionTrue
		}
	}
	return (configModified && !configApplied)
}

// conds: ConfigModified : False + ConfigApplied : True
func UpdateNTPConfigApplied(nodecfgctl ctlv1.NodeConfigController, nodecfg *nodeconfigv1.NodeConfig) (*nodeconfigv1.NodeConfig, error) {
	conds := []nodeconfigv1.ConfigStatus{
		NewNTPConfigModifiedCondition(corev1.ConditionFalse),
		NewNTPConfigAppliedCondition(corev1.ConditionTrue),
	}
	nodecfgCpy := nodecfg.DeepCopy()
	if err := updateNodeConfigStatus(NTPName, &nodecfgCpy.Status, conds); err != nil {
		logrus.Errorf("Update NTP config status to applied failed. err: %v", err)
		return nil, err
	}
	if !reflect.DeepEqual(nodecfg.Status, nodecfgCpy.Status) {
		return nodecfgctl.UpdateStatus(nodecfgCpy)
	}
	return nodecfg, nil
}

// conds: ConfigModified : True + ConfigApplied : False
func UpdateNTPConfigChanged(nodecfgctl ctlv1.NodeConfigController, nodecfg *nodeconfigv1.NodeConfig) (*nodeconfigv1.NodeConfig, error) {
	conds := []nodeconfigv1.ConfigStatus{
		NewNTPConfigModifiedCondition(corev1.ConditionTrue),
		NewNTPConfigAppliedCondition(corev1.ConditionFalse),
	}
	nodecfgCpy := nodecfg.DeepCopy()
	if err := updateNodeConfigStatus(NTPName, &nodecfgCpy.Status, conds); err != nil {
		logrus.Errorf("Update NTP config status to changed failed. err: %v", err)
		return nil, err
	}
	if !reflect.DeepEqual(nodecfg.Status, nodecfgCpy.Status) {
		logrus.Infof("DEBUG: update status: %+v", nodecfgCpy.Status)
		return nodecfgctl.UpdateStatus(nodecfgCpy)
	}
	return nodecfg, nil
}

func NewNTPConfigModifiedCondition(status corev1.ConditionStatus) nodeconfigv1.ConfigStatus {
	return nodeconfigv1.ConfigStatus{
		Type:    nodeconfigv1.ConfigModified,
		Status:  status,
		Reason:  "NTPConfigModified",
		Message: "NTP config is created or modified, need to wait it to be applied",
	}
}

func NewNTPConfigAppliedCondition(status corev1.ConditionStatus) nodeconfigv1.ConfigStatus {
	return nodeconfigv1.ConfigStatus{
		Type:    nodeconfigv1.ConfigApplied,
		Status:  status,
		Reason:  "NTPConfigApplied",
		Message: "NTP config is Applied",
	}
}

func updateNodeConfigStatus(configName string, status *nodeconfigv1.NodeConfigStatus, conds []nodeconfigv1.ConfigStatus) error {
	switch configName {
	case "ntp":
		for _, cond := range conds {
			status.NTPConditions = updateCondition(status.NTPConditions, cond)
		}
	default:
		return fmt.Errorf("not supported config name: %s", configName)
	}
	return nil
}

func updateCondition(conditions []nodeconfigv1.ConfigStatus, c nodeconfigv1.ConfigStatus) []nodeconfigv1.ConfigStatus {
	found := false
	var pos = 0
	logrus.Debugf("Prepare to check the coming Type: %s, Status: %s", c.Type, c.Status)
	for id, condition := range conditions {
		logrus.Debugf("condition.Type: %s, condition.Status: %s", condition.Type, condition.Status)
		if condition.Type == c.Type {
			found = true
			pos = id
			break
		}
	}

	if found {
		logrus.Debugf("found Current Type: %s, condition.Status: %s", conditions[pos].Type, conditions[pos].Status)
		if conditions[pos].Status != c.Status {
			conditions[pos] = c
		}
	} else {
		conditions = append(conditions, c)
	}
	return conditions
}

func reGenerateNTPConfig(ntpconfigs *nodeconfigv1.NTPConfig) *nodeconfigv1.NTPConfig {
	if ntpconfigs.NTPServers == "" {
		return ntpconfigs
	}

	// fileter the duplicated NTP servers
	currentNTPServers := strings.Split(ntpconfigs.NTPServers, " ")
	parsedNTPServers := make([]string, 0)
	for _, ntpServer := range currentNTPServers {
		if !slices.Contains(parsedNTPServers, ntpServer) {
			parsedNTPServers = append(parsedNTPServers, ntpServer)
		}
	}
	return &nodeconfigv1.NTPConfig{
		NTPServers: strings.Join(parsedNTPServers, " "),
	}

}
