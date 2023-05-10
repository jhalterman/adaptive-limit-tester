## Overview

This tester produces loads from different clients against a server, and throttles load using CPU limits (simulating a loaded machine) and/or adaptive concurrency limits, which try to adapt to changes in load.

## How it works

The available CPU, client RPS, and request latency, are all configured in `config.yaml`:
 
- The server has an `available_cpu_time` budget of CPU available to handle requests before limits start to kick in.
- Different clients can be configured with individual `rps` and `cpu_time` (latency) settings.

## Adaptive Limiting

Adaptive limiting can be enabled/disabled under the `server/adaptive_limiting` setting. 

### Without Adaptive Limiting

With adaptive limiting disabled, each client request's `cpu_time` is checked against the current available CPU, which is initially set by the `available_cpu_time `, and rejected if there is no available CPU. Since this approach rejects requests without fairness, you'll notice that clients don't necessarily get their fair share of resources when clients are configured to use more than the available concurrency (in terms of Little's Law).

### With Adaptive Limiting

With adaptive limiting enabled, each request is checked against some concurrency limit, which is controlled by a TCP Vegas algorithm, and rejected if the client is above their limit. The limit adjusts up or down based on recent min latencies. When the CPU is fully utilized, the CPU limiter increases latency based on how far over the CPU limit we are. This will cause the Vegas algorithm to adjust concurrency limits down.

### Fairness

When Adaptive Limiting is enabled, the overall concurrency limit is divided by the number of clients, so each client has their own limit. This ensures that no client is utilizing more than their share of resources. When concurrency is being throttled, you'll notice that clients get close to their expected fair share of resources, even with a noisy neighbor.

## Dashboard

The `/dashboard` directory contains a Grafana dashboard that you can import, which includes:

- Success rate per client
- Responses per second: 200=success, 429=cpu limited, 430=concurrency limited
- Average RTT
- Concurrency limits per client
- Concurrency usage per client
- Concurrent requests
- CPU time used

You'll need a prometheus scraping metrics from `localhost:8080`, which is exposed by the tester, added as a datasource in your Grafana.