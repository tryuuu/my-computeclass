/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	scalingv1 "tryu.com/my-computeclass/api/v1"
)

// MyComputeClassReconciler reconciles a MyComputeClass object
type MyComputeClassReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=scaling.tryu.com,resources=mycomputeclasses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scaling.tryu.com,resources=mycomputeclasses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scaling.tryu.com,resources=mycomputeclasses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyComputeClass object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *MyComputeClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Initialize GKE client
	gkeClient, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		logger.Error(err, "Failed to create GKE client")
		return ctrl.Result{}, err
	}
	logger.Info("GKE client created")

	// Fetch the MyComputeClass instance
	instance := &scalingv1.MyComputeClass{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		logger.Error(err, "Failed to get MyComputeClass")
		return ctrl.Result{}, err
	}

	projectID := "ryu-project-441804"
	location := "asia-northeast1"
	clusterName := "sreake-intern-tryu-gke"

	// https://cloud.google.com/python/docs/reference/container/latest/google.cloud.container_v1.types.ListNodePoolsRequest
	reqNodePools := &containerpb.ListNodePoolsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, clusterName),
	}
	if err != nil {
		logger.Error(err, "Failed to list NodePools")
		return ctrl.Result{}, err
	}
	// list NodePools
	resp, err := gkeClient.ListNodePools(ctx, reqNodePools)
	if err != nil {
		logger.Error(err, "Failed to list NodePools")
		return ctrl.Result{}, err
	}
	// Log NodePools information
	for _, nodePool := range resp.NodePools {
		logger.Info("NodePool", "name", nodePool.Name, "status", nodePool.Status)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyComputeClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1.MyComputeClass{}).
		Named("mycomputeclass").
		Complete(r)
}
