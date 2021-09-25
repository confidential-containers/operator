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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:Enum=kata
type CCRuntimeName string

const (
	// Kata
	KataCCRuntime CCRuntimeName = "kata"

	// Other CC Runtime
)

// ConfidentialContainersRuntimeSpec defines the desired state of ConfidentialContainersRuntime
type ConfidentialContainersRuntimeSpec struct {
	// KataConfigPoolSelector is used to filer the worker nodes
	// if not specified, all worker nodes are selected
	// +optional
	// +nullable
	ConfidentialContainersNodeSelector *metav1.LabelSelector `json:"condidentialContainersNodeSelector"`

	RuntimeName CCRuntimeName `json:"runtimeName"`

	Config ConfidentialContainersInstallConfig `json:"config"`
}

// +kubebuilder:validation:Enum=bundle;osnative
type CCInstallType string

const (
	// Use container image with all installation artifacts
	BundleInstallType CCInstallType = "bundle"

	// Use native OS packages (rpm/deb)
	OsNativeInstallType CCInstallType = "osnative"
)

// ConfidentialContainersRuntimeStatus defines the observed state of ConfidentialContainersRuntime
type ConfidentialContainersRuntimeStatus struct {
	// RuntimeClass is the name of the runtime class as used in container runtime configuration
	RuntimeClass string `json:"runtimeClass"`

	// ConfidentialContainers Runtime Name
	RuntimeName CCRuntimeName `json:"runtimeName"`

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

	// This indicates whether to use native OS packaging (rpm/deb) or Container image
	// Default is bundle (container image)
	InstallType CCInstallType `json:"installType"`

	// This specifies the location of the container image with all artifacts (CC runtime binaries, initrd, kernel, config etc)
	// when using "bundle" installType
	PayloadImage string `json:"payloadImage"`

	// This specifies the registry secret to pull of the container images
	// +optional
	ImagePullSecret *corev1.LocalObjectReference `json:"ImagePullSecret,omitempty"`

	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// This specifies the repo location to be used when using rpm/deb packages
	// Some examples
	//   add-apt-repository 'deb [arch=amd64] https://repo.confidential-containers.org/apt/ubuntu’
	//   add-apt-repository ppa:confidential-containers/cc-bundle
	//   dnf install -y https://repo.confidential-containers.org/yum/centos/cc-bundle-repo.rpm
	// +optional
	OsNativeRepo string `json:"osNativeRepo,omitempty"`

	// This specifies the location of the container image containing the CC runtime binaries
	// If both payloadImage and runtimeImage are specified, then runtimeImage content will override the equivalent one in payloadImage
	// +optional
	RuntimeImage string `json:"runtimeImage,omitempty"`

	// This specifies the location of the container image containing the guest kernel
	// If both bundleImage and guestKernelImage are specified, then guestKernelImage content will override the equivalent one in payloadImage
	// +optional
	GuestKernelImage string `json:"guestKernelImage,omitempty"`

	// This specifies the location of the container image containing the guest initrd
	// If both bundleImage and guestInitrdImage are specified, then guestInitrdImage content will override the equivalent one in payloadImage
	// +optional
	GuestInitrdImage string `json:"guestInitrdImage,omitempty"`
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
