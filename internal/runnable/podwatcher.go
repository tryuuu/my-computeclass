package runnable

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	scalingv1 "tryu.com/my-computeclass/api/v1"
)

type PodWatcher struct {
	Client    client.Client
	Cache     cache.Cache
	Namespace string
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
func (p *PodWatcher) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	if p.Cache == nil {
		logger.Error(nil, "Cache is not set")
		return fmt.Errorf("cache is not set")
	}
	informer, err := p.Cache.GetInformer(ctx, &corev1.Pod{})
	if err != nil {
		logger.Error(err, "Failed to get Pod Informer")
		return err
	}

	_, err = informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			p.handlePod(ctx, obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			p.handlePod(ctx, newObj)
		},
	})
	if err != nil {
		logger.Error(err, "Failed to add event handler")
		return err
	}

	go func() {
		if err := p.Cache.Start(ctx); err != nil {
			logger.Error(err, "Failed to start cache")
		}
	}()

	if !p.Cache.WaitForCacheSync(ctx) {
		logger.Error(nil, "Cache failed to sync")
		return fmt.Errorf("cache sync failed")
	}

	logger.Info("PodWatcher started")
	<-ctx.Done()
	logger.Info("Stopping PodWatcher")
	return nil
}

func (p *PodWatcher) handlePod(ctx context.Context, obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}

	if pod.Status.Phase == corev1.PodPending {
		log.FromContext(ctx).Info("Pending Pod detected", "PodName", pod.GetName())
		p.processPendingPod(ctx, pod)
	}
}

func (p *PodWatcher) processPendingPod(ctx context.Context, pod *corev1.Pod) {
	logger := log.FromContext(ctx)

	// get the latest version of the Pod
	var latestPod corev1.Pod
	if err := p.Client.Get(ctx, client.ObjectKey{
		Namespace: pod.Namespace,
		Name:      pod.Name,
	}, &latestPod); err != nil {
		logger.Error(err, "Failed to fetch the latest version of Pod", "PodName", pod.Name)
		return
	}

	_, secondPriorityMachineFamily, err := p.InitializeSettings(ctx)
	if err != nil {
		logger.Error(err, "Failed to initialize settings")
		return
	}

	p.addTolerationWithSecondPriority(ctx, pod, secondPriorityMachineFamily)
	if err := p.Client.Update(ctx, pod); err != nil {
		logger.Error(err, "Failed to update Pod", "PodName", pod.Name)
	} else {
		logger.Info("Toleration added successfully", "PodName", pod.Name, "Namespace", pod.Namespace)
	}
}
