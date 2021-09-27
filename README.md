# CronHPA

CronHPA is an operator to update HPA resources based on schedules. For example, you can decrease min replicas in the night-time and increase it in the day-time.

## Build

Build and load the Docker image to your cluster.

```bash
$ make docker-build

# run a command to load the image to your cluster.
```

If you use a kind cluster, there's a useful shortcut.

```
$ make kind-load
```

## Deployment

Install the CRD to the cluster.

```bash
$ make install
```

Deploy a controller to the cluster.

```bash
$ make deploy
```

## Usage

Now, deploy the samples.

```bash
$ make deploy-samples
```

You will see sample HPA and deployment in the current context, maybe `default` depending on your env. The HPA resource gets updated periodically by the CronHPA.

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new [Pull Request](../../pull/new/master)

## Copyright

Copyright (c) 2021 Daisuke Taniwaki. See [LICENSE](LICENSE) for details.

