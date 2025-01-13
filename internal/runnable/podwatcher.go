package runnable

import (
	"context"
	"fmt"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	scalingv1 "tryu.com/my-computeclass/api/v1"
)

type PodWatcher struct {
	Client client.Client
}

func (p *PodWatcher) InitializeSettings(ctx context.Context) ([]scalingv1.InstanceProperty, string, error) {
	var myComputeClassList scalingv1.MyComputeClassList
	if err := p.Client.List(ctx, &myComputeClassList); err != nil {
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

func (p *PodWatcher) addTolerationWithSecondPriority(ctx context.Context, pod *corev1.Pod, machineFamily string) {
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

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;patch

// Start implements the Runnable interface
// https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager#Runnable
func (p *PodWatcher) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	ticker := time.NewTicker(3 * time.Minute) // chech every 3 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, secondPriorityMachineFamily, err := p.InitializeSettings(ctx)
			if err != nil {
				logger.Error(err, "Failed to initialize settings")
				continue
			}

			if secondPriorityMachineFamily == "" {
				logger.Info("No second priority machine family found, skipping")
				continue
			}

			var podList corev1.PodList
			if err := p.Client.List(ctx, &podList); err != nil {
				logger.Error(err, "Failed to list Pods")
				continue
			}

			// Filter Pods that are not Running
			nonRunningPods := []corev1.Pod{}
			for _, pod := range podList.Items {
				if pod.Status.Phase != corev1.PodRunning {
					nonRunningPods = append(nonRunningPods, pod)
					logger.Info("Found non-running Pod", "PodName", pod.Name, "Namespace", pod.Namespace)
				}
			}

			// wait 2 minutes and check the status
			time.Sleep(2 * time.Minute)
			for _, pod := range nonRunningPods {
				var updatedPod corev1.Pod
				if err := p.Client.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, &updatedPod); err != nil {
					logger.Error(err, "Failed to get updated Pod", "PodName", pod.Name, "Namespace", pod.Namespace)
					continue
				}

				// Check if the Pod is still not Running
				if updatedPod.Status.Phase != corev1.PodRunning {
					logger.Info("Pod is still not Running", "PodName", updatedPod.Name, "Namespace", updatedPod.Namespace)

					// Add toleration with second priority
					p.addTolerationWithSecondPriority(ctx, &updatedPod, secondPriorityMachineFamily)
					if err := p.Client.Update(ctx, &updatedPod); err != nil {
						logger.Error(err, "Failed to update Pod", "PodName", updatedPod.Name)
					} else {
						logger.Info("Toleration added successfully", "PodName", updatedPod.Name, "Namespace", updatedPod.Namespace)
					}
				} else {
					logger.Info("Pod transitioned to Running state", "PodName", updatedPod.Name, "Namespace", updatedPod.Namespace)
				}
			}
		case <-ctx.Done():
			logger.Info("Stopping PodWatcher")
			return nil
		}
	}
}
