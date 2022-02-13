package test

import "k8s.io/apimachinery/pkg/runtime"

type FakeRecorder struct{}

func (m *FakeRecorder) Event(object runtime.Object, eventtype, reason, message string) {}
func (m *FakeRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
}
func (m *FakeRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
}
