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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CronContext struct {
	reconciler *CronHorizontalPodAutoscalerReconciler
	cronhpa    *CronHorizontalPodAutoscaler
	patchName  string
}

func (cronctx *CronContext) Run() {
	ctx := context.Background()
	logger := log.Log
	if err := cronctx.run(ctx); err != nil {
		logger.Error(err, "Failed to run a cron job")
	}
}

func (cronctx *CronContext) run(ctx context.Context) error {
	logger := log.FromContext(ctx)
	cronhpa := cronctx.cronhpa
	now := time.Now()

	logger.Info(fmt.Sprintf("Execute a cron job of CronHPA %s in %s", cronhpa.Name, cronhpa.Namespace))

	err := cronctx.reconciler.Get(ctx, cronhpa.ToNamespacedName(), cronhpa.ToCompatible())
	if err != nil {
		if errors.IsNotFound(err) {
			// Remove the lost cron.
			cronctx.reconciler.Cron.RemoveResourceEntries(cronhpa.ToNamespacedName())
			return nil
		}
		return err
	}

	if err := cronhpa.CreateOrPatchHPA(ctx, cronctx.patchName, now, cronctx.reconciler); err != nil {
		return err
	}

	return nil
}
