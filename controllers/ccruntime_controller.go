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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	nodeapi "k8s.io/api/node/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ccv1beta1 "github.com/confidential-containers/operator/api/v1beta1"
)

// CcRuntimeReconciler reconciles a CcRuntime object
type CcRuntimeReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Log       logr.Logger
	ccRuntime *ccv1beta1.CcRuntime
}

//+kubebuilder:rbac:groups=confidentialcontainers.org,resources=ccruntimes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=confidentialcontainers.org,resources=ccruntimes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=confidentialcontainers.org,resources=ccruntimes/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;patch
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;delete;update;patch
//+kubebuilder:rbac:groups=node.k8s.io,resources=runtimeclasses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CcRuntime object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *CcRuntimeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = log.FromContext(ctx)
	_ = r.Log.WithValues("ccruntime", req.NamespacedName)
	r.Log.Info("Reconciling CcRuntime in Kubernetes Cluster")

	// Fetch the CcRuntime instance
	r.ccRuntime = &ccv1beta1.CcRuntime{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, r.ccRuntime)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Check if the CcRuntime instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if r.ccRuntime.GetDeletionTimestamp() != nil {
		r.Log.Info("ccRuntime instance marked deleted")
		return r.processCcRuntimeDeleteRequest()
	}

	return r.processCcRuntimeInstallRequest()
}

