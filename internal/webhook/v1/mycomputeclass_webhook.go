package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	scalingv1 "tryu.com/my-computeclass/api/v1"
)

// log is for logging in this package.
var mycomputeclasslog = logf.Log.WithName("mycomputeclass-resource")

// SetupMyComputeClassWebhookWithManager registers the webhook for MyComputeClass in the manager.
func SetupMyComputeClassWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&scalingv1.MyComputeClass{}).
		WithValidator(&MyComputeClassCustomValidator{}).
		WithDefaulter(&MyComputeClassCustomDefaulter{}).
		Complete()
}

// CustomDefaulterWrapper wraps admission.CustomDefaulter interface to provide admission.Handler interface.
type CustomDefaulterWrapper struct {
	Defaulter admission.CustomDefaulter
}

// MyComputeClassCustomDefaulter sets default values for MyComputeClass.
type MyComputeClassCustomDefaulter struct {
	Client client.Client
}

var _ admission.CustomDefaulter = &MyComputeClassCustomDefaulter{}

// Handle implements admission.Handler interface by invoking CustomDefaulter's Default method.
// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/webhook/admission#Handler
func (w *CustomDefaulterWrapper) Handle(ctx context.Context, req admission.Request) admission.Response {
	var obj runtime.Object

	switch req.Kind.Kind {
	case "Pod":
		obj = &corev1.Pod{}
	case "Node":
		obj = &corev1.Node{}
	default:
		return admission.Errored(400, fmt.Errorf("unsupported resource type: %s", req.Kind.Kind))
	}

	if err := json.Unmarshal(req.Object.Raw, obj); err != nil {
		return admission.Errored(400, fmt.Errorf("failed to unmarshal resource: %w", err))
	}

	if err := w.Defaulter.Default(ctx, obj); err != nil {
		return admission.Errored(500, err)
	}

	marshaledObj, err := json.Marshal(obj)
	if err != nil {
		return admission.Errored(500, fmt.Errorf("failed to marshal object: %w", err))
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledObj)
}

func (d *MyComputeClassCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	switch obj := obj.(type) {
	case *corev1.Pod:
		return d.AddPodSettings(ctx, obj)
	case *corev1.Node:
		return d.AddTaints(ctx, obj)
	default:
		return fmt.Errorf("unsupported resource type: %T", obj)
	}
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind MyComputeClass.
func (d *MyComputeClassCustomDefaulter) AddPodSettings(ctx context.Context, pod *corev1.Pod) error {
	mycomputeclasslog.Info("Applying default toleration to Pod", "podName", pod.GetName())

	// get the priority list and top machine family
	priorityList, topPriorityMachineFamily, err := d.InitializeSettings(ctx)
	if err != nil {
		return err
	}

	// add tolerations
	d.addTolerations(pod, topPriorityMachineFamily)
	// add node affinity(avoid scheduled to default e2 instances)
	d.addNodeAffinity(pod, priorityList)

	// check custom resource for Spot instance
	if priorityList[0].Spot != nil && *priorityList[0].Spot {
		d.addSpotNodeSelector(pod)
	}
	if priorityList[0].Spot != nil && !*priorityList[0].Spot {
		d.addNonSpotNodeAffinity(pod)
	}
	return nil
}

func (d *MyComputeClassCustomDefaulter) AddTaints(ctx context.Context, node *corev1.Node) error {
	mycomputeclasslog.Info("Applying default taint to Node", "nodeName", node.GetName())

	// get the machineFamily label from the Node
	machineFamily, exists := node.Labels["cloud.google.com/machine-family"]
	if !exists || machineFamily == "" {
		mycomputeclasslog.Info("Node does not have a machineFamily label, skipping taint application", "nodeName", node.GetName())
		return nil
	}

	// make a taint
	taint := corev1.Taint{
		Key:    "my-compute-class",
		Value:  machineFamily,
		Effect: corev1.TaintEffectNoSchedule,
	}

	// check if the taint already exists
	for _, existingTaint := range node.Spec.Taints {
		if existingTaint.Key == taint.Key {
			mycomputeclasslog.Info("Taint already exists on Node, skipping addition", "nodeName", node.GetName(), "key", taint.Key)
			return nil
		}
	}

	// add the taint to the node
	node.Spec.Taints = append(node.Spec.Taints, taint)

	mycomputeclasslog.Info("Taint successfully applied to Node", "nodeName", node.GetName(), "taint", taint)
	return nil
}

// MyComputeClassCustomValidator validates the MyComputeClass resource when it is created, updated, or deleted.
type MyComputeClassCustomValidator struct{}

var _ webhook.CustomValidator = &MyComputeClassCustomValidator{}

// ValidateCreate validates MyComputeClass upon creation.
func (v *MyComputeClassCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	mycomputeclass, ok := obj.(*scalingv1.MyComputeClass)
	if !ok {
		return nil, fmt.Errorf("expected MyComputeClass object but got %T", obj)
	}
	mycomputeclasslog.Info("Validation for MyComputeClass upon creation", "name", mycomputeclass.GetName())

	// TODO(user): Add validation logic upon creation.

	return nil, nil
}

// ValidateUpdate validates MyComputeClass upon update.
func (v *MyComputeClassCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	mycomputeclass, ok := newObj.(*scalingv1.MyComputeClass)
	if !ok {
		return nil, fmt.Errorf("expected MyComputeClass object but got %T", newObj)
	}
	mycomputeclasslog.Info("Validation for MyComputeClass upon update", "name", mycomputeclass.GetName())

	// TODO(user): Add validation logic upon update.

	return nil, nil
}

// ValidateDelete validates MyComputeClass upon deletion.
func (v *MyComputeClassCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	mycomputeclass, ok := obj.(*scalingv1.MyComputeClass)
	if !ok {
		return nil, fmt.Errorf("expected MyComputeClass object but got %T", obj)
	}
	mycomputeclasslog.Info("Validation for MyComputeClass upon deletion", "name", mycomputeclass.GetName())

	// TODO(user): Add validation logic upon deletion.

	return nil, nil
}

func (d *MyComputeClassCustomDefaulter) InitializeSettings(ctx context.Context) ([]scalingv1.InstanceProperty, string, error) {
	var myComputeClassList scalingv1.MyComputeClassList
	if err := d.Client.List(ctx, &myComputeClassList); err != nil {
		mycomputeclasslog.Error(err, "Failed to list MyComputeClass resources")
		return nil, "", fmt.Errorf("failed to list MyComputeClass resources: %w", err)
	}

	var priorityList []scalingv1.InstanceProperty
	for _, myComputeClass := range myComputeClassList.Items {
		priorityList = append(priorityList, myComputeClass.Spec.Properties...)
	}

	if len(priorityList) == 0 {
		mycomputeclasslog.Info("No priority list defined, skipping defaulting")
		return nil, "", nil
	}

	sort.Slice(priorityList, func(i, j int) bool {
		return priorityList[i].Priority < priorityList[j].Priority
	})
	topPriorityMachineFamily := priorityList[0].MachineFamily
	mycomputeclasslog.Info("Top priority instance type", "machineFamily", topPriorityMachineFamily)

	return priorityList, topPriorityMachineFamily, nil
}

func (d *MyComputeClassCustomDefaulter) addNonSpotNodeAffinity(pod *corev1.Pod) {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{}
	}

	// Collect existing NodeSelectorTerms
	existingTerms := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

	// Merge existing terms with the "non-spot" condition
	mergedSelector := corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      "cloud.google.com/gke-preemptible",
				Operator: corev1.NodeSelectorOpNotIn,
				Values:   []string{"true"},
			},
		},
	}

	// Add existing terms as part of the new MatchExpressions
	for _, term := range existingTerms {
		mergedSelector.MatchExpressions = append(mergedSelector.MatchExpressions, term.MatchExpressions...)
	}

	// Replace all terms with the merged selector
	pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []corev1.NodeSelectorTerm{mergedSelector}

	mycomputeclasslog.Info("Updated NodeAffinity for non-spot with merged conditions", "podName", pod.GetName())
}

