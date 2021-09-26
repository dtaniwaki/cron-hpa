# CronHPA

CronHPA is an operator to update HPA resources based on schedules.

## Deployment

Build and load the Docker image to your kind cluster.

```bash
$ make kind-load
```

Install the CRD to the cluster.

```bash
$ make install
```

Deploy a controller to the cluster.

```bash
$ make deploy
```
