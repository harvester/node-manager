package config

const (
	// we use `99_settings.yaml` because it needs to be run after `90_custom.yaml`
	// with elemental works, the later change would override the previous one
	settingsOEMPath           = "/host/oem/99_settings.yaml"
	settingsOEMPathBackupPath = "/host/oem/99_settings.yaml.bak"
	yipStageInitramfs         = "initramfs"
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
