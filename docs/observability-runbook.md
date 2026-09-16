# Observability Runbook

## Scope

WGA exports vendor-neutral OpenTelemetry traces and metrics to a private Collector Contrib service. The collector owns sampling, buffering, retries, and final-backend authentication. WGA remains available when telemetry cannot be exported.

## Railway collector service

Create one service named `otel-collector` from the WGA repository with its root directory set to `/deploy/otel-collector`. The directory contains the pinned collector image, configuration, and Railway health-check settings. Keep the service at one replica so every span in a trace reaches the same tail-sampling decision point.

Do not generate a public Railway domain for the collector. WGA sends OTLP/gRPC to port `4317` over Railway private networking. Railway's service health check uses `/status` on the collector's injected `PORT`; a healthy deployment is the operational readiness signal.

Set a memory limit of at least 512 MiB and a CPU limit appropriate for expected ingestion, initially 1 vCPU. The collector's `memory_limiter` starts shedding telemetry around 384 MiB and allows a 96 MiB spike. Do not set the Railway memory limit below 512 MiB without first lowering both collector limits and validating the configuration.

Configure these variables on the collector service:

| Variable                             | Purpose                                                                                        |
| ------------------------------------ | ---------------------------------------------------------------------------------------------- |
| `OTEL_BACKEND_OTLP_ENDPOINT`         | Final backend's OTLP/gRPC host and port.                                                       |
| `OTEL_BACKEND_AUTHORIZATION`         | Complete backend authorisation header value, such as `Bearer …`; store it as a Railway secret. |
| `OTEL_TAIL_SAMPLING_SLOW_REQUEST_MS` | Optional slow-trace threshold; defaults to `2000`.                                             |
| `OTEL_TAIL_SAMPLING_PERCENTAGE`      | Optional percentage of otherwise healthy traces retained; defaults to `5`.                     |

Backend credentials belong only on `otel-collector`. Do not add them to WGA variables, committed configuration, telemetry attributes, or operator logs.

## WGA private endpoint

Set this variable on WGA in the same Railway environment:

```text
OTEL_EXPORTER_OTLP_ENDPOINT=http://${{otel-collector.RAILWAY_PRIVATE_DOMAIN}}:4317
```

This Railway reference variable follows the collector's private domain without hard-coding it. Private-network traffic is encrypted by Railway, so the validated WGA endpoint may use `http` for the `railway.internal` address. Do not create or reference a public collector domain.

Endpoint presence is the only application-side enablement control. WGA has no telemetry enable flag and must not receive final-backend credentials.

## Rollout and verification

1. Deploy `otel-collector` in UAT with the final-backend variables and the documented resource limits. Confirm Railway reports its `/status` health check as healthy and that the collector has no repeated export or memory-limiter errors.
2. Leave `OTEL_EXPORTER_OTLP_ENDPOINT` absent from WGA and deploy the application changes. Confirm requests still succeed and the backend receives no WGA telemetry.
3. Add the private endpoint reference to UAT WGA and redeploy it. Issue one request that cold-loads a collection-holdings or artist-availability projection, then repeat the same request to produce a cache hit.
4. Confirm the backend receives a matched-route request span, one stable repository-operation span for the cold load, cache request outcomes for `miss` then `hit`, and operation/cache duration metrics. Confirm spans and metric dimensions contain no raw route values, search terms, SQL, record identifiers, client addresses, payloads, or arbitrary error text.
5. Exercise a controlled request-protection rejection and confirm aggregate `429` or `503` admission metrics. Confirm error, `503`, and slow traces are retained while ordinary successful traces follow probabilistic sampling. Metrics must remain present independently of trace sampling.
6. Observe UAT volume, collector memory, queue pressure, export failures, and healthy-trace retention. Tune only the two documented sampling variables before production rollout.
7. Deploy the collector to production first. After it is healthy, add the same private endpoint reference to production WGA and verify the controlled signals again.

Collector or backend unavailability must not change WGA responses. If queues fill, bounded telemetry loss is preferable to application back-pressure.

## Rollback

Remove `OTEL_EXPORTER_OTLP_ENDPOINT` from the affected WGA environment and redeploy WGA. This disables exporters and telemetry middleware without changing request behaviour. Verify normal requests still succeed and new WGA telemetry stops arriving. The collector service and backend variables can then be rolled back or removed independently.
