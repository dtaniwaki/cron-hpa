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
	"reflect"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	cronhpav1alpha1 "github.com/ubie-oss/cron-hpa/api/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CronHorizontalPodAutoscaler cronhpav1alpha1.CronHorizontalPodAutoscaler

type CronHPAEvent = string

const annotationNameSkip = "cron-hpa.ubie-oss.github.com/skip"

const (
	CronHPAEventCreated     CronHPAEvent = "Created"
	CronHPAEventUpdated     CronHPAEvent = "Updated"
	CronHPAEventScheduled   CronHPAEvent = "Scheduled"
	CronHPAEventUnscheduled CronHPAEvent = "Unscheduled"
	CronHPAEventSkipped     CronHPAEvent = "Skipped"
	CronHPAEventNone        CronHPAEvent = ""
)

const MAX_SCHEDULE_TRY = 1000000

func (cronhpa *CronHorizontalPodAutoscaler) UpdateSchedules(ctx context.Context, reconciler *CronHorizontalPodAutoscalerReconciler) error {
	logger := log.FromContext(ctx)

	logger.Info("Update schedules")
	reconciler.Cron.RemoveResourceEntry(cronhpa.ToNamespacedName())
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
		logger.Info(fmt.Sprintf("Scheduled %s", scheduledPatch.Name))
	}
	msg := fmt.Sprintf("Scheduled: %s", strings.Join(entryNames, ","))
	reconciler.Recorder.Event((*cronhpav1alpha1.CronHorizontalPodAutoscaler)(cronhpa), corev1.EventTypeNormal, CronHPAEventScheduled, msg)
	return nil
}

func (cronhpa *CronHorizontalPodAutoscaler) ClearSchedules(ctx context.Context, reconciler *CronHorizontalPodAutoscalerReconciler) error {
	reconciler.Cron.RemoveResourceEntry(cronhpa.ToNamespacedName())
	msg := "Unscheduled"
	reconciler.Recorder.Event((*cronhpav1alpha1.CronHorizontalPodAutoscaler)(cronhpa), corev1.EventTypeNormal, CronHPAEventUnscheduled, msg)
	return nil
}

func (cronhpa *CronHorizontalPodAutoscaler) ApplyHPAPatch(patchName string, hpa *autoscalingv2.HorizontalPodAutoscaler) error {
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
			hpa.Spec.Metrics = make([]autoscalingv2.MetricSpec, len(scheduledPatch.Patch.Metrics))
			for i, metric := range scheduledPatch.Patch.Metrics {
				hpa.Spec.Metrics[i] = metric
			}
		}
	}
	return nil
}

func (cronhpa *CronHorizontalPodAutoscaler) NewHPA(patchName string) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	template := cronhpa.Spec.Template.DeepCopy()
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: autoscalingv2.SchemeGroupVersion.Identifier(),
		},
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

