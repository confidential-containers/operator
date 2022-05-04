package controllers

// DaemonOperation represents the operation the daemon is going to perform
type DaemonOperation string

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
)

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
