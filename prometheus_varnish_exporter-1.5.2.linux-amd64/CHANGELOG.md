1.5.2
=====
* Fix metric names and missing labels for file cache metrics ([#55](https://github.com/jonnenauha/prometheus_varnish_exporter/pull/55) @thedustin)
* Fix scraping for Varnish 3.x. Removes the `main_n_ban` grouping. Metrics will now have individual `bans_<type>` metrics instad of the grouped metric that had `type` as a label. ([#51](https://github.com/jonnenauha/prometheus_varnish_exporter/pull/51) @glennslaven)
    * If you previously updated to 1.5 your exports would have already been broken as the grouping tries to combine gauge and counter metrics, which is not allowed by Prometheus.
    * This is breaking change if you are using Varnish 3.x and use ban metrics in your dashboards, you'll need to update them to the new ones.
* Clean exported backend name if beginning with reload_ ([#56](https://github.com/jonnenauha/prometheus_varnish_exporter/pull/56) @stromnet)

1.5.1
=====
* Fix incorrectly typing Varnish 4.0.x stat flag `a` metrics as gauges instead of counters. ([#48](https://github.com/jonnenauha/prometheus_varnish_exporter/pull/48) @glennslaven)
* Fix `-test` mode to wait for full metrics scrape before continuing.

1.5
===
* Deprecate `-no-exit`. Default behavior is now not to exit on scrape errors as it should be for a long running HTTP server.
  * This was design misstep. You will now get a deprecation warning if you pass `-no-exit` but the process behaves as before.
  * New explicit `-exit-on-errors` has been added for users who want the old default behavior back.
* Correctly export gauge and counter types from `varnishstat` output `type` property.
* Add go module support.
* Use `github.com/prometheus/client_golang` v1.0.0
* Start building releases with go 1.12.6

1.4.1
=====
* `-docker-container-name` to signal that `varnishstat` should be ran in a docker container with `docker exec <container-name>` .
* Support Varnish 6.0.0 by testing the main logic works and metrics are exported.
* Start building releases with go 1.10.3

1.4
===
* Standard non Varnish prometheus metrics need to now be enabled with `-with-go-metrics`. Before they were included by default. Now dropped to export less clutter that majority of users will never need (@nipuntalukdar).
* Fix `varnish_backend_up` with Varnish 4.0 and earlier versions.

1.3.4
=====
* New per backend metric `varnish_backend_up` with 1/0 value that reflects the latest health probe result. The Varnish bitmap uint64 `varnish_backend_happy` as a prometheus float metric was not that useful in detecting latest up/down per backend.
* Ability to give custom path to varnishstat with `-varnishstat-path` (@zstyblik)
* Github releases now include Grafana dashboards archive. This includes all the dashboards posted by users in the repo, starting with my own.

1.3.3
=====
* New `-no-exit` mode that does not exit the process if varnish is not running at the time of startup.
* Support Varnish 5.2 [that removed](http://varnish-cache.org/docs/5.2/whats-new/upgrading-5.2.html#other-changes) `type` and `ident` properties from varnishstat JSON output. If `ident` is not present, it is now parsed from the metric name.
* Add tests to run scrape on static json files.
* Start building releases with go 1.9.1

1.3.2
=====
* Update readme to mention that exporter has been tested to work against Varnish 5.x releases.
* Start building releases with go 1.9

1.3.1
=====

* Don't return a 400 for `/` to behave more like other Prometheus exporters out there. Can now be used for health checks. ([#15](https://github.com/jonnenauha/prometheus_varnish_exporter/pull/15))
* Start building releases with go 1.8

1.3
===
* Release packages now use the same naming and internal structure scheme with [promu](https://github.com/prometheus/promu).
  * Fixes issues running this exporter with systems like [puppet-prometheus](https://github.com/voxpupuli/puppet-prometheus)
* No code changes
* Start building releases with go 1.7.5

1.2
===
* Fix VBE label inconsistencies by always having `backend` and `server` labels present. ([#5](https://github.com/jonnenauha/prometheus_varnish_exporter/issues/5) [#8](https://github.com/jonnenauha/prometheus_varnish_exporter/issues/8))
 * Resulted in varnish reporting lots of errors for a while after VCL reloads.
* Fix bugs in `backend` and `server` label value parsing from VBE ident. ([#5](https://github.com/jonnenauha/prometheus_varnish_exporter/issues/5) [#8](https://github.com/jonnenauha/prometheus_varnish_exporter/issues/8))
* Add travis-ci build and test integration. Also auto pushes cross compiled binaries to github releases on tags.

1.1
===
* `-web.health-path <path>` can be configured to return a 200 OK response, by default not enabled. [#6](https://github.com/jonnenauha/prometheus_varnish_exporter/pull/6)
* Start building releases with go 1.7.3

1.0
===
* First official release
* Start building releases with go 1.7.1
