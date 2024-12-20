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

package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	scalingv1 "tryu.com/my-computeclass/api/v1"
)

// nolint:unused
// log is for logging in this package.
var mycomputeclasslog = logf.Log.WithName("mycomputeclass-resource")

// SetupMyComputeClassWebhookWithManager registers the webhook for MyComputeClass in the manager.
func SetupMyComputeClassWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&scalingv1.MyComputeClass{}).
		WithValidator(&MyComputeClassCustomValidator{}).
		WithDefaulter(&MyComputeClassCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-scaling-tryu-com-v1-mycomputeclass,mutating=true,failurePolicy=fail,sideEffects=None,groups=scaling.tryu.com,resources=mycomputeclasses,verbs=create;update,versions=v1,name=mmycomputeclass-v1.kb.io,admissionReviewVersions=v1

// MyComputeClassCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind MyComputeClass when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type MyComputeClassCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &MyComputeClassCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind MyComputeClass.
func (d *MyComputeClassCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	mycomputeclass, ok := obj.(*scalingv1.MyComputeClass)

	if !ok {
		return fmt.Errorf("expected an MyComputeClass object but got %T", obj)
	}
	mycomputeclasslog.Info("Defaulting for MyComputeClass", "name", mycomputeclass.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-scaling-tryu-com-v1-mycomputeclass,mutating=false,failurePolicy=fail,sideEffects=None,groups=scaling.tryu.com,resources=mycomputeclasses,verbs=create;update,versions=v1,name=vmycomputeclass-v1.kb.io,admissionReviewVersions=v1

// MyComputeClassCustomValidator struct is responsible for validating the MyComputeClass resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type MyComputeClassCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &MyComputeClassCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type MyComputeClass.
func (v *MyComputeClassCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	mycomputeclass, ok := obj.(*scalingv1.MyComputeClass)
	if !ok {
		return nil, fmt.Errorf("expected a MyComputeClass object but got %T", obj)
	}
	mycomputeclasslog.Info("Validation for MyComputeClass upon creation", "name", mycomputeclass.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type MyComputeClass.
func (v *MyComputeClassCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	mycomputeclass, ok := newObj.(*scalingv1.MyComputeClass)
	if !ok {
		return nil, fmt.Errorf("expected a MyComputeClass object for the newObj but got %T", newObj)
	}
	mycomputeclasslog.Info("Validation for MyComputeClass upon update", "name", mycomputeclass.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type MyComputeClass.
func (v *MyComputeClassCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	mycomputeclass, ok := obj.(*scalingv1.MyComputeClass)
	if !ok {
		return nil, fmt.Errorf("expected a MyComputeClass object but got %T", obj)
	}
	mycomputeclasslog.Info("Validation for MyComputeClass upon deletion", "name", mycomputeclass.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
