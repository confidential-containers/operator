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
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	nodeapi "k8s.io/api/node/v1"
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
	Namespace string
}

//+kubebuilder:rbac:groups=confidentialcontainers.org,resources=ccruntimes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=confidentialcontainers.org,resources=ccruntimes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=confidentialcontainers.org,resources=ccruntimes/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;patch;update
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;delete;update;patch
//+kubebuilder:rbac:groups=node.k8s.io,resources=runtimeclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;update

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
	err := r.Get(context.TODO(), req.NamespacedName, r.ccRuntime)
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
	// Create the uninstall DaemonSet
	ds := r.processDaemonset(UninstallOperation)
	if err := controllerutil.SetControllerReference(r.ccRuntime, ds, r.Scheme); err != nil {
		r.Log.Error(err, "Failed setting ControllerReference for uninstall DS")
		return ctrl.Result{}, err
	}
	foundDs := &appsv1.DaemonSet{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, foundDs)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating cleanup Daemonset", "ds.Namespace", ds.Namespace, "ds.Name", ds.Name)
		err = r.Create(context.TODO(), ds)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check if the CcRuntime instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if r.ccRuntime.GetDeletionTimestamp() != nil {
		r.Log.Info("ccRuntime instance marked deleted")
		res, err := r.processCcRuntimeDeleteRequest()
		if err != nil || res.Requeue {
			return res, err
		} else {
			return ctrl.Result{}, nil
		}

	}

	return r.processCcRuntimeInstallRequest()
}

func (r *CcRuntimeReconciler) getNodeClient() (v1.NodeInterface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	nodeClient := clientset.CoreV1().Nodes()
	return nodeClient, err
}

