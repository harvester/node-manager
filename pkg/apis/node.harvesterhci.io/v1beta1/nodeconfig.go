package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConfigsApplied     = "Applied"
	ConfigsWaitApplied = "WaitApplied"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName=nodeconfigs,scope=Namespaced
// +kubebuilder:printcolumn:name="Scope",type=string,JSONPath=`.spec.nodeName`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.allConfigStatus`

type NodeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              NodeConfigSpec   `json:"spec"`
	Status            NodeConfigStatus `json:"status"`
}

type NodeConfigSpec struct {
	NTPConfig *NTPConfig `json:"ntpConfigs,omitempty"`
}
type NTPConfig struct {
	NTPServers     string `json:"ntpServers"`
	ConfigModified bool   `json:"configModified"`
}

type NodeConfigStatus struct {
	NTPConditions []ConfigStatus `json:"ntpConditions,omitempty"`
	// AllConfigStatus present the all configs status (Applied or WaitApplied)
	// the current state config, options are "Applied", "WaitApplied"
	// +kubebuilder:validation:Enum:=Applied;WaitApplied
	// +kubebuilder:default:="WaitApplied"
	AllConfigStatus string `json:"allConfigsStatus"`
}

type ConfigStatus struct {
	Type   ConfigConditionType    `json:"type"`
	Status corev1.ConditionStatus `json:"status"`
	// +nullable
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

type ConfigConditionType string

const (
	// first time when we create the config, is "WaitApplied"
	ConfigWaitApplied ConfigConditionType = "WaitApplied"

	// when the config is applied, is "Applied"
	ConfigApplied ConfigConditionType = "Applied"

	// when the applied config is modified, is "Modified"
	ConfigModified ConfigConditionType = "Modified"
)
