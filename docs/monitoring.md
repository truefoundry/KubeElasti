## Monitoring

Set `.global.enableMonitoring` to `true` in the [values.yaml](https://github.com/truefoundry/elasti/blob/main/charts/elasti/values.yaml) file to enable monitoring.

This will create two ServiceMonitor custom resources to enable Prometheus to discover the Elasti components. To verify this, you can open your Prometheus interface and search for metrics prefixed with `elasti_`, or navigate to the Targets section to check if Elasti is listed.

Once verification is complete, you can use the [provided Grafana dashboard](https://github.com/truefoundry/elasti/blob/main/playground/infra/elasti-dashboard.yaml) to monitor the internal metrics and performance of Elasti.


<figure markdown="span">
  ![Image title](../images/grafana-dashboard.png){ loading=lazy }
  <figcaption>Grafana dashboard</figcaption>
</figure>