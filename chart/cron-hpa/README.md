# CronHPA Chart

## Deployment

First, deploy the CRD.

```bash
$ kubectl apply -f crds
```

Then, deploy the resources by Helm.

```bash
$ helm install test-cron-hpa .
```