// This function sets the StartUninstallLabel on all nodes that completed
// the ccruntime install (have InstallDoneLabel set), which is used by
// the uninstall DS
func (r *CcRuntimeReconciler) setCleanupNodeLabels() (ctrl.Result, error) {
	var nodesList = &corev1.NodeList{}
	nodesClient, err := r.getNodeClient()
	if err != nil {
		r.Log.Info("Couldn't get nodes client")
		return ctrl.Result{}, err
	}

	listOpts := []client.ListOption{
		client.MatchingLabels(r.ccRuntime.Spec.Config.InstallDoneLabel),
	}

	err = r.List(context.TODO(), nodesList, listOpts...)
	if err != nil {
		r.Log.Info("failed to list nodes during uninstallation status update")
		return ctrl.Result{}, err
	}

	installDoneLabel := make([]string, 0, len(r.ccRuntime.Spec.Config.InstallDoneLabel))
	for key := range r.ccRuntime.Spec.Config.InstallDoneLabel {
		installDoneLabel = append(installDoneLabel, key)
	}
	if len(installDoneLabel) > 1 {
		return ctrl.Result{}, fmt.Errorf("installDoneLabel must only have one entry")
	}

	for _, node := range nodesList.Items {
		labels := node.GetLabels()
		if val, exists := labels[installDoneLabel[0]]; exists && val == "true" {
			labels[StartUninstallLabel[0]] = StartUninstallLabel[1]
			_, err := nodesClient.Update(context.TODO(), &node, metav1.UpdateOptions{
				TypeMeta:     metav1.TypeMeta{},
				DryRun:       nil,
				FieldManager: "",
			})
			if err != nil {
				r.Log.Info("failed to update node labels")
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) processCcRuntimeDeleteRequest() (ctrl.Result, error) {
	// Create the uninstall DaemonSet
	ds := r.processDaemonset(UninstallOperation)
	if err := controllerutil.SetControllerReference(r.ccRuntime, ds, r.Scheme); err != nil {
		r.Log.Error(err, "Failed setting ControllerReference for uninstall DS")
		return ctrl.Result{}, err
	}
	foundDs := &appsv1.DaemonSet{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, foundDs)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating cleanup Daemonset", "ds.Namespace", ds.Namespace, "ds.Name", ds.Name)
		err = r.Create(context.TODO(), ds)
	}

	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
	}

	if !contains(r.ccRuntime.GetFinalizers(), RuntimeConfigFinalizer) {
		return ctrl.Result{}, err
	}

	return handleFinalizers(r)
}

func handleFinalizers(r *CcRuntimeReconciler) (ctrl.Result, error) {
	var result = ctrl.Result{}

	// Check for nodes with label set by install DS prestop hook.
	// If no nodes exist then remove finalizer and reconcile
	nodes, err := r.getNodesWithLabels(r.ccRuntime.Spec.Config.UninstallDoneLabel)
	if err != nil {
		r.Log.Error(err, "Error in getting list of nodes with uninstallDoneLabel")
		return ctrl.Result{}, err
	}

	finishedNodes := len(nodes.Items)

	if allNodesDone(finishedNodes, r) {
		if r.ccRuntime.Spec.Config.PostUninstall.Image == "" {
			controllerutil.RemoveFinalizer(r.ccRuntime, RuntimeConfigFinalizer)

			if err != nil {
				return result, err
			}

			return r.updateCcRuntime()
		}

		result, err = handlePostUninstall(r)
		if !result.Requeue {
			controllerutil.RemoveFinalizer(r.ccRuntime, RuntimeConfigFinalizer)
			result, err = r.updateCcRuntime()
			if err != nil {
				return result, err
			}
			result, err = r.deleteUninstallDaemonsets()
			prepostLabels := map[string]string{}
			if r.ccRuntime.Spec.Config.PreInstall.Image != "" {
				prepostLabels[PreInstallDoneLabel[0]] = PreInstallDoneLabel[1]
			}
			if r.ccRuntime.Spec.Config.PostUninstall.Image != "" {
				prepostLabels[PostUninstallDoneLabel[0]] = PostUninstallDoneLabel[1]

			}
			nodes, err := r.getNodesWithLabels(prepostLabels)
			if err != nil {
				r.Log.Error(err, "an error occured when getting the list of nodes from which we want to"+
					"remove preinstall/postuninstall labels")
				return ctrl.Result{}, err
			}

			postuninstalledNodes := len(nodes.Items)
			if !allNodesDone(postuninstalledNodes, r) {
				return ctrl.Result{Requeue: true}, nil
			}

			if postuninstalledNodes > 0 {
				result, err = r.removeNodeLabels(nodes)
				if err != nil {
					r.Log.Error(err, "removing the labels from nodes failed")
					return ctrl.Result{}, err
				}
			}
		}
		if err != nil {
			return result, err
		}
		return r.updateCcRuntime()
	}

	result, err = r.setCleanupNodeLabels()
	if err != nil {
		r.Log.Error(err, "updating the cleanup labels on nodes failed")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, err
	}

	result, err = r.updateUninstallationStatus(finishedNodes)
	if err != nil {
		r.Log.Info("Error from updateUninstallationStatus")
		return result, err
	}
	result.Requeue = true
	result.RequeueAfter = time.Second * 10
	return result, err
}

func (r *CcRuntimeReconciler) updateCcRuntime() (ctrl.Result, error) {
	err := r.Update(context.TODO(), r.ccRuntime)
	if err != nil {
		r.Log.Error(err, "failed to update ccRuntime")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
	}
	return ctrl.Result{}, nil
}

func allNodesDone(finishedNodes int, r *CcRuntimeReconciler) bool {
	return finishedNodes == r.ccRuntime.Status.TotalNodesCount
}

func (r *CcRuntimeReconciler) updateUninstallationStatus(finishedNodes int) (ctrl.Result, error) {
	var doneNodes []string

	cleanupNodes, err := r.getNodesWithLabels(r.ccRuntime.Spec.Config.UninstallDoneLabel)
	if err != nil {
		r.Log.Error(err, "Error in getting list of nodes with UninstallDoneLabel")
		return ctrl.Result{}, err
	}

	// Update CR
	r.ccRuntime.Status.UnInstallationStatus.Completed.CompletedNodesCount = finishedNodes
	r.ccRuntime.Status.UnInstallationStatus.InProgress.InProgressNodesCount = r.ccRuntime.Status.TotalNodesCount - finishedNodes
	for i := range cleanupNodes.Items {
		doneNodes = append(doneNodes, cleanupNodes.Items[i].Name)
	}
	r.ccRuntime.Status.UnInstallationStatus.InProgress.BinariesUnInstalledNodesList = doneNodes
	err = r.Update(context.TODO(), r.ccRuntime)
	if err != nil {
		r.Log.Error(err, "failed to update ccRuntime with finalizer")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, err
	}
	return ctrl.Result{}, nil
}

func handlePostUninstall(r *CcRuntimeReconciler) (ctrl.Result, error) {
	postUninstallDoneLabel := map[string]string{PostUninstallDoneLabel[0]: PostUninstallDoneLabel[1]}
	nodes, err := r.getNodesWithLabels(postUninstallDoneLabel)
	if err != nil {
		r.Log.Info("couldn't get nodes labeled with postuninstall done label")
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
	}

	if r.ccRuntime.Spec.Config.PostUninstall.Image != "" &&
		len(nodes.Items) < r.ccRuntime.Status.TotalNodesCount &&
		r.ccRuntime.Status.TotalNodesCount > 0 {
		postUninstallDs := r.makeHookDaemonset(PostUninstallOperation)
		// get daemonset
		res, err := r.handlePrePostDs(postUninstallDs, postUninstallDoneLabel)
		if res.Requeue {
			if err != nil {
				r.Log.Info("error from handlePrePostDs")
			}
		}
		return res, err
	} else if len(nodes.Items) == r.ccRuntime.Status.TotalNodesCount {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
}

func (r *CcRuntimeReconciler) processCcRuntimeInstallRequest() (ctrl.Result, error) {
	nodesList := &corev1.NodeList{}
	r.Log.Info("processCcRuntimeInstallRequest")

	if r.ccRuntime.Spec.CcNodeSelector == nil {
		r.ccRuntime.Spec.CcNodeSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{"node.kubernetes.io/worker": ""},
		}
	}

	listOpts := []client.ListOption{
		client.MatchingLabels(r.ccRuntime.Spec.CcNodeSelector.MatchLabels),
	}

	err := r.List(context.TODO(), nodesList, listOpts...)
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

	preInstallDoneLabel := map[string]string{PreInstallDoneLabel[0]: PreInstallDoneLabel[1]}

	// if ds exists, get all labels
	nodes, err := r.getNodesWithLabels(preInstallDoneLabel)
	if err != nil {
		r.Log.Info("couldn't GET labelled nodes")
		return ctrl.Result{}, err
	}
	if r.ccRuntime.Spec.Config.PreInstall.Image != "" &&
		len(nodes.Items) < r.ccRuntime.Status.TotalNodesCount {
		preInstallDs := r.makeHookDaemonset(PreInstallOperation)
		r.Log.Info("ds = ", "daemonset", preInstallDs)
		res, err := r.handlePrePostDs(preInstallDs, preInstallDoneLabel)
		if res.Requeue {
			r.Log.Info("requeue request from handlePrePostDs")
			return res, err
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
		err := r.Get(context.TODO(), types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, foundDs)
		if err != nil && errors.IsNotFound(err) {
			r.Log.Info("Creating a new installation Daemonset", "ds.Namespace", ds.Namespace, "ds.Name", ds.Name)
			err = r.Create(context.TODO(), ds)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else if err != nil {
			return ctrl.Result{}, err
		}

	}
	return r.monitorCcRuntimeInstallation()

}

/*
This creates DaemonSets for pre-install/post-uninstall unless it already exists.
We leave the DaemonSets running until the ccRuntime finalizer is called.
This way the running DaemonSet automatically applies changes when a new
node is added.
*/
func (r *CcRuntimeReconciler) handlePrePostDs(preInstallDs *appsv1.DaemonSet, doneLabel map[string]string) (
	ctrl.Result, error,
) {
	foundPreinstallDs := &appsv1.DaemonSet{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: preInstallDs.Name, Namespace: preInstallDs.Namespace}, foundPreinstallDs)
	r.Log.Info("create preinstall/postuninstall DS", "DS", preInstallDs)
	if err != nil && errors.IsNotFound(err) {
		err = r.Create(context.TODO(), preInstallDs)
		if err != nil {
			r.Log.Info("failed to create preinstall/postuninstall DS", "DS", preInstallDs)
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
		}
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
	} else if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
	}
	// if ds exists, get all labels
	nodes, err := r.getNodesWithLabels(doneLabel)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
	}
	if len(nodes.Items) < r.ccRuntime.Status.TotalNodesCount {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
	}

	return ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) monitorCcRuntimeInstallation() (ctrl.Result, error) {
	var (
		err    error
		result ctrl.Result
	)

	// If the installation of the binaries is successful on all nodes, proceed with creating the runtime classes
	if r.allNodesInstalled() {
		// Update runtimeClass field
		var runtimeClassNames []string
		runtimeClasses := r.ccRuntime.Spec.Config.RuntimeClasses
		for _, runtimeClass := range runtimeClasses {
			foundRc := &nodeapi.RuntimeClass{}
			err := r.Get(context.TODO(), types.NamespacedName{Name: runtimeClass.Name}, foundRc)
			if errors.IsNotFound(err) {
				r.Log.Info("The runtime payload failed to create the runtime class", "runtimeClassName", runtimeClass.Name)
				return ctrl.Result{}, err
			}
			runtimeClassNames = append(runtimeClassNames, runtimeClass.Name)
		}
		r.ccRuntime.Status.RuntimeClass = strings.Join(runtimeClassNames, ",")

		// Add finalizer for this CR
		if !contains(r.ccRuntime.GetFinalizers(), RuntimeConfigFinalizer) {
			if err := r.addFinalizer(); err != nil {
				return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
			}
		}
		r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesCount = len(r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesList)
		r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList = []string{}
		r.ccRuntime.Status.InstallationStatus.InProgress.InProgressNodesCount = 0
	}

	err = r.Client.Status().Update(context.TODO(), r.ccRuntime)
	if err != nil {
		r.Log.Info("failed to update status while monitoring installation")
		return ctrl.Result{}, err
	}

	nodesList, result, err := r.getAllNodes()
	if err != nil {
		return result, err
	}

	result, err = r.updateInstallationStatus(nodesList)
	if err != nil {
		return result, err
	}

	if r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesCount != r.ccRuntime.Status.TotalNodesCount {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) updateInstallationStatus(nodesList *corev1.NodeList) (ctrl.Result, error) {
	r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList = []string{}
	for _, node := range nodesList.Items {
		r.ccRuntime.Status.InstallationStatus.InProgress.InProgressNodesCount = len(r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList)
		for k, v := range node.GetLabels() {
			doneLabel, exists := r.ccRuntime.Spec.Config.InstallDoneLabel[k]
			if exists && v == doneLabel {
				if !contains(r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesList, node.Name) {
					r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesCount++
					r.Log.Info("adding new node to completed list", "nodeName", node.Name)
					r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesList =
						append(r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesList, node.Name)
					r.ccRuntime.Status.InstallationStatus.InProgress.InProgressNodesCount = len(r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList)
					if !contains(r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList, node.Name) {
						r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList =
							append(r.ccRuntime.Status.InstallationStatus.InProgress.BinariesInstalledNodesList, node.Name)
					}
				}
			}
		}
		err := r.Client.Status().Update(context.TODO(), r.ccRuntime)
		if err != nil {
			r.Log.Info("Updating status of completed nodes etc failed")
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, err
		}

	}
	return ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) getAllNodes() (*corev1.NodeList, ctrl.Result, error) {
	nodesList := &corev1.NodeList{}

	if r.ccRuntime.Spec.CcNodeSelector == nil {
		r.ccRuntime.Spec.CcNodeSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{"node.kubernetes.io/worker": ""},
		}
	}

	listOpts := []client.ListOption{
		client.MatchingLabels(r.ccRuntime.Spec.CcNodeSelector.MatchLabels),
	}

	err := r.List(context.TODO(), nodesList, listOpts...)
	if err != nil {
		r.Log.Info("listing the nodes failed while monitoring the installation")
		return nil, ctrl.Result{}, err
	}
	return nodesList, ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) allNodesInstalled() bool {
	return r.ccRuntime.Status.TotalNodesCount > 0 &&
		r.ccRuntime.Status.InstallationStatus.Completed.CompletedNodesCount == r.ccRuntime.Status.TotalNodesCount
}

