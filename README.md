## How it works

This tester produces loads from different clients against a server, configured in `config.yaml`.
 
- The server has an `initial_cpu_time` budget of CPU available to handle requests before they start getting rejected with 429.
- Clients can send different requests per second, configured under `client/rps`. This represents request rate in Little's Law.
- Different client requests, handled by the server, use different amounts of `cpu_time`. This represents latency in Little's Law.

## Fairness

Fairness can be enabled/disabled under the `server/fairness` setting. 

### Without Fairness

With fairness disabled, each request is checked against the current available CPU, which is initially set by the `initial_cpu_time`, and rejected if there is no available CPU. Since this approach rejects requests without fairness, you'll notice that tenants who are below their fair share of the `initial_cpu_time` (in terms of Little's Law) don't necessarily have 100% success rate.

### With Fairness

With fairness enabled, each request is checked first against some concurrency limit, which is controlled by a TCP Vegas algorithm, and then against the CPU limit, and can be rejected by either. If recent requests are getting rejected by the CPU limit, then the Vegas algorithm will lower the concurrency limit. If recent requests are succeeding with the CPU limiter, then the Vegas algorithm will increase the concurrency limit.

The concurrency limit is further divided by the number of tenants, so each tenant has their own limit, which provides decent fairness.

Presently, the Vegas algorhtm tends to set the concurrency limit too low, which is not allowing tenants to get their expected "fair" share of CPU. Further chagnes are needed to increase latency of request handling, rather than reject it outright, when the CPU is fully utilized.

## Dashboard

The `/dashboard` directory contains a Grafana dashboard that you can import. You'll need a prometheus scraping metrics from `localhost:8080`, which is exposed by the tester, added as a datasource in your Grafana.