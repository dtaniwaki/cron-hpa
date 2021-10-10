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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
)

func TestCron(t *testing.T) {
	cron := NewCron()
	err := cron.Add(types.NamespacedName{Name: "foo", Namespace: "ns"}, "patch-1", "0 * * * *", &CronContext{
		reconciler: nil,
		cronhpa:    nil,
		patchName:  "patch-1",
	})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = cron.Add(types.NamespacedName{Name: "foo", Namespace: "ns"}, "patch-2", "0 * * * *", &CronContext{
		reconciler: nil,
		cronhpa:    nil,
		patchName:  "patch-2",
	})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = cron.Add(types.NamespacedName{Name: "foo", Namespace: "ns"}, "patch-3", "0 * * * *", &CronContext{
		reconciler: nil,
		cronhpa:    nil,
		patchName:  "patch-3",
	})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = cron.Add(types.NamespacedName{Name: "bar", Namespace: "ns"}, "patch-4", "0 * * * *", &CronContext{
		reconciler: nil,
		cronhpa:    nil,
		patchName:  "patch-4",
	})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	resourceEntry := cron.ListResourceEntry(types.NamespacedName{Name: "foo", Namespace: "ns"})
	assert.Equal(t, ResourceEntry{
		"patch-1": 1,
		"patch-2": 2,
		"patch-3": 3,
	}, resourceEntry)

	cron.Remove(types.NamespacedName{Name: "foo", Namespace: "ns"}, "patch-1")
	resourceEntry = cron.ListResourceEntry(types.NamespacedName{Name: "foo", Namespace: "ns"})
	assert.Equal(t, ResourceEntry{
		"patch-2": 2,
		"patch-3": 3,
	}, resourceEntry)
	cron.Remove(types.NamespacedName{Name: "foo", Namespace: "ns"}, "paptch-1")

	cron.RemoveResourceEntry(types.NamespacedName{Name: "foo", Namespace: "ns"})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	resourceEntry = cron.ListResourceEntry(types.NamespacedName{Name: "foo", Namespace: "ns"})
	assert.Equal(t, ResourceEntry{}, resourceEntry)
}
