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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MyComputeClassSpec defines the desired state of MyComputeClass.
type MyComputeClassSpec struct {
	Properties []InstanceProperty `json:"properties"`
}

// MyComputeClassStatus defines the observed state of MyComputeClass.
type MyComputeClassStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

type InstanceProperty struct {
	MachineFamily string `json:"machineFamily"`
	Priority      int    `json:"priority"`
	Spot          *bool  `json:"spot,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MyComputeClass is the Schema for the mycomputeclasses API.
type MyComputeClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyComputeClassSpec   `json:"spec,omitempty"`
	Status MyComputeClassStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MyComputeClassList contains a list of MyComputeClass.
type MyComputeClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyComputeClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MyComputeClass{}, &MyComputeClassList{})
}
