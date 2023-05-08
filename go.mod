module github.com/dtaniwaki/cron-hpa

go 1.16

require (
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.27.4
	github.com/robfig/cron/v3 v3.0.0
	github.com/stretchr/testify v1.8.1
	k8s.io/api v0.27.1
	k8s.io/apimachinery v0.27.1
	k8s.io/client-go v0.27.1
	sigs.k8s.io/controller-runtime v0.14.6
	sigs.k8s.io/yaml v1.3.0
)
