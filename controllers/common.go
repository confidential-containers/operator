package controllers

import (
	corev1 "k8s.io/api/core/v1"
)

// DaemonOperation represents the operation the daemon is going to perform
type DaemonOperation string

var (
	PreInstallDoneLabel    = []string{"cc-preinstall/done", "true"}
	PostUninstallDoneLabel = []string{"cc-postuninstall/done", "true"}
	KataRuntimeLabel       = []string{"katacontainers.io/kata-runtime", "cleanup"}
)

const (
	// InstallOperation denotes the installation operation
	InstallOperation DaemonOperation = "install"

	// UninstallOperation denotes the uninstallation operation
	UninstallOperation DaemonOperation = "uninstall"

	// PreInstallOperation denotes the pre-install operation
	PreInstallOperation DaemonOperation = "pre-install"

	//PostUninstallOperation denotes the post-uninstall operation
	PostUninstallOperation DaemonOperation = "post-uninstall"

	// UpgradeOperation denotes the upgrade operation
	UpgradeOperation DaemonOperation = "upgrade"

	RuntimeConfigFinalizer = "runtimeconfig.confidentialcontainers.org/finalizer"

	DefaultImagePullPolicy = corev1.PullAlways
)

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func imagePullPolicyOrDefault(policy corev1.PullPolicy) corev1.PullPolicy {
	if policy == "" {
		return DefaultImagePullPolicy
	}
	return policy
}
