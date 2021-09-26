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

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/dtaniwaki/cron-hpa/api/v1alpha1"
	cronhpav1alpha1 "github.com/dtaniwaki/cron-hpa/api/v1alpha1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CronHorizontalPodAutoscaler cronhpav1alpha1.CronHorizontalPodAutoscaler

type CronHPAEvent = string

const (
	CronHPAEventCreated     CronHPAEvent = "Created"
	CronHPAEventUpdated     CronHPAEvent = "Updated"
	CronHPAEventScheduled   CronHPAEvent = "Scheduled"
	CronHPAEventUnscheduled CronHPAEvent = "Unscheduled"
	CronHPAEventNone        CronHPAEvent = ""
)

func (cronhpa *CronHorizontalPodAutoscaler) UpdateSchedules(ctx context.Context, reconciler *CronHorizontalPodAutoscalerReconciler) error {
	logger := log.FromContext(ctx)
	reconciler.Cron.RemoveResourceEntries(cronhpa.ToNamespacedName())
	entryNames := make([]string, 0)
	for _, scheduledPatch := range cronhpa.Spec.ScheduledPatches {
		entryNames = append(entryNames, scheduledPatch.Name)
		tzs := scheduledPatch.Schedule
		if scheduledPatch.Timezone != "" {
			tzs = "CRON_TZ=" + scheduledPatch.Timezone + " " + scheduledPatch.Schedule
		}
		err := reconciler.Cron.Add(cronhpa.ToNamespacedName(), scheduledPatch.Name, tzs, &CronContext{
			reconciler: reconciler,
			cronhpa:    cronhpa,
			patchName:  scheduledPatch.Name,
		})
		if err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("Scheduled %s of CronHPA %s in %s", scheduledPatch.Name, cronhpa.Name, cronhpa.Namespace))
	}
	msg := fmt.Sprintf("Scheduled: %s", strings.Join(entryNames, ","))
	reconciler.Recorder.Event((*cronhpav1alpha1.CronHorizontalPodAutoscaler)(cronhpa), corev1.EventTypeNormal, CronHPAEventScheduled, msg)
	return nil
}

func (cronhpa *CronHorizontalPodAutoscaler) ClearSchedules(ctx context.Context, reconciler *CronHorizontalPodAutoscalerReconciler) error {
	reconciler.Cron.RemoveResourceEntries(cronhpa.ToNamespacedName())
	msg := "Unscheduled"
	reconciler.Recorder.Event((*cronhpav1alpha1.CronHorizontalPodAutoscaler)(cronhpa), corev1.EventTypeNormal, CronHPAEventUnscheduled, msg)
	return nil
}

func (cronhpa *CronHorizontalPodAutoscaler) ApplyHPAPatch(patchName string, hpa *autoscalingv2beta2.HorizontalPodAutoscaler) error {
	var scheduledPatch *cronhpav1alpha1.CronHorizontalPodAutoscalerScheduledPatch
	for _, sp := range cronhpa.Spec.ScheduledPatches {
		if sp.Name == patchName {
			scheduledPatch = &sp
			break
		}
	}
	if scheduledPatch == nil {
		return fmt.Errorf("No schedule patch named %s", patchName)
	}

	// Apply patches on the template.
	if scheduledPatch.Patch != nil {
		if scheduledPatch.Patch.MinReplicas != nil {
			*hpa.Spec.MinReplicas = *scheduledPatch.Patch.MinReplicas
		}
		if scheduledPatch.Patch.MaxReplicas != nil {
			hpa.Spec.MaxReplicas = *scheduledPatch.Patch.MaxReplicas
		}
		if scheduledPatch.Patch.Metrics != nil {
			hpa.Spec.Metrics = make([]autoscalingv2beta2.MetricSpec, len(scheduledPatch.Patch.Metrics))
			for i, metric := range scheduledPatch.Patch.Metrics {
				hpa.Spec.Metrics[i] = metric
			}
		}
	}
	return nil
}

func (cronhpa *CronHorizontalPodAutoscaler) NewHPA(patchName string) (*autoscalingv2beta2.HorizontalPodAutoscaler, error) {
	template := cronhpa.Spec.Template.DeepCopy()
	hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronhpa.Name,
			Namespace: cronhpa.Namespace,
		},
		Spec: template.Spec,
	}
	if template.Metadata != nil {
		hpa.ObjectMeta.Labels = template.Metadata.Labels
		hpa.ObjectMeta.Annotations = template.Metadata.Annotations
	}
	if patchName != "" {
		if err := cronhpa.ApplyHPAPatch(patchName, hpa); err != nil {
			return nil, err
		}
	}
	return hpa, nil
}

func (cronhpa *CronHorizontalPodAutoscaler) CreateOrPatchHPA(ctx context.Context, patchName string, reconciler *CronHorizontalPodAutoscalerReconciler) error {
	logger := log.FromContext(ctx)
	hpa, err := cronhpa.NewHPA(patchName)
	if err != nil {
		return err
	}
	op, err := controllerutil.CreateOrPatch(ctx, reconciler.Client, hpa, func() error {
		if err := controllerutil.SetControllerReference(cronhpa.ToCompatible(), hpa, reconciler.Client.Scheme()); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	event := ""
	msg := ""
	if op == controllerutil.OperationResultCreated {
		logger.Info(fmt.Sprintf("Created an HPA successfully: %s in %s", cronhpa.Name, cronhpa.Namespace))
		event = CronHPAEventCreated
		msg = fmt.Sprintf("Created HPA %s", hpa.Name)
	} else if op == controllerutil.OperationResultUpdated {
		logger.Info(fmt.Sprintf("Updated an HPA successfully: %s in %s", cronhpa.Name, cronhpa.Namespace))
		event = CronHPAEventUpdated
		msg = fmt.Sprintf("Updated HPA %s", hpa.Name)
	} else {
		logger.Info(fmt.Sprintf("Updated an HPA without changes: %s in %s", cronhpa.Name, cronhpa.Namespace))
		event = CronHPAEventUpdated
		msg = fmt.Sprintf("Updated HPA %s without changes", hpa.Name)
	}
	if msg != "" && patchName != "" {
		msg = fmt.Sprintf("%s with %s", msg, patchName)
	}
	reconciler.Recorder.Event((*cronhpav1alpha1.CronHorizontalPodAutoscaler)(cronhpa), corev1.EventTypeNormal, event, msg)
	return nil
}

func (cronhpa *CronHorizontalPodAutoscaler) ToCompatible() *v1alpha1.CronHorizontalPodAutoscaler {
	return (*v1alpha1.CronHorizontalPodAutoscaler)(cronhpa)
}

func (cronhpa *CronHorizontalPodAutoscaler) ToNamespacedName() types.NamespacedName {
	return types.NamespacedName{Namespace: cronhpa.Namespace, Name: cronhpa.Name}
}
