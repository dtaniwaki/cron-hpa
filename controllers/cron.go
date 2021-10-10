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
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/types"
)

type ResourceEntry map[string]cron.EntryID

type Cron struct {
	cron            *cron.Cron
	resourceEntries map[string]ResourceEntry
	lock            sync.RWMutex
}

func NewCron() *Cron {
	return &Cron{
		cron:            cron.New(),
		resourceEntries: make(map[string]ResourceEntry),
		lock:            sync.RWMutex{},
	}
}

func (c *Cron) Start() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cron.Start()
}

func (c *Cron) Stop() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cron.Stop()
}

func (c *Cron) Add(namespacedName types.NamespacedName, patchName, tzs string, job cron.Job) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	resourceName := getResourceEntryName(namespacedName)
	resourceEntry, ok := c.resourceEntries[resourceName]
	if !ok {
		resourceEntry = make(map[string]cron.EntryID)
		c.resourceEntries[resourceName] = resourceEntry
	}
	entryID, err := c.cron.AddJob(tzs, job)
	if err != nil {
		return err
	}
	resourceEntry[patchName] = entryID
	return nil
}

func (c *Cron) Remove(namespacedName types.NamespacedName, patchName string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	resourceName := getResourceEntryName(namespacedName)
	resourceEntry, ok := c.resourceEntries[resourceName]
	if ok && resourceEntry != nil {
		entryID, eok := resourceEntry[patchName]
		if eok {
			c.cron.Remove(entryID)
		}
		delete(resourceEntry, patchName)
	}
}

func (c *Cron) RemoveResourceEntry(namespacedName types.NamespacedName) {
	c.lock.Lock()
	defer c.lock.Unlock()

	resourceName := getResourceEntryName(namespacedName)
	resourceEntry, ok := c.resourceEntries[resourceName]
	if ok && resourceEntry != nil {
		for _, entryID := range resourceEntry {
			c.cron.Remove(entryID)
		}
		delete(c.resourceEntries, resourceName)
	}
}

func (c *Cron) ListResourceEntry(namespacedName types.NamespacedName) ResourceEntry {
	c.lock.Lock()
	defer c.lock.Unlock()

	resourceName := getResourceEntryName(namespacedName)
	resourceEntry, ok := c.resourceEntries[resourceName]
	if ok && resourceEntry != nil {
		return resourceEntry
	}
	return ResourceEntry{}
}

func getResourceEntryName(namespacedName types.NamespacedName) string {
	return fmt.Sprintf("%s/%s", namespacedName.Namespace, namespacedName.Name)
}
