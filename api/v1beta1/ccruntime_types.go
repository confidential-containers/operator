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

// +kubebuilder:validation:Enum=kata;enclave-cc
type CcRuntimeName string

// CcRuntimeSpec defines the desired state of CcRuntime
type CcRuntimeSpec struct {
	// CcNodeSelector is used to select the worker nodes to deploy the runtime
	// if not specified, all worker nodes are selected
	// +optional
	// +nullable
	CcNodeSelector *metav1.LabelSelector `json:"ccNodeSelector"`

	RuntimeName CcRuntimeName `json:"runtimeName"`

	Config CcInstallConfig `json:"config"`
}

// +kubebuilder:validation:Enum=bundle;osnative
type CcInstallType string

const (
	// Use container image with all installation artifacts
	BundleInstallType CcInstallType = "bundle"

	// Use native OS packages (rpm/deb)
	OsNativeInstallType CcInstallType = "osnative"
)

// CcRuntimeStatus defines the observed state of CcRuntime
type CcRuntimeStatus struct {
	// RuntimeClass is the name of the runtime class as used in container runtime configuration
	RuntimeClass string `json:"runtimeClass"`

	// Cc Runtime Name
	RuntimeName CcRuntimeName `json:"runtimeName"`

	// TotalNodesCounts is the total number of worker nodes targeted by this CR
	TotalNodesCount int `json:"totalNodesCount"`

	// InstallationStatus reflects the status of the ongoing runtime installation
	// +optional
	InstallationStatus CcInstallationStatus `json:"installationStatus,omitempty"`

	// UnInstallationStatus reflects the status of the ongoing runtime uninstallation
	// +optional
	UnInstallationStatus CcUnInstallationStatus `json:"unInstallationStatus,omitempty"`

	// Upgradestatus reflects the status of the ongoing runtime upgrade
	// +optional
	Upgradestatus CcUpgradeStatus `json:"upgradeStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=ccruntimes,shortName=ccr,scope=Cluster

// CcRuntime is the Schema for the ccruntimes API
type CcRuntime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CcRuntimeSpec   `json:"spec,omitempty"`
	Status CcRuntimeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CcRuntimeList contains a list of CcRuntime
type CcRuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CcRuntime `json:"items"`
}

// CcInstallConfig is a placeholder struct
type CcInstallConfig struct {

	// This indicates whether to use native OS packaging (rpm/deb) or Container image
	// Default is bundle (container image)
	InstallType CcInstallType `json:"installType"`

	// This specifies the location of the container image with all artifacts (Cc runtime binaries, initrd, kernel, config etc)
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

	// This specifies the location of the container image containing the Cc runtime binaries
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

	// This specifies volume mounts required for the installer pods
	// +optional
	InstallerVolumeMounts []corev1.VolumeMount `json:"installerVolumeMounts,omitempty"`

	// This specifies volumes required for the installer pods
	// +optional
	InstallerVolumes []corev1.Volume `json:"installerVolumes,omitempty"`

	// This specifies the command for installation of the runtime on the nodes
	// +optional
	InstallCmd []string `json:"installCmd,omitempty"`

	// This specifies the command for uninstallation of the runtime on the nodes
	// +optional
	UninstallCmd []string `json:"uninstallCmd,omitempty"`

	// This specifies the command for cleanup on the nodes
	// +optional
	CleanupCmd []string `json:"cleanupCmd,omitempty"`

	// This specifies the RuntimeClasses that needs to be created
	// +optional
	RuntimeClassNames []string `json:"runtimeClassNames,omitempty"`

	// This specifies the environment variables required by the daemon set
	// +optional
	EnvironmentVariables []corev1.EnvVar `json:"environmentVariables,omitempty"`
	// This specifies the label that the install daemonset adds to nodes
	// when the installation is done
	InstallDoneLabel map[string]string `json:"installDoneLabel,omitempty"`

	// This specifies the label that the uninstall daemonset adds to nodes
	// when the uninstallation  is done
	UninstallDoneLabel map[string]string `json:"uninstallDoneLabel,omitempty"`

	// This specifies the configuration for the pre-install daemonset
	// +optional
	PreInstall PreInstallConfig `json:"preInstall,omitempty"`

	// This specifies the configuration for the post-uninstall daemonset
	// +optional
	PostUninstall PostUninstallConfig `json:"postUninstall,omitempty"`
}

type PostUninstallConfig struct {
	// This specifies the command executes before UnInstallCmd
	// +optional
	Cmd []string `json:"cmd,omitempty"`

	// This specifies the pull spec for the postuninstall daemonset image
	// +optional
	Image string `json:"image,omitempty"`

	// This specifies the env variables for the post-uninstall daemon set
	// +optional
	EnvironmentVariables []corev1.EnvVar `json:"environmentVariables,omitempty"`

	// This specifies the volumes for the post-uninstall daemon set
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// This specifies the volumeMounts for the post-uninstall daemon set
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

type PreInstallConfig struct {
	// This specifies the command executes before InstallCmd
	// +optional
	Cmd []string `json:"cmd,omitempty"`

	// This specifies the image for the pre-install scripts
	// +optional
	Image string `json:"image,omitempty"`

	// This specifies the env variables for the pre-install daemon set
	// +optional
	EnvironmentVariables []corev1.EnvVar `json:"environmentVariables,omitempty"`

	// This specifies the volumes for the pre-install daemon set
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`
	// This specifies the volumeMounts for the pre-install daemon set
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

// CcInstallationStatus reflects the status of the ongoing confidential containers runtime installation
type CcInstallationStatus struct {
	// InProgress reflects the status of nodes that are in the process of installation
	InProgress CcInstallationInProgressStatus `json:"inProgress,omitempty"`

	// Completed reflects the status of nodes that have completed the installation
	Completed CcCompletedStatus `json:"completed,omitempty"`

	// Failed reflects the status of nodes that have failed installation
	Failed CcFailedNodeStatus `json:"failed,omitempty"`
}

// CcInstallationInProgressStatus reflects the status of nodes that are in the process of installing
// the confidential containers runtime
type CcInstallationInProgressStatus struct {
	// InProgressNodesCount reflects the number of nodes that are in the process of installation
	InProgressNodesCount int `json:"inProgressNodesCount,omitempty"`
	// +optional
	BinariesInstalledNodesList []string `json:"binariesInstallNodesList,omitempty"`
}

// CcCompletedStatus reflects the status of nodes that have completed the installation of
// the confidential containers runtime
type CcCompletedStatus struct {
	// CompletedNodesCount reflects the number of nodes that have completed install operation
	CompletedNodesCount int `json:"completedNodesCount,omitempty"`

	// CompletedNodesList reflects the list of nodes that have completed install operation
	// +optional
	CompletedNodesList []string `json:"completedNodesList,omitempty"`
}

// CcFailedNodeStatus reflects the status of nodes that have failed installation of
// the confidential containers runtime
type CcFailedNodeStatus struct {
	// FailedNodesCount reflects the number of nodes that have failed installation
	FailedNodesCount int `json:"failedNodesCount,omitempty"`

	// FailedNodesList reflects the list of nodes that have failed installation
	// +optional
	FailedNodesList []FailedNodeStatus `json:"failedNodesList,omitempty"`
}

// CcUnInstallationStatus reflects the status of the ongoing uninstallation of
// the confidential containers runtime
type CcUnInstallationStatus struct {
	// InProgress reflects the status of nodes that are in the process of uninstallation
	InProgress CcUnInstallationInProgressStatus `json:"inProgress,omitempty"`

	// Completed reflects the status of nodes that have completed the uninstallation operation
	Completed CcCompletedStatus `json:"completed,omitempty"`

	// Failed reflects the status of nodes that have failed uninstallation
	Failed CcFailedNodeStatus `json:"failed,omitempty"`
}

// CcUnInstallationInProgressStatus reflects the status of nodes that are in the process of uninstalling
// the confidential containers runtime
type CcUnInstallationInProgressStatus struct {
	// InProgressNodesCount reflects the number of nodes that are in the process of uninstallation
	InProgressNodesCount int `json:"inProgressNodesCount,omitempty"`
	// +optional
	BinariesUnInstalledNodesList []string `json:"binariesUninstallNodesList,omitempty"`
}

// CcUpgradeStatus reflects the status of the ongoing upgrade of
// the confidential containers runtime
type CcUpgradeStatus struct {
}

// FailedNodeStatus holds the name and the error message of the failed node
type FailedNodeStatus struct {
	// Name of the failed node
	Name string `json:"name"`
	// Error message of the failed node reported by the installation daemon
	Error string `json:"error"`
}

func init() {
	SchemeBuilder.Register(&CcRuntime{}, &CcRuntimeList{})
}