func (r *CcRuntimeReconciler) processCcRuntimeDeleteRequest() (ctrl.Result, error) {
	// Run kata-cleanup logic
	r.Log.Info("processCcRuntimeDeleteRequest()")

	if contains(r.ccRuntime.GetFinalizers(), RuntimeConfigFinalizer) {
		// Delete install DS
		r.Log.Info("Deleting install Daemonset")
		installDs := &appsv1.DaemonSet{}
		installDsName := "cc-operator-daemon-" + string(InstallOperation)
		installDsNamespace := "confidential-containers-system"
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: installDsName, Namespace: installDsNamespace}, installDs)
		if err != nil {
			if errors.IsNotFound(err) {
				// Check for nodes with label set by install DS prestop hook.
				// If no nodes exist then remove finalizer and reconcile
				err, nodes := r.getNodesWithLabels(map[string]string{"katacontainers.io/kata-runtime": "cleanup"})
				if err != nil {
					r.Log.Error(err, "Error in getting list of nodes with label katacontainers.io/kata-runtime=cleanup")
					return ctrl.Result{}, err
				}
				if len(nodes.Items) == 0 {
					r.Log.Info("No Nodes with required labels found. Remove Finalizer")
					controllerutil.RemoveFinalizer(r.ccRuntime, RuntimeConfigFinalizer)
					// Update CR
					err = r.Client.Update(context.TODO(), r.ccRuntime)
					if err != nil {
						r.Log.Error(err, "Failed to update ccRuntime with finalizer")
						return ctrl.Result{}, err
					}
					// Requeue the request
					return ctrl.Result{Requeue: true}, nil
				}
			}
			r.Log.Error(err, "Error in getting Install Daemonset")
			return ctrl.Result{}, err
		} else {
			err = r.Client.Delete(context.TODO(), installDs)
			if err != nil {
				r.Log.Error(err, "Error in deleting Install Daemonset")
				return ctrl.Result{}, err
			} else {
				return ctrl.Result{Requeue: true, RequeueAfter: 20 * time.Second}, nil
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) processCcRuntimeInstallRequest() (ctrl.Result, error) {
	if r.ccRuntime.Status.TotalNodesCount == 0 {

		nodesList := &corev1.NodeList{}

		if r.ccRuntime.Spec.CcNodeSelector == nil {
			r.ccRuntime.Spec.CcNodeSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{"node-role.kubernetes.io/worker": ""},
			}
		}

		listOpts := []client.ListOption{
			client.MatchingLabels(r.ccRuntime.Spec.CcNodeSelector.MatchLabels),
		}

		err := r.Client.List(context.TODO(), nodesList, listOpts...)
		if err != nil {
			return ctrl.Result{}, err
		}
		r.ccRuntime.Status.TotalNodesCount = len(nodesList.Items)

		if r.ccRuntime.Status.TotalNodesCount == 0 {
			return ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Second},
				fmt.Errorf("no suitable worker nodes found for runtime installation. Please make sure to label the nodes with labels specified in CcNodeSelector")
		}

		if r.ccRuntime.Spec.Config.PayloadImage == "" {
			return ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Second},
				fmt.Errorf("PayloadImage must be specified to download the runtime binaries")
		}

		r.ccRuntime.Status.RuntimeName = r.ccRuntime.Spec.RuntimeName

		err = r.Client.Status().Update(context.TODO(), r.ccRuntime)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Don't create the daemonset if the runtime is already installed on the cluster nodes
	if r.ccRuntime.Status.TotalNodesCount > 0 &&
		r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesCount != r.ccRuntime.Status.TotalNodesCount {
		ds := r.processDaemonset(InstallOperation)
		// Set CcRuntime instance as the owner and controller
		if err := controllerutil.SetControllerReference(r.ccRuntime, ds, r.Scheme); err != nil {
			r.Log.Error(err, "Failed setting ControllerReference")
			return ctrl.Result{}, err
		}
		foundDs := &appsv1.DaemonSet{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, foundDs)
		if err != nil && errors.IsNotFound(err) {
			r.Log.Info("Creating a new installation Daemonset", "ds.Namespace", ds.Namespace, "ds.Name", ds.Name)
			err = r.Client.Create(context.TODO(), ds)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else if err != nil {
			return ctrl.Result{}, err
		}

		return r.monitorCcRuntimeInstallation()
	}

	// Create the uninstall DaemonSet as well. It'll run only when nodes gets labelled as part of preStop hook
	ds := r.processDaemonset(UninstallOperation)
	// Set CcRuntime instance as the owner and controller
	if err := controllerutil.SetControllerReference(r.ccRuntime, ds, r.Scheme); err != nil {
		r.Log.Error(err, "Failed setting ControllerReference for uninstall DS")
		return ctrl.Result{}, err
	}
	foundDs := &appsv1.DaemonSet{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, foundDs)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating cleanup Daemonset", "ds.Namespace", ds.Namespace, "ds.Name", ds.Name)
		err = r.Client.Create(context.TODO(), ds)
		if err != nil {
			r.Log.Error(err, "Error in creating Cleanup Daemonset")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Add finalizer for this CR
	if !contains(r.ccRuntime.GetFinalizers(), RuntimeConfigFinalizer) {
		if err := r.addFinalizer(); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) monitorCcRuntimeInstallation() (ctrl.Result, error) {
	// If the installation of the binaries is successful on all nodes, proceed with creating the runtime classes
	if r.ccRuntime.Status.TotalNodesCount > 0 && r.ccRuntime.Status.InstallationStatus.InProgress.InProgressNodesCount == r.ccRuntime.Status.TotalNodesCount {
		rs, err := r.setRuntimeClass()
		if err != nil {
			return rs, err
		}

		r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesList = r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList
		r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesCount = len(r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesList)
		r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList = []string{}
		r.ccRuntime.Status.InstallationStatus.InProgress.InProgressNodesCount = 0

		err = r.Client.Status().Update(context.TODO(), r.ccRuntime)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	nodesList := &corev1.NodeList{}

	if r.ccRuntime.Spec.CcNodeSelector == nil {
		r.ccRuntime.Spec.CcNodeSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{"node-role.kubernetes.io/worker": ""},
		}
	}

	listOpts := []client.ListOption{
		client.MatchingLabels(r.ccRuntime.Spec.CcNodeSelector.MatchLabels),
	}

	err := r.Client.List(context.TODO(), nodesList, listOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, node := range nodesList.Items {
		if !contains(r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList, node.Name) {
			for k, v := range node.GetLabels() {
				//kata-deploy labels with katacontainers.io/kata-runtime:"true"
				//TODO use generic label like confidentialcontainers.org/runtime:"true"
				if k == "katacontainers.io/kata-runtime" && v == "true" {
					r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList = append(r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList, node.Name)
					r.ccRuntime.Status.InstallationStatus.InProgress.InProgressNodesCount++

					err = r.Client.Status().Update(context.TODO(), r.ccRuntime)
					if err != nil {
						return ctrl.Result{}, err
					}
				}
			}
		}
		if r.ccRuntime.Status.InstallationStatus.InProgress.InProgressNodesCount != r.ccRuntime.Status.TotalNodesCount {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) setRuntimeClass() (ctrl.Result, error) {
	runtimeClassNames := []string{"kata-qemu", "kata", "kata-cc"}

	for _, runtimeClassName := range runtimeClassNames {
		rc := func() *nodeapi.RuntimeClass {
			rc := &nodeapi.RuntimeClass{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "node.k8s.io/v1",
					Kind:       "RuntimeClass",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: runtimeClassName,
				},
				Handler: runtimeClassName,
			}

			if r.ccRuntime.Spec.CcNodeSelector != nil {
				rc.Scheduling = &nodeapi.Scheduling{
					NodeSelector: r.ccRuntime.Spec.CcNodeSelector.MatchLabels,
				}
			}
			return rc
		}()

		// Set CcRuntime r.ccRuntime as the owner and controller
		if err := controllerutil.SetControllerReference(r.ccRuntime, rc, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		foundRc := &nodeapi.RuntimeClass{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: rc.Name}, foundRc)
		if err != nil && errors.IsNotFound(err) {
			r.Log.Info("Creating a new RuntimeClass", "rc.Name", rc.Name)
			err = r.Client.Create(context.TODO(), rc)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

	}

	r.ccRuntime.Status.RuntimeClass = strings.Join(runtimeClassNames, ",")
	err := r.Client.Status().Update(context.TODO(), r.ccRuntime)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) processDaemonset(operation DaemonOperation) *appsv1.DaemonSet {
	runPrivileged := true
	var runAsUser int64 = 0
	hostPt := corev1.HostPathType("DirectoryOrCreate")

	dsName := "cc-operator-daemon-" + string(operation)
	labels := map[string]string{
		"name": dsName,
	}

	r.Log.Info("processDaemonset", "operation", operation)

	var nodeSelector map[string]string
	if r.ccRuntime.Spec.CcNodeSelector != nil {
		nodeSelector = r.ccRuntime.Spec.CcNodeSelector.MatchLabels
	} else {
		nodeSelector = map[string]string{
			"node-role.kubernetes.io/worker": "",
		}
	}

	containerCommand := []string{}
	preStopHook := &corev1.Lifecycle{}

	if operation == InstallOperation {
		preStopHook = &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"bash", "-c", "/opt/kata-artifacts/scripts/kata-deploy.sh cleanup"},
				},
			},
		}
		containerCommand = []string{"bash", "-c", "/opt/kata-artifacts/scripts/kata-deploy.sh install"}
	}

	if operation == UninstallOperation {
		containerCommand = []string{"bash", "-c", "/opt/kata-artifacts/scripts/kata-deploy.sh reset"}
		nodeSelector = map[string]string{
			"katacontainers.io/kata-runtime": "cleanup",
		}
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: "confidential-containers-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: "RollingUpdate",
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 1,
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "cc-operator-controller-manager",
					NodeSelector:       nodeSelector,
					Containers: []corev1.Container{
						{
							Name:            "cc-runtime-install-pod",
							Image:           r.ccRuntime.Spec.Config.PayloadImage,
							ImagePullPolicy: "Always",
							Lifecycle:       preStopHook,
							SecurityContext: &corev1.SecurityContext{
								// TODO - do we really need to run as root?
								Privileged: &runPrivileged,
								RunAsUser:  &runAsUser,
							},
							Command: containerCommand,
							Env: []corev1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "CONFIGURE_CC",
									Value: "yes",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "crio-conf",
									MountPath: "/etc/crio/",
								},
								{
									Name:      "containerd-conf",
									MountPath: "/etc/containerd/",
								},
								{
									Name:      "kata-artifacts",
									MountPath: "/opt/kata/",
								},
								{
									Name:      "dbus",
									MountPath: "/var/run/dbus",
								},
								{
									Name:      "systemd",
									MountPath: "/run/systemd",
								},
								{
									Name:      "local-bin",
									MountPath: "/usr/local/bin/",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "crio-conf",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/crio/",
								},
							},
						},
						{
							Name: "containerd-conf",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/containerd/",
								},
							},
						},
						{
							Name: "kata-artifacts",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/opt/kata/",
									Type: &hostPt,
								},
							},
						},
						{
							Name: "dbus",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run/dbus",
								},
							},
						},
						{
							Name: "systemd",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/systemd",
								},
							},
						},
						{
							Name: "local-bin",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/usr/local/bin/",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *CcRuntimeReconciler) addFinalizer() error {
	r.Log.Info("Adding Finalizer for the RuntimeConfig")
	controllerutil.AddFinalizer(r.ccRuntime, RuntimeConfigFinalizer)

	// Update CR
	err := r.Client.Update(context.TODO(), r.ccRuntime)
	if err != nil {
		r.Log.Error(err, "Failed to update ccRuntime with finalizer")
		return err
	}
	return nil
}

