module github.com/confidential-containers-operator

go 1.16

replace github.com/confidential-containers/confidential-containers-operator => ./

require (
	github.com/confidential-containers/confidential-containers-operator v0.0.0-20210922081251-3642b26fcb5b
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/controller-runtime v0.10.1
)
