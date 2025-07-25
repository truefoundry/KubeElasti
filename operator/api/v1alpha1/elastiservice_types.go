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

package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ElastiServiceFinalizer = "elasti.truefoundry.com/finalizer"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ElastiServiceSpec defines the desired state of ElastiService
type ElastiServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Required
	ScaleTargetRef ScaleTargetRef `json:"scaleTargetRef,omitempty"`
	// +kubebuilder:validation:Required
	Service string `json:"service,omitempty"`
	// +kubebuilder:validation:Minimum=1
	MinTargetReplicas int32 `json:"minTargetReplicas,omitempty" default:"1"`
	// This is the cooldown period in seconds
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=604800
	// +kubebuilder:default=900
	CooldownPeriod int32           `json:"cooldownPeriod,omitempty"`
	Triggers       []ScaleTrigger  `json:"triggers,omitempty"`
	Autoscaler     *AutoscalerSpec `json:"autoscaler,omitempty"`
}

type ScaleTargetRef struct {
	// +kubebuilder:validation:Enum=apps/v1;argoproj.io/v1alpha1
	APIVersion string `json:"apiVersion,omitempty"`
	// +kubebuilder:validation:Enum=deployments;rollouts
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}

// ElastiServiceStatus defines the observed state of ElastiService
type ElastiServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastReconciledTime metav1.Time  `json:"lastReconciledTime,omitempty"`
	LastScaledUpTime   *metav1.Time `json:"lastScaledUpTime,omitempty"`
	Mode               string       `json:"mode,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ElastiService is the Schema for the elastiservices API
type ElastiService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElastiServiceSpec   `json:"spec,omitempty"`
	Status ElastiServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ElastiServiceList contains a list of ElastiService
type ElastiServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElastiService `json:"items"`
}

type ScaleTrigger struct {
	// +kubebuilder:validation:Enum=prometheus
	Type string `json:"type"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type AutoscalerSpec struct {
	// +kubebuilder:validation:Enum=hpa;keda
	Type string `json:"type"`
	Name string `json:"name"` // Name of the ScaledObject/HorizontalPodAutoscaler
}

func init() {
	SchemeBuilder.Register(&ElastiService{}, &ElastiServiceList{})
}
