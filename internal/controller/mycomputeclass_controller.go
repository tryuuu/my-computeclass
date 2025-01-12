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
	"sort"
	"strings"

	container "cloud.google.com/go/container/apiv1"
	containerpb "cloud.google.com/go/container/apiv1/containerpb"
	corev1 "k8s.io/api/core/v1"
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
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update

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

	gkeClient, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		logger.Error(err, "Failed to create GKE client")
		return ctrl.Result{}, err
	}
	logger.Info("GKE client created")

	priorityList, err := r.HandlePriorityList(ctx, req)
	if err != nil {
		logger.Error(err, "Failed to fetch or sort priority list")
		return ctrl.Result{}, err
	}

	projectID := "ryu-project-441804"
	location := "asia-northeast1"
	clusterName := "sreake-intern-tryu-gke"

	// Provision new NodePool
	err = createNodePool(priorityList, projectID, location, clusterName, gkeClient, ctx)
	if err != nil {
		logger.Error(err, "Failed to create node pool")
		return ctrl.Result{}, err
	}

	reqNodePools := &containerpb.ListNodePoolsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, clusterName),
	}
	// list NodePools
	// https://pkg.go.dev/google.golang.org/cloud/container/apiv1#ClusterManagerClient.ListNodePools
	resp, err := gkeClient.ListNodePools(ctx, reqNodePools)
	if err != nil {
		logger.Error(err, "Failed to list NodePools")
		return ctrl.Result{}, err
	}

	for _, nodePool := range resp.NodePools {
		machineType := ""
		if nodePool.Config != nil {
			// https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/NodeConfig
			machineType = nodePool.Config.MachineType
		}
		logger.Info("NodePool", "name", nodePool.Name, "status", nodePool.Status, "machineType", machineType)
		// apply taint
		if err := r.applyTaintToNodePool(ctx, machineType); err != nil {
			logger.Error(err, "Failed to apply taint")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func createNodePool(priorityList []scalingv1.InstanceProperty, projectID, location, clusterName string, gkeClient *container.ClusterManagerClient, ctx context.Context) error {
	logger := log.FromContext(ctx)
	var err error

	// Iterate over the priority list to find the first valid configuration
	for _, property := range priorityList {
		nodePoolName := fmt.Sprintf("auto-nodepool-%s", property.MachineFamily)
		// spot node pool
		if property.Spot != nil && *property.Spot {
			nodePoolName = fmt.Sprintf("%s-spot", nodePoolName)
		}
		// NodePool configuration
		newNodePool := &containerpb.NodePool{
			Name: nodePoolName,
			Config: &containerpb.NodeConfig{
				MachineType: fmt.Sprintf("%s-standard-4", property.MachineFamily),
				Preemptible: func() bool {
					if property.Spot != nil {
						return *property.Spot
					}
					return false // default

				}(),
			},
			Autoscaling: &containerpb.NodePoolAutoscaling{
				Enabled:      true,
				MinNodeCount: 1,
				MaxNodeCount: 5,
			},
			InitialNodeCount: 1,
		}

		// Create the NodePool
		req := &containerpb.CreateNodePoolRequest{
			Parent:   fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, clusterName),
			NodePool: newNodePool,
		}

		logger.Info("Creating NodePool", "nodePoolName", nodePoolName, "machineType", property.MachineFamily)
		_, err = gkeClient.CreateNodePool(ctx, req)
		if err != nil {
			logger.Error(err, "Failed to create NodePool", "nodePoolName", nodePoolName)
			continue
		}

		logger.Info("NodePool created successfully", "nodePoolName", nodePoolName)
	}

	logger.Info("No suitable configuration found in priority list for NodePool creation")
	return fmt.Errorf("failed to create NodePool: no valid configurations found")
}

func (r *MyComputeClassReconciler) HandlePriorityList(ctx context.Context, req ctrl.Request) ([]scalingv1.InstanceProperty, error) {
	logger := log.FromContext(ctx)

	// Fetch the MyComputeClass instance
	instance := &scalingv1.MyComputeClass{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		logger.Error(err, "Failed to get MyComputeClass")
		return nil, err
	}

	priorityList := instance.Spec.Properties
	if len(priorityList) == 0 {
		logger.Info("No priority list defined")
		return nil, nil
	}

	// Sort by priority
	sort.Slice(priorityList, func(i, j int) bool {
		return priorityList[i].Priority < priorityList[j].Priority
	})

	return priorityList, nil
}
func (r *MyComputeClassReconciler) applyTaintToNodePool(ctx context.Context, machineType string) error {
	logger := log.FromContext(ctx)
	nodeList := &corev1.NodeList{}
	// use label selector to filter nodes
	if err := r.List(ctx, nodeList, client.MatchingLabels{"beta.kubernetes.io/instance-type": machineType}); err != nil {
		logger.Error(err, "Failed to list nodes for instance type", "instanceType", machineType)
		return err
	}
	for _, node := range nodeList.Items {
		machineFamily := extractMachineFamily(machineType)
		logger.Info("Applying taint to node", "nodeName", node.Name, "machineType", machineFamily)

		// Check if the taint already exists
		taintExists := false
		for _, taint := range node.Spec.Taints {
			if taint.Key == "my-compute-class" && taint.Value == machineFamily {
				taintExists = true
				break
			}
		}
		if taintExists {
			logger.Info("Taint already exists", "nodeName", node.Name, "machineFamily", machineFamily)
			continue
		}

		// Add taint
		newTaint := corev1.Taint{
			Key:    "my-compute-class",
			Value:  machineFamily,
			Effect: corev1.TaintEffectNoSchedule,
		}
		node.Spec.Taints = append(node.Spec.Taints, newTaint)

		// Update node
		if err := r.Update(ctx, &node); err != nil {
			logger.Error(err, "Failed to apply taint to node", "nodeName", node.Name)
			return err
		}
		logger.Info("Taint applied to node", "nodeName", node.Name)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyComputeClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1.MyComputeClass{}).
		Named("mycomputeclass").
		Complete(r)
}

// extracts machine family from machine type
func extractMachineFamily(machineType string) string {
	parts := strings.Split(machineType, "-")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