// Get Nodes container specific labels
func (r *CcRuntimeReconciler) getNodesWithLabels(nodeLabels map[string]string) (error, *corev1.NodeList) {
	nodes := &corev1.NodeList{}
	labelSelector := labels.SelectorFromSet(nodeLabels)
	listOpts := []client.ListOption{
		client.MatchingLabelsSelector{Selector: labelSelector},
	}

	if err := r.Client.List(context.TODO(), nodes, listOpts...); err != nil {
		r.Log.Error(err, "Getting list of nodes having specified labels failed")
		return err, &corev1.NodeList{}
	}
	return nil, nodes
}

func (r *CcRuntimeReconciler) mapCcRuntimeToRequests(ccRuntimeObj client.Object) []reconcile.Request {
	ccRuntimeList := &ccv1beta1.CcRuntimeList{}

	err := r.Client.List(context.TODO(), ccRuntimeList)
	if err != nil {
		return []reconcile.Request{}
	}

	reconcileRequests := make([]reconcile.Request, len(ccRuntimeList.Items))
	for _, ccRuntime := range ccRuntimeList.Items {
		reconcileRequests = append(reconcileRequests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: ccRuntime.Name,
			},
		})
	}
	return reconcileRequests
}

// SetupWithManager sets up the controller with the Manager.
func (r *CcRuntimeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ccv1beta1.CcRuntime{}).
		Watches(
			&source.Kind{Type: &corev1.Node{}},
			handler.EnqueueRequestsFromMapFunc(r.mapCcRuntimeToRequests)).
		Complete(r)
}
