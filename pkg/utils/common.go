package utils

const (
	AnnotationNTP = "harvesterhci.io/ntp-service"
)

type NTPStatusAnnotation struct {
	NTPSyncStatus     string `json:"ntpSyncStatus"`
	CurrentNTPServers string `json:"currentNtpServers"`
}