func (r *CcRuntimeReconciler) processDaemonset(operation DaemonOperation) *appsv1.DaemonSet {
	runPrivileged := true
	var runAsUser int64 = 0

	dsName := "cc-operator-daemon-" + string(operation)
	dsLabelSelectors := map[string]string{
		"name": dsName,
	}

	var nodeSelector map[string]string
	if r.ccRuntime.Spec.CcNodeSelector != nil && operation == InstallOperation {
		nodeSelector = r.ccRuntime.Spec.CcNodeSelector.MatchLabels
	} else if operation == UninstallOperation {
		nodeSelector = map[string]string{StartUninstallLabel[0]: StartUninstallLabel[1]}
	} else {
		nodeSelector = map[string]string{
			"node.kubernetes.io/worker": "",
		}
	}

	var containerCommand []string
	preStopHook := &corev1.Lifecycle{}

	if operation == InstallOperation {
		preStopHook = &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: r.ccRuntime.Spec.Config.CleanupCmd,
				},
			},
		}
		containerCommand = r.ccRuntime.Spec.Config.InstallCmd
	}

	if operation == UninstallOperation {
		containerCommand = r.ccRuntime.Spec.Config.UninstallCmd
	}

	var debug = strconv.FormatBool(r.ccRuntime.Spec.Config.Debug)

	var defaultShim = ""
	var createDefaultRuntimeClass = "false"
	if strings.HasPrefix(r.ccRuntime.Spec.Config.DefaultRuntimeClassName, "kata-") {
		// Remove the "kata-" prefix from DefaultRuntimeClassName
		defaultShim = strings.TrimPrefix(r.ccRuntime.Spec.Config.DefaultRuntimeClassName, "kata-")
		createDefaultRuntimeClass = "true"
	}

	var shims []string
	var snapshotter_handler_mapping []string
	var pull_type_mapping []string
	for _, runtimeClass := range r.ccRuntime.Spec.Config.RuntimeClasses {
		// Similarly to what's being done for the default shim, let's remove
		// the "kata-" prefix from the runtime class names
		shim := strings.TrimPrefix(runtimeClass.Name, "kata-")
		shims = append(shims, shim)

		if runtimeClass.Snapshotter != "" {
			mapping := shim + ":" + runtimeClass.Snapshotter
			snapshotter_handler_mapping = append(snapshotter_handler_mapping, mapping)
		}

		if runtimeClass.PullType != "" {
			mapping := shim + ":" + runtimeClass.PullType
			pull_type_mapping = append(pull_type_mapping, mapping)
		}
	}

	var usingNFD = strconv.FormatBool(r.ccRuntime.Spec.Config.UsingNFD)

	var envVars = []corev1.EnvVar{
		{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name:  "DEBUG",
			Value: debug,
		},
		{
			Name:  "DEFAULT_SHIM",
			Value: defaultShim,
		},
		{
			Name:  "CREATE_DEFAULT_RUNTIMECLASS",
			Value: createDefaultRuntimeClass,
		},
		{
			Name:  "CREATE_RUNTIMECLASSES",
			Value: "true",
		},
		{
			Name:  "SHIMS",
			Value: strings.Join(shims, " "),
		},
		{
			Name:  "SNAPSHOTTER_HANDLER_MAPPING",
			Value: strings.Join(snapshotter_handler_mapping, ","),
		},
		{
			Name:  "PULL_TYPE_MAPPING",
			Value: strings.Join(pull_type_mapping, ","),
		},
		{
			Name:  "USING_NFD",
			Value: usingNFD,
		},
	}
	envVars = append(envVars, r.ccRuntime.Spec.Config.EnvironmentVariables...)

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: r.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: dsLabelSelectors,
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
					Labels: dsLabelSelectors,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "cc-operator-controller-manager",
					NodeSelector:       nodeSelector,
					Tolerations:        r.ccRuntime.Spec.CcTolerations,
					HostPID:            true,
					Containers: []corev1.Container{
						{
							Name:            "cc-runtime-install-pod",
							Image:           r.ccRuntime.Spec.Config.PayloadImage,
							ImagePullPolicy: imagePullPolicyOrDefault(r.ccRuntime.Spec.Config.ImagePullPolicy),
							Lifecycle:       preStopHook,
							SecurityContext: &corev1.SecurityContext{
								// TODO - do we really need to run as root?
								Privileged: &runPrivileged,
								RunAsUser:  &runAsUser,
							},
							Command:      containerCommand,
							Env:          envVars,
							VolumeMounts: r.ccRuntime.Spec.Config.InstallerVolumeMounts,
						},
					},
					Volumes: r.ccRuntime.Spec.Config.InstallerVolumes,
				},
			},
		},
	}
}