// add node selector for spot instance
func (d *MyComputeClassCustomDefaulter) addSpotNodeSelector(pod *corev1.Pod) {
	if pod.Spec.NodeSelector == nil {
		pod.Spec.NodeSelector = map[string]string{}
	}

	// Add node selector for spot instances
	pod.Spec.NodeSelector["cloud.google.com/gke-preemptible"] = "true"
	pod.Spec.NodeSelector["cloud.google.com/gke-provisioning"] = "preemptible"

	mycomputeclasslog.Info("Added spot node selector to Pod", "podName", pod.GetName())
}

// addTolerations adds a toleration to the Pod if it doesn't already exist.
func (d *MyComputeClassCustomDefaulter) addTolerations(pod *corev1.Pod, machineFamily string) {
	tolerationExists := false
	for _, toleration := range pod.Spec.Tolerations {
		if toleration.Key == "my-compute-class" && toleration.Value == machineFamily {
			tolerationExists = true
			mycomputeclasslog.Info("Toleration already exists", "podName", pod.GetName(), "machineFamily", machineFamily)
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
		mycomputeclasslog.Info("Toleration added", "podName", pod.GetName(), "machineFamily", machineFamily)
	}
}

// addNodeAffinity adds NodeAffinity rules to the Pod for each machine family in the priority list.
func (d *MyComputeClassCustomDefaulter) addNodeAffinity(pod *corev1.Pod, priorityList []scalingv1.InstanceProperty) {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{}
	}

	// Combine all machine families into a single NodeSelectorTerm
	machineFamilies := []string{}
	for _, property := range priorityList {
		machineFamilies = append(machineFamilies, property.MachineFamily)
	}

	nodeSelector := corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      "cloud.google.com/machine-family",
				Operator: corev1.NodeSelectorOpIn,
				Values:   machineFamilies,
			},
		},
	}

	// Append the combined NodeSelectorTerm
	pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
		nodeSelector,
	)

	mycomputeclasslog.Info("NodeAffinity added", "podName", pod.GetName(), "machineFamilies", machineFamilies)
}
