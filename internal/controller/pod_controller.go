package controller

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	scalingv1 "tryu.com/my-computeclass/api/v1"
)

type PodReconciler struct {
	Client client.Client
}

func (r *PodReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	var pod corev1.Pod
	if err := r.Client.Get(ctx, req.NamespacedName, &pod); err != nil {
		logger.Error(err, "Failed to get Pod")
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if pod.Status.Phase == corev1.PodPending {
		logger.Info("Pending Pod detected", "PodName", pod.GetName())
		r.processPendingPod(ctx, &pod)
	}

	return reconcile.Result{}, nil
}

func (r *PodReconciler) processPendingPod(ctx context.Context, pod *corev1.Pod) {
	logger := log.FromContext(ctx)

	_, secondPriorityMachineFamily, err := r.initializeSettings(ctx)
	if err != nil {
		logger.Error(err, "Failed to initialize settings")
		return
	}

	r.addTolerationWithSecondPriority(ctx, pod, secondPriorityMachineFamily)
	if err := r.Client.Update(ctx, pod); err != nil {
		logger.Error(err, "Failed to update Pod", "PodName", pod.Name)
	} else {
		logger.Info("Toleration added successfully", "PodName", pod.Name, "Namespace", pod.Namespace)
	}
}

func (r *PodReconciler) initializeSettings(ctx context.Context) ([]scalingv1.InstanceProperty, string, error) {
	var myComputeClassList scalingv1.MyComputeClassList
	if err := r.Client.List(ctx, &myComputeClassList); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list MyComputeClass resources")
		return nil, "", fmt.Errorf("failed to list MyComputeClass resources: %w", err)
	}

	var priorityList []scalingv1.InstanceProperty
	for _, myComputeClass := range myComputeClassList.Items {
		priorityList = append(priorityList, myComputeClass.Spec.Properties...)
	}

	if len(priorityList) < 2 {
		log.FromContext(ctx).Info("Less than 2 priorities defined, skipping")
		return nil, "", nil
	}

	sort.Slice(priorityList, func(i, j int) bool {
		return priorityList[i].Priority < priorityList[j].Priority
	})

	secondPriorityMachineFamily := priorityList[1].MachineFamily
	log.FromContext(ctx).Info("Second priority instance type", "machineFamily", secondPriorityMachineFamily)

	return priorityList, secondPriorityMachineFamily, nil
}

func (r *PodReconciler) addTolerationWithSecondPriority(ctx context.Context, pod *corev1.Pod, machineFamily string) {
	tolerationExists := false
	for _, toleration := range pod.Spec.Tolerations {
		if toleration.Key == "my-compute-class" && toleration.Value == machineFamily {
			tolerationExists = true
			log.FromContext(ctx).Info("Toleration already exists", "podName", pod.GetName(), "machineFamily", machineFamily)
			break
		}
	}

	if !tolerationExists {
		pod.Spec.Tolerations = append(pod.Spec.Tolerations, corev1.Toleration{
			Key:      "my-compute-class",
			Operator: corev1.TolerationOpEqual,
			Value:    machineFamily,
			Effect:   corev1.TaintEffectNoSchedule,
		})
		log.FromContext(ctx).Info("Toleration added", "podName", pod.GetName(), "machineFamily", machineFamily)
	}
}

func (r *PodReconciler) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
