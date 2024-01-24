/*
Copyright 2021 Daisuke Taniwaki.

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
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TemplateMetadata is a metadata type only for labels and annotations.
type TemplateMetadata struct {
	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// HPATemplate is the template of HPA to create.
type HPATemplate struct {
	Metadata *TemplateMetadata                         `json:"metadata,omitempty"`
	Spec     autoscalingv2.HorizontalPodAutoscalerSpec `json:"spec"`
}

// HPAPatch is a patch applied to the template.
type HPAPatch struct {
	// minReplicas is the lower limit for the number of replicas to which the autoscaler
	// can scale down.  It defaults to 1 pod.  minReplicas is allowed to be 0 if the
	// alpha feature gate HPAScaleToZero is enabled and at least one Object or External
	// metric is configured.  Scaling is active as long as at least one metric value is
	// available.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale up.
	// It cannot be less that minReplicas.
	// +optional
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
	// metrics contains the specifications for which to use to calculate the
	// desired replica count (the maximum replica count across all metrics will
	// be used).  The desired replica count is calculated multiplying the
	// ratio between the target value and the current value by the current
	// number of pods.  Ergo, metrics used must decrease as the pod count is
	// increased, and vice-versa.  See the individual metric source types for
	// more information about how each type of metric must respond.
	// +optional
	Metrics []autoscalingv2.MetricSpec `json:"metrics,omitempty"`
}

// CronHorizontalPodAutoscalerScheduledPatch is a patch w/ schedule to apply.
type CronHorizontalPodAutoscalerScheduledPatch struct {
	// Name is the name of this schedule.
	// +kubebuilder:validation:MaxLength=16
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\-]+
	Name string `json:"name"`
	// Schedule is a schedule to apply the HPA in the cron format like `0 */2 * * *`.
	// See https://pkg.go.dev/github.com/robfig/cron
	Schedule string `json:"schedule"`
	// Timezone is a timezone of the schedule
	Timezone string `json:"timezone"`
	// Patch is a patch to apply to the template at the schedule.
	Patch *HPAPatch `json:"patch,omitempty"`
}

// CronHorizontalPodAutoscalerSpec defines the desired state of CronHorizontalPodAutoscaler
type CronHorizontalPodAutoscalerSpec struct {
	// Template is the template of HPA.
	Template HPATemplate `json:"template"`
	// schedules contain the specifications of HPA with a schedule.
	ScheduledPatches []CronHorizontalPodAutoscalerScheduledPatch `json:"scheduledPatches"`
}

// CronHorizontalPodAutoscalerStatus defines the observed state of CronHorizontalPodAutoscaler.
type CronHorizontalPodAutoscalerStatus struct {
	// LastCronTimestamp is the time of last cron job.
	LastCronTimestamp *metav1.Time `json:"lastCronTimestamp,omitempty"`
	// LastScheduledPatchName is the last patch name applied to the HPA.
	LastScheduledPatchName string `json:"lastScheduledPatchName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=cronhpa

// CronHorizontalPodAutoscaler is the Schema for the cronhorizontalpodautoscalers API.
type CronHorizontalPodAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronHorizontalPodAutoscalerSpec   `json:"spec,omitempty"`
	Status CronHorizontalPodAutoscalerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CronHorizontalPodAutoscalerList contains a list of CronHorizontalPodAutoscaler
type CronHorizontalPodAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronHorizontalPodAutoscaler `json:"items"`
}
