# Introduction
This document provides an overview of how the payload container image is used
by the operator to deploy the confidential containers (CC) runtime on the
Kubernetes nodes.


## Key components of the payload image

- It should contain all the required artifacts like binaries, config files etc.
- It should provide an installer (eg. a bash script) with 3 options: install, uninstall, cleanup

## How the payload image is used by the operator ?

The following are the key attributes of the CRD which are used for installing the runtime
on the Kubernetes nodes.


- `installerVolumeMounts`: These are the hostpath mount points that are used by the
runtime installer script to copy the required artifacts to the Kubernetes node.
- `installCmd`: The command provided by the payload image to install the runtime
- `uninstallCmd`: The command provided by the payload image to uninstall the runtime
- `cleanupCmd`: The command provided by the payload image to perform any cleanups post uninstall.


Following is an example for Kata runtime depicting the key attributes.

```
    installerVolumeMounts:
      - mountPath: /etc/containerd/
        name: containerd-conf
      - mountPath: /opt/confidential-containers/
        name: kata-artifacts
      - mountPath: /var/run/dbus/system_bus_socket
        name: dbus
      - mountPath: /run/systemd/system
        name: systemd
      - mountPath: /usr/local/bin/
        name: local-bin
    installerVolumes:
      - hostPath:
          path: /etc/containerd/
          type: ""
        name: containerd-conf
      - hostPath:
          path: /opt/confidential-containers/
          type: DirectoryOrCreate
        name: kata-artifacts
      - hostPath:
          path: /var/run/dbus/system_bus_socket
          type: ""
        name: dbus
      - hostPath:
          path: /run/systemd/system
          type: ""
        name: systemd
      - hostPath:
          path: /usr/local/bin/
          type: ""
        name: local-bin
    installCmd: ["/opt/kata-artifacts/scripts/kata-deploy.sh", "install"]
    uninstallCmd: ["/opt/kata-artifacts/scripts/kata-deploy.sh", "cleanup"]
    cleanupCmd: ["/opt/kata-artifacts/scripts/kata-deploy.sh", "reset"]
```

The installer (`/opt/kata-artifacts/scripts/kata-deploy.sh`) copies the required kata artifacts to
the `/opt/confidential-containers` directory on the Kubernetes node.

The following code snippet in the `kata-deploy.sh` script copies the artifacts from
the `/opt/kata-artifacts/opt/confidential-containers/` directory on the payload image
to the `/opt/confidential-containers/` directory on the Kubernetes node.


```
function install_artifacts() {
        echo "copying kata artifacts onto host"
        cp -a /opt/kata-artifacts/opt/confidential-containers/* /opt/confidential-containers/
        chmod +x /opt/confidential-containers/bin/*
}
```

Additionally the following is the directory layout for the Kata runtime payload image.


```
/opt
`-- kata-artifacts
    |-- opt
    |   `-- confidential-containers
    |       |-- bin
    |       |-- libexec
    |       `-- share
    |           |-- bash-completion
    |           |   `-- completions
    |           |-- defaults
    |           |   `-- kata-containers
    |           |-- kata-containers
    |           `-- kata-qemu
    |               `-- qemu
    |                   `-- firmware
    `-- scripts
```

The top level directory is `/opt`. All the contents under the `/opt/kata-artifacts/opt/confidential-containers`
directory in the payload image is copied to the `/opt/confidential-containers` directory on the Kubernetes node by
the `kata-deploy.sh` script.
