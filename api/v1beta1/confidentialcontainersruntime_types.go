/*
Copyright 2021 CNCF.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ConfidentialContainersRuntimeSpec defines the desired state of ConfidentialContainersRuntime
type ConfidentialContainersRuntimeSpec struct {
	// KataConfigPoolSelector is used to filer the worker nodes
	// if not specified, all worker nodes are selected
	// +optional
	// +nullable
	ConfidentialContainersNodeSelector *metav1.LabelSelector `json:"condidentialContainersNodeSelector"`

	// +optional
	Config ConfidentialContainersInstallConfig `json:"config"`
}

// ConfidentialContainersRuntimeStatus defines the observed state of ConfidentialContainersRuntime
type ConfidentialContainersRuntimeStatus struct {
	// RuntimeClass is the name of the runtime class as used in container runtime configuration
	RuntimeClass string `json:"runtimeClass"`

	// ConfidentialContainersRuntimeImage is the image used for delivering kata binaries
	ConfidentialContainersRuntimeImage string `json:"confidentialContainersRuntimeImage"`

	// TotalNodesCounts is the total number of worker nodes targeted by this CR
	TotalNodesCount int `json:"totalNodesCount"`

	// InstallationStatus reflects the status of the ongoing runtime installation
	// +optional
	InstallationStatus ConfidentialContainersInstallationStatus `json:"installationStatus,omitempty"`

	// UnInstallationStatus reflects the status of the ongoing runtime uninstallation
	// +optional
	UnInstallationStatus ConfidentialContainersUnInstallationStatus `json:"unInstallationStatus,omitempty"`

	// Upgradestatus reflects the status of the ongoing runtime upgrade
	// +optional
	Upgradestatus ConfidentialContainersUpgradeStatus `json:"upgradeStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ConfidentialContainersRuntime is the Schema for the confidentialcontainersruntimes API
type ConfidentialContainersRuntime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfidentialContainersRuntimeSpec   `json:"spec,omitempty"`
	Status ConfidentialContainersRuntimeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConfidentialContainersRuntimeList contains a list of ConfidentialContainersRuntime
type ConfidentialContainersRuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfidentialContainersRuntime `json:"items"`
}

// ConfidentialContainersInstallConfig is a placeholder struct
type ConfidentialContainersInstallConfig struct {
	// SourceImage is the name of the kata-deploy image
	SourceImage string `json:"sourceImage"`
}

// ConfidentialContainersInstallationStatus reflects the status of the ongoing kata installation
type ConfidentialContainersInstallationStatus struct {
	// InProgress reflects the status of nodes that are in the process of kata installation
	InProgress ConfidentialContainersInstallationInProgressStatus `json:"inProgress,omitempty"`

	// Completed reflects the status of nodes that have completed kata installation
	Completed ConfidentialContainerCompletedStatus `json:"completed,omitempty"`

	// Failed reflects the status of nodes that have failed kata installation
	Failed ConfidentialContainersFailedNodeStatus `json:"failed,omitempty"`
}

// ConfidentialContainersInstallationInProgressStatus reflects the status of nodes that are in the process of kata installation
type ConfidentialContainersInstallationInProgressStatus struct {
	// InProgressNodesCount reflects the number of nodes that are in the process of kata installation
	InProgressNodesCount int `json:"inProgressNodesCount,omitempty"`
	// +optional
	BinariesInstalledNodesList []string `json:"binariesInstallNodesList,omitempty"`
}

// ConfidentialContainerCompletedStatus reflects the status of nodes that have completed kata operation
type ConfidentialContainerCompletedStatus struct {
	// CompletedNodesCount reflects the number of nodes that have completed kata operation
	CompletedNodesCount int `json:"completedNodesCount,omitempty"`

	// CompletedNodesList reflects the list of nodes that have completed kata operation
	// +optional
	CompletedNodesList []string `json:"completedNodesList,omitempty"`
}

// ConfidentialContainersFailedNodeStatus reflects the status of nodes that have failed kata operation
type ConfidentialContainersFailedNodeStatus struct {
	// FailedNodesCount reflects the number of nodes that have failed kata operation
	FailedNodesCount int `json:"failedNodesCount,omitempty"`

	// FailedNodesList reflects the list of nodes that have failed kata operation
	// +optional
	FailedNodesList []FailedNodeStatus `json:"failedNodesList,omitempty"`
}

// ConfidentialContainersUnInstallationStatus reflects the status of the ongoing kata uninstallation
type ConfidentialContainersUnInstallationStatus struct {
	// InProgress reflects the status of nodes that are in the process of kata uninstallation
	InProgress ConfidentialContainersUnInstallationInProgressStatus `json:"inProgress,omitempty"`

	// Completed reflects the status of nodes that have completed kata uninstallation
	Completed ConfidentialContainerCompletedStatus `json:"completed,omitempty"`

	// Failed reflects the status of nodes that have failed kata uninstallation
	Failed ConfidentialContainersFailedNodeStatus `json:"failed,omitempty"`
}

// ConfidentialContainersUnInstallationInProgressStatus reflects the status of nodes that are in the process of kata installation
type ConfidentialContainersUnInstallationInProgressStatus struct {
	InProgressNodesCount int `json:"inProgressNodesCount,omitempty"`
	// +optional
	BinariesUnInstalledNodesList []string `json:"binariesUninstallNodesList,omitempty"`
}

// ConfidentialContainersUpgradeStatus reflects the status of the ongoing kata upgrade
type ConfidentialContainersUpgradeStatus struct {
}

// FailedNodeStatus holds the name and the error message of the failed node
type FailedNodeStatus struct {
	// Name of the failed node
	Name string `json:"name"`
	// Error message of the failed node reported by the installation daemon
	Error string `json:"error"`
}

func init() {
	SchemeBuilder.Register(&ConfidentialContainersRuntime{}, &ConfidentialContainersRuntimeList{})
}
