package test

import (
	"context"
	"path/filepath"

	cronhpav1alpha1 "github.com/ubie-oss/cron-hpa/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func NewFakeClient(ctx context.Context) (client.Client, error) {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := testEnv.Start()
	if err != nil {
		return nil, err
	}
	scheme := runtime.NewScheme()
	err = clientgoscheme.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = cronhpav1alpha1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return k8sClient, nil
}