func (r *CcRuntimeReconciler) addFinalizer() error {
	r.Log.Info("Adding Finalizer for the RuntimeConfig")
	controllerutil.AddFinalizer(r.ccRuntime, RuntimeConfigFinalizer)

	// Update CR
	err := r.Update(context.TODO(), r.ccRuntime)
	if err != nil {
		r.Log.Error(err, "Failed to update ccRuntime with finalizer")
		return err
	}
	return nil
}

// Get Nodes container specific labels
func (r *CcRuntimeReconciler) getNodesWithLabels(nodeLabels map[string]string) (*corev1.NodeList, error) {
	nodes := &corev1.NodeList{}
	labelSelector := labels.SelectorFromSet(nodeLabels)
	listOpts := []client.ListOption{
		client.MatchingLabelsSelector{Selector: labelSelector},
	}

	if err := r.List(context.TODO(), nodes, listOpts...); err != nil {
		r.Log.Error(err, "Getting list of nodes having specified labels failed")
		return &corev1.NodeList{}, err
	}
	return nodes, nil
}

func (r *CcRuntimeReconciler) mapCcRuntimeToRequests(ctx context.Context, ccRuntimeObj client.Object) []reconcile.Request {
	ccRuntimeList := &ccv1beta1.CcRuntimeList{}

	err := r.List(ctx, ccRuntimeList)
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
			&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(r.mapCcRuntimeToRequests)).
		Complete(r)
}
func (r *CcRuntimeReconciler) deleteUninstallDaemonsets() (ctrl.Result, error) {
	ds := r.processDaemonset(UninstallOperation)
	result, err := r.deleteDaemonset(ds)
	if err != nil {
		return result, err
	}
	ds = r.makeHookDaemonset(PostUninstallOperation)
	result, err = r.deleteDaemonset(ds)
	if err != nil {
		return result, err
	}
	ds = r.makeHookDaemonset(PreInstallOperation)
	result, err = r.deleteDaemonset(ds)
	if err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) deleteDaemonset(ds *appsv1.DaemonSet) (ctrl.Result, error) {
	err := r.Delete(context.TODO(), ds)
	if err != nil && !errors.IsNotFound(err) && !errors.IsGone(err) {
		r.Log.Error(err, "Couldn't delete Daemonset ", "Name:", ds.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CcRuntimeReconciler) makeHookDaemonset(operation DaemonOperation) *appsv1.DaemonSet {
	var (
		runPrivileged       = true
		runAsUser     int64 = 0
		image               = "" //nolint: ineffassign
		dsName        string
		volumes       []corev1.Volume
		volumeMounts  []corev1.VolumeMount
		envVars       = []corev1.EnvVar{
			{
				Name: "NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
		}
	)

	envVars = append(envVars, r.ccRuntime.Spec.Config.EnvironmentVariables...)

	switch operation {
	case PreInstallOperation:
		dsName = "cc-operator-pre-install-daemon"
		image = r.ccRuntime.Spec.Config.PreInstall.Image
		volumeMounts = r.ccRuntime.Spec.Config.PreInstall.VolumeMounts
		volumes = r.ccRuntime.Spec.Config.PreInstall.Volumes
		envVars = append(envVars, r.ccRuntime.Spec.Config.PreInstall.EnvironmentVariables...)
	case PostUninstallOperation:
		dsName = "cc-operator-post-uninstall-daemon"
		image = r.ccRuntime.Spec.Config.PostUninstall.Image
		volumeMounts = r.ccRuntime.Spec.Config.PostUninstall.VolumeMounts
		volumes = r.ccRuntime.Spec.Config.PostUninstall.Volumes
		envVars = append(envVars, r.ccRuntime.Spec.Config.PostUninstall.EnvironmentVariables...)
	default:
		dsName = "invalid operation"
		image = "invalid image"
		volumeMounts = []corev1.VolumeMount{}
		volumes = []corev1.Volume{}
		envVars = []corev1.EnvVar{}
	}

	dsLabelSelectors := map[string]string{
		"name": dsName,
	}

	var nodeSelector map[string]string
	if r.ccRuntime.Spec.CcNodeSelector != nil {
		nodeSelector = r.ccRuntime.Spec.CcNodeSelector.MatchLabels
	} else {
		nodeSelector = map[string]string{
			"node.kubernetes.io/worker": "",
		}
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: r.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: dsLabelSelectors,
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
					Labels: dsLabelSelectors,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "cc-operator-controller-manager",
					NodeSelector:       nodeSelector,
					Tolerations:        r.ccRuntime.Spec.CcTolerations,
					HostPID:            true,
					Containers: []corev1.Container{
						{
							Name:            "cc-runtime-" + string(operation) + "-pod",
							Image:           image,
							ImagePullPolicy: imagePullPolicyOrDefault(r.ccRuntime.Spec.Config.ImagePullPolicy),
							SecurityContext: &corev1.SecurityContext{
								Privileged: &runPrivileged,
								RunAsUser:  &runAsUser,
							},
							Command: []string{"/bin/sh", "-c", "/opt/confidential-containers-pre-install-artifacts/scripts/" + string(operation) + ".sh"},
							Env:     envVars,

							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
}

func (r *CcRuntimeReconciler) removeNodeLabels(nodesList *corev1.NodeList) (ctrl.Result, error) {
	nodesClient, err := r.getNodeClient()
	if err != nil {
		r.Log.Info("Couldn't get nodes client")
		return ctrl.Result{}, err
	}

	kataCleanupDoneLabel := make([]string, 0, len(r.ccRuntime.Spec.Config.UninstallDoneLabel))
	for key := range r.ccRuntime.Spec.Config.UninstallDoneLabel {
		kataCleanupDoneLabel = append(kataCleanupDoneLabel, key)
	}
	if len(kataCleanupDoneLabel) != 1 {
		return ctrl.Result{}, fmt.Errorf("UninstallDoneLabel must only have one entry")
	}

	for _, node := range nodesList.Items {
		nodeLabels := node.GetLabels()
		if val, ok := nodeLabels[PreInstallDoneLabel[0]]; ok && val == PreInstallDoneLabel[1] {
			delete(nodeLabels, PreInstallDoneLabel[0])
		}

		if val, ok := nodeLabels[PostUninstallDoneLabel[0]]; ok && val == PostUninstallDoneLabel[1] {
			delete(nodeLabels, PostUninstallDoneLabel[0])
		}

		if val, ok := nodeLabels[kataCleanupDoneLabel[0]]; ok && val == "cleanup" {
			delete(nodeLabels, kataCleanupDoneLabel[0])
		}
		if val, ok := nodeLabels[StartUninstallLabel[0]]; ok && val == StartUninstallLabel[1] {
			delete(nodeLabels, StartUninstallLabel[0])
		}
		node.SetLabels(nodeLabels)
		_, err := nodesClient.Update(context.TODO(), &node, metav1.UpdateOptions{
			TypeMeta:     metav1.TypeMeta{},
			DryRun:       nil,
			FieldManager: "",
		})
		if err != nil {
			r.Log.Info("failed to update node labels")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
