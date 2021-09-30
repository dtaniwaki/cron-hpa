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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestNewHPA(t *testing.T) {
	cronHPAManifest := `
apiVersion: cron-hpa.dtaniwaki.github.com/v1alpha1
kind: CronHorizontalPodAutoscaler
metadata:
  name: cron-hpa-sample
  namespace: default
spec:
  template:
    spec:
      scaleTargetRef:
        apiVersion: apps/v1
        kind: Deployment
        name: cron-hpa-nginx
      minReplicas: 1
      maxReplicas: 10
      metrics:
      - type: Resource
        resource:
          name: cpu
          target:
            type: Utilization
            averageUtilization: 50
  scheduledPatches:
  - name: one
    schedule: "0,10,20,30,40,50 * * * *"
    timezone: "Asia/Tokyo"
    patch:
      minReplicas: 3
      maxReplicas: 15
      metrics:
      - type: Resource
        resource:
          name: cpu
          target:
            type: Utilization
            averageUtilization: 30
`

	cronhpa := &CronHorizontalPodAutoscaler{}
	err := yaml.Unmarshal([]byte(cronHPAManifest), cronhpa.ToCompatible())
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	defaultHPAManifest := `
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: cron-hpa-sample
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: cron-hpa-nginx
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
`

	defaultHPA := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	err = yaml.Unmarshal([]byte(defaultHPAManifest), defaultHPA)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	hpa, err := cronhpa.NewHPA("")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	fmt.Printf("kind %s %s\n", defaultHPA.Kind, hpa.Kind)
	fmt.Printf("kind %s %s\n", defaultHPA.TypeMeta.Kind, hpa.TypeMeta.Kind)
	if !assert.Equal(t, defaultHPA, hpa) {
		t.FailNow()
	}

	withPatchHPAManifest := `
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: cron-hpa-sample
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: cron-hpa-nginx
  minReplicas: 3
  maxReplicas: 15
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 30
`

	withPatchHPA := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	err = yaml.Unmarshal([]byte(withPatchHPAManifest), withPatchHPA)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	hpa, err = cronhpa.NewHPA("one")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, withPatchHPA, hpa) {
		t.FailNow()
	}
}

func TestGetCurrentPatchName(t *testing.T) {
	ctx := context.TODO()

	cronHPAManifest := `
apiVersion: cron-hpa.dtaniwaki.github.com/v1alpha1
kind: CronHorizontalPodAutoscaler
metadata:
  name: cron-hpa-sample
  namespace: default
spec:
  template:
    spec:
      scaleTargetRef:
        apiVersion: apps/v1
        kind: Deployment
        name: cron-hpa-nginx
      minReplicas: 1
      maxReplicas: 10
      metrics:
      - type: Resource
        resource:
          name: cpu
          target:
            type: Utilization
            averageUtilization: 50
  scheduledPatches:
  - name: weekday
    schedule: "0 0 * 10 mon-fri"
    timezone: "Asia/Tokyo"
  - name: weekend
    schedule: "0 0 * 10 sat,sun"
    timezone: "Asia/Tokyo"
`

	cronhpa := &CronHorizontalPodAutoscaler{}
	err := yaml.Unmarshal([]byte(cronHPAManifest), cronhpa.ToCompatible())
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	currentTime := time.Time{}
	_ = currentTime.UnmarshalText([]byte("2021-09-04T00:00:00+09:00"))
	cronhpa.Status.LastCronTimestamp = &metav1.Time{
		Time: currentTime,
	}

	// In-range weekday.
	_ = currentTime.UnmarshalText([]byte("2021-10-04T00:00:00+09:00")) // Mon
	patchName, err := cronhpa.GetCurrentPatchName(ctx, currentTime)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, "weekday", patchName) {
		t.FailNow()
	}

	// In-range weekend.
	_ = currentTime.UnmarshalText([]byte("2021-10-02T00:00:00+09:00")) // Sat
	patchName, err = cronhpa.GetCurrentPatchName(ctx, currentTime)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, "weekend", patchName) {
		t.FailNow()
	}

	// Out-range date
	_ = currentTime.UnmarshalText([]byte("2021-09-15T00:00:00+09:00")) // Wed
	patchName, err = cronhpa.GetCurrentPatchName(ctx, currentTime)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, "", patchName) {
		t.FailNow()
	}

	// Out-range date with last patch name.
	_ = currentTime.UnmarshalText([]byte("2021-09-15T00:00:00+09:00")) // Wed
	cronhpa.Status.LastScheduledPatchName = "weekday"
	patchName, err = cronhpa.GetCurrentPatchName(ctx, currentTime)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, "weekday", patchName) {
		t.FailNow()
	}
	cronhpa.Status.LastScheduledPatchName = ""

	// Without last timestamp
	cronhpa.Status.LastCronTimestamp = nil
	_ = currentTime.UnmarshalText([]byte("2021-10-02T00:00:00+09:00")) // Sat
	patchName, err = cronhpa.GetCurrentPatchName(ctx, currentTime)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, "", patchName) {
		t.FailNow()
	}
}
