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

// CustomDefaulterWrapper wraps admission.CustomDefaulter to provide admission.Handler interface.
type CustomDefaulterWrapper struct {
	Defaulter admission.CustomDefaulter
}

// Handle implements admission.Handler interface by invoking CustomDefaulter's Default method.
func (w *CustomDefaulterWrapper) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Kind.Kind != "Pod" {
		return admission.Errored(400, fmt.Errorf("expected Pod but got %s", req.Kind.Kind))
	}
	obj := &corev1.Pod{}
	if err := json.Unmarshal(req.Object.Raw, obj); err != nil {
		return admission.Errored(400, fmt.Errorf("failed to unmarshal object: %w", err))
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

// MyComputeClassCustomDefaulter sets default values for MyComputeClass.
type MyComputeClassCustomDefaulter struct {
	Client client.Client
}

var _ admission.CustomDefaulter = &MyComputeClassCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind MyComputeClass.
func (d *MyComputeClassCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected Pod but got %T", obj)
	}

	mycomputeclasslog.Info("Applying default toleration to Pod", "podName", pod.GetName())

	var myComputeClassList scalingv1.MyComputeClassList
	if err := d.Client.List(ctx, &myComputeClassList); err != nil {
		mycomputeclasslog.Error(err, "Failed to list MyComputeClass resources")
		return fmt.Errorf("failed to list MyComputeClass resources: %w", err)
	}

	var priorityList []scalingv1.InstanceProperty
	for _, myComputeClass := range myComputeClassList.Items {
		priorityList = append(priorityList, myComputeClass.Spec.Properties...)
	}

	if len(priorityList) == 0 {
		mycomputeclasslog.Info("No priority list defined, skipping defaulting")
		return nil
	}

	sort.Slice(priorityList, func(i, j int) bool {
		return priorityList[i].Priority < priorityList[j].Priority
	})
	topPriorityInstanceType := priorityList[0].InstanceType
	mycomputeclasslog.Info("Top priority instance type", "instanceType", topPriorityInstanceType)

	tolerationExists := false
	for _, toleration := range pod.Spec.Tolerations {
		if toleration.Key == "my-compute-class" && toleration.Value == topPriorityInstanceType {
			tolerationExists = true
			break
		}
	}
	if !tolerationExists {
		pod.Spec.Tolerations = append(pod.Spec.Tolerations, corev1.Toleration{
			Key:      "my-compute-class",
			Operator: corev1.TolerationOpEqual,
			Value:    topPriorityInstanceType,
			Effect:   corev1.TaintEffectNoSchedule,
		})
		mycomputeclasslog.Info("Toleration added", "podName", pod.GetName(), "instanceType", topPriorityInstanceType)
	}

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