func (cronhpa *CronHorizontalPodAutoscaler) GetCurrentPatchName(ctx context.Context, currentTime time.Time) (string, error) {
	logger := log.FromContext(ctx)

	logger.Info("Get current patch")
	currentPatchName := ""
	for _, scheduledPatch := range cronhpa.Spec.ScheduledPatches {
		if scheduledPatch.Name == cronhpa.Status.LastScheduledPatchName {
			currentPatchName = scheduledPatch.Name
			break
		}
	}
	if cronhpa.Status.LastScheduledPatchName != "" && currentPatchName == "" {
		logger.Info(fmt.Sprintf("Lost scheduled patch %s", cronhpa.Status.LastScheduledPatchName))
	}
	lastCronTimestamp := cronhpa.Status.LastCronTimestamp
	if lastCronTimestamp != nil {
		standardParser := cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		)

		mostLatestTime := lastCronTimestamp.Time
		for _, scheduledPatch := range cronhpa.Spec.ScheduledPatches {
			tzs := scheduledPatch.Schedule
			if scheduledPatch.Timezone != "" {
				tzs = "CRON_TZ=" + scheduledPatch.Timezone + " " + scheduledPatch.Schedule
			}
			schedule, err := standardParser.Parse(tzs)
			if err != nil {
				return "", err
			}
			nextTime := lastCronTimestamp.Time
			latestTime := lastCronTimestamp.Time
			for i := 0; i <= MAX_SCHEDULE_TRY; i++ {
				nextTime = schedule.Next(nextTime)
				if nextTime.After(currentTime) || nextTime.IsZero() {
					break
				}
				latestTime = nextTime
				if i == MAX_SCHEDULE_TRY {
					return "", fmt.Errorf("Cannot find the next schedule of patch %s", scheduledPatch.Name)
				}
			}
			if latestTime.After(mostLatestTime) && (latestTime.Before(currentTime) || latestTime.Equal(currentTime)) {
				currentPatchName = scheduledPatch.Name
				mostLatestTime = latestTime
			}
		}

	}
	if cronhpa.Status.LastScheduledPatchName != currentPatchName {
		logger.Info(fmt.Sprintf("Current patch changed from %s to %s", cronhpa.Status.LastScheduledPatchName, currentPatchName))
	}
	return currentPatchName, nil
}

func (cronhpa *CronHorizontalPodAutoscaler) CreateOrPatchHPA(ctx context.Context, patchName string, currentTime time.Time, reconciler *CronHorizontalPodAutoscalerReconciler) error {
	logger := log.FromContext(ctx)

	logger.Info("Create or update HPA")

	newhpa, err := cronhpa.NewHPA(patchName)
	if err != nil {
		return err
	}
	if err := controllerutil.SetControllerReference(cronhpa.ToCompatible(), newhpa, reconciler.Client.Scheme()); err != nil {
		return err
	}

	event := ""
	msg := ""
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := reconciler.Get(ctx, cronhpa.ToNamespacedName(), hpa); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if err := reconciler.Create(ctx, newhpa); err != nil {
			return err
		}
		logger.Info("Created an HPA successfully")
		event = CronHPAEventCreated
		msg = "Created HPA"
	} else {
		if hpa.Annotations[annotationNameSkip] == "true" {
			logger.Info("Skip updating an HPA by an annotation")
			event = CronHPAEventSkipped
			msg = "Skipped updating HPA by an annotation"
		} else if reflect.DeepEqual(hpa.Spec, newhpa.Spec) {
			logger.Info("Skip updating an HPA with no changes")
			event = CronHPAEventSkipped
			msg = "Skipped updating HPA with no changes"
		} else {
			patch := client.MergeFrom(hpa)
			if err := reconciler.Patch(ctx, newhpa, patch); err != nil {
				return err
			}
			logger.Info("Updated an HPA successfully")
			event = CronHPAEventUpdated
			msg = "Updated HPA"
		}
	}

	if event != "" {
		cronhpa.Status.LastCronTimestamp = &metav1.Time{
			Time: currentTime,
		}
		cronhpa.Status.LastScheduledPatchName = patchName
		if err := reconciler.Status().Update(ctx, cronhpa.ToCompatible()); err != nil {
			return err
		}
		if patchName != "" {
			msg = fmt.Sprintf("%s with %s", msg, patchName)
		}
		reconciler.Recorder.Event(cronhpa.ToCompatible(), corev1.EventTypeNormal, event, msg)
	}

	return nil
}

func (cronhpa *CronHorizontalPodAutoscaler) ToCompatible() *cronhpav1alpha1.CronHorizontalPodAutoscaler {
	return (*cronhpav1alpha1.CronHorizontalPodAutoscaler)(cronhpa)
}

func (cronhpa *CronHorizontalPodAutoscaler) ToNamespacedName() types.NamespacedName {
	return types.NamespacedName{Namespace: cronhpa.Namespace, Name: cronhpa.Name}
}
