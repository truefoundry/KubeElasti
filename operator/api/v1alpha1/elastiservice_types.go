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

// EnabledPeriod defines when the scale-to-zero policy is active.
// Outside of this period, services maintain minTargetReplicas and scale-down is prevented.
type EnabledPeriod struct {
	// Schedule is a 5-item cron expression (minute hour day month weekday).
	// Uses UTC timezone. Example: "0 9 * * 1-5" for 9 AM Monday-Friday.
	// +kubebuilder:default="0 0 * * *"
	Schedule string `json:"schedule,omitempty"`

	// Duration specifies how long the enabled period lasts from each scheduled trigger.
	// Accepts formats like "1h", "30m", "8h", etc.
	// +kubebuilder:default="24h"
	Duration string `json:"duration,omitempty"`
}

// +kubebuilder:validation:Required={"scaleTargetRef","service"}
type ElastiServiceSpec struct {
	// ScaleTargetRef of the target resource to scale
	ScaleTargetRef ScaleTargetRef `json:"scaleTargetRef"`
	// Service to scale
	Service string `json:"service"`
	// Minimum number of replicas to scale to
	// +kubebuilder:validation:Minimum=1
	MinTargetReplicas int32 `json:"minTargetReplicas,omitempty" default:"1"`
	// Cooldown period in seconds.
	// It tells how long a target resource can be idle before scaling it down
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=604800
	// +kubebuilder:default=900
	CooldownPeriod int32 `json:"cooldownPeriod,omitempty"`
	// Triggers to scale the target resource
	// +kubebuilder:validation:MinItems=1
	Triggers   []ScaleTrigger  `json:"triggers,omitempty"`
	Autoscaler *AutoscalerSpec `json:"autoscaler,omitempty"`
	// EnabledPeriod defines when the scale-to-zero policy is active.
	// When omitted, scale-to-zero is always enabled (default behavior).
	// When specified, scale-down only occurs during the cron schedule window.
	EnabledPeriod *EnabledPeriod `json:"enabledPeriod,omitempty"`
}

func (es *ElastiServiceSpec) GetScaleTargetRef() ScaleTargetRef {
	// NOTE: Required for backwards compatibility, since so far, we have been using "deployments" instead of "Deployment" in exisiting
	// CRD files. Since calse doesn't recognize "deployments" as a valid kind, we need to convert it to "Deployment".
	// We can remove it once we have migrated all the existing CRD files to use "Deployment" instead of "deployments".
	switch es.ScaleTargetRef.Kind {
	case "deployments":
		es.ScaleTargetRef.Kind = "Deployment"
	case "rollouts":
		es.ScaleTargetRef.Kind = "Rollout"
	default:
		return es.ScaleTargetRef
	}

	return es.ScaleTargetRef
}

type ScaleTargetRef struct {
	// API version of the target resource
	// +kubebuilder:validation:Enum=apps/v1;argoproj.io/v1alpha1
	APIVersion string `json:"apiVersion"`
	// Kind of the target resource
	// +kubebuilder:validation:Enum=deployments;rollouts;Deployment;StatefulSet;Rollout
	Kind string `json:"kind"`
	// Name of the target resource
	Name string `json:"name"`
}

type ElastiServiceStatus struct {
	// Last time the ElastiService was reconciled
	LastReconciledTime metav1.Time `json:"lastReconciledTime,omitempty"`
	// Last time the ElastiService was scaled up
	LastScaledUpTime *metav1.Time `json:"lastScaledUpTime,omitempty"`
	// Current mode of the ElastiService, either "proxy" or "serve".
	// "proxy" mode is when the ScaleTargetRef is scaled to 0 replicas.
	// "serve" mode is when the ScaleTargetRef is scaled to at least 1 replica.
	Mode string `json:"mode,omitempty"`
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

func (es *ElastiService) GetSpec() ElastiServiceSpec {
	es.Spec.ScaleTargetRef = es.Spec.GetScaleTargetRef()
	return es.Spec
}

//+kubebuilder:object:root=true

// ElastiServiceList contains a list of ElastiService
type ElastiServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElastiService `json:"items"`
}

type ScaleTrigger struct {
	// Type of the trigger, currently only prometheus is supported
	// +kubebuilder:validation:Enum=prometheus
	Type string `json:"type"`
	// Metadata like query, serverAddress, threshold, uptimeFilter etc.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type AutoscalerSpec struct {
	// +kubebuilder:validation:Enum=hpa;keda
	Type string `json:"type"`
	Name string `json:"name"`
}

func init() {
	SchemeBuilder.Register(&ElastiService{}, &ElastiServiceList{})
}
