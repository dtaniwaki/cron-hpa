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
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cronhpav1alpha1 "github.com/ubie-oss/cron-hpa/api/v1alpha1"
)

// CronHorizontalPodAutoscalerReconciler reconciles a CronHorizontalPodAutoscaler object
type CronHorizontalPodAutoscalerReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Cron     *Cron
}

const finalizerName = "cron-hpa.ubie-oss.github.com/finalizer"

//+kubebuilder:rbac:groups=cron-hpa.ubie-oss.github.com,resources=cronhorizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cron-hpa.ubie-oss.github.com,resources=cronhorizontalpodautoscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cron-hpa.ubie-oss.github.com,resources=cronhorizontalpodautoscalers/finalizers,verbs=update
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *CronHorizontalPodAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	now := time.Now()

	// Fetch the CronHorizontalPodAutoscaler instance.
	logger.Info("Fetch CronHPA")
	cronhpa := &CronHorizontalPodAutoscaler{}
	err := r.Get(ctx, req.NamespacedName, cronhpa.ToCompatible())
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deleted resources.
	if !cronhpa.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(cronhpa.ToCompatible(), finalizerName) {
			logger.Info("Clear schedules")
			if err := cronhpa.ClearSchedules(ctx, r); err != nil {
				logger.Error(err, "Failed to clear schedules")
			}

			controllerutil.RemoveFinalizer(cronhpa.ToCompatible(), finalizerName)
			if err := r.Update(ctx, cronhpa.ToCompatible()); err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Set finalizer.
	if !controllerutil.ContainsFinalizer(cronhpa.ToCompatible(), finalizerName) {
		logger.Info("Set finalizer")
		cronhpa.ObjectMeta.Finalizers = append(cronhpa.ObjectMeta.Finalizers, finalizerName)
		if err := r.Update(ctx, cronhpa.ToCompatible()); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Fetch the corresponded HPA instance.
	logger.Info("Fetch HPA")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := r.Get(ctx, req.NamespacedName, hpa); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Create or update HPA")
	patchName, err := cronhpa.GetCurrentPatchName(ctx, now)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := cronhpa.CreateOrPatchHPA(ctx, patchName, now, r); err != nil {
		return ctrl.Result{}, err
	}

	// Update the schedules.
	logger.Info("Update schedules")
	if err := cronhpa.UpdateSchedules(ctx, r); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronHorizontalPodAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cronhpav1alpha1.CronHorizontalPodAutoscaler{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Complete(r)
}
