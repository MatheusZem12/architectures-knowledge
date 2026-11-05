# 13. Serverless

> You write functions, and a cloud provider runs them on demand. No server of yours is running: not even an idle one.

There are still servers, of course. They are just not your problem. The best-known examples are AWS Lambda, Google Cloud
Functions and Azure Functions.

**An honest limit of this folder:** it does not run in a cloud. `cloud/` is a small local emulator of what a provider sells,
so that everything runs on your machine with no account or cost. The structure of the code is real. The operational
side (real cold starts, real scaling, the bill) can only be described here, not measured.

## Structure

```text
13-serverless/
├── main.go                     ← the deployment table, like a serverless.yml
├── architecture_test.go        ← functions must be stateless and independent
├── cloud/                      ← THE EMULATOR: in production, this is not your code
│   ├── runtime.go              ← runs functions, API gateway, queue
│   └── kv.go                   ← managed database, like DynamoDB
└── functions/                  ← YOUR CODE
    ├── shared/                 ← common types, like a Lambda layer
    ├── createproduct/handler.go
    ├── placeorder/handler.go
    ├── payorder/handler.go
    ├── sendnotification/handler.go  ← triggered by the queue, not by HTTP
    └── ...
```

## The flow

```text
                 ┌─ API gateway ──► placeorder (starts on demand)
  client ────────┤                     │  │
                 │                     │  └──► managed database
                 │                     ▼
                 │                  queue ──► sendnotification (starts on demand)
                 └─ no process of yours is running in between
```

## What it looks like in practice

**The application is a table.** There is no `http.ListenAndServe` of yours in spirit: only which function answers what:

```go
provider.Route("POST /orders", "placeorder", placeorder.Handler)
provider.Route("POST /orders/{id}/pay", "payorder", payorder.Handler)
provider.Trigger("order-events", "sendnotification", sendnotification.Handler)
```

**A function is just a call.** It has no `net/http`, and no state of its own:

```go
func Handler(env cloud.Env, req cloud.Request) cloud.Response {
```

**Nothing survives between calls.** An instance can vanish at any moment, so anything kept in a variable may be lost. This is
the central constraint of the architecture, and here it is a test that breaks the build:

```go
t.Errorf("%s: a package-level variable is state that would be lost with the instance", path)
```

**All-or-nothing comes from the managed database:**

```go
// R05, R06 and R07: a failure anywhere in this transaction discards every write in it
```

## Cold start: the famous downside

A function with no running instance must start one first. This is simulated with `COLD_START_MS=200`. Real output, for
three calls in a row:

```text
POST /products #1: 0.200974s     ← cold start
POST /products #2: 0.000412s     ← warm instance
POST /products #3: 0.000271s     ← warm instance

$ curl localhost:8080/__cloud/stats
{"function":"createproduct","invocations":3,"cold_starts":1}
```

The 200 ms here is a value that was configured, not a measurement. Real cold starts depend on the provider, the language and
the size of the function. `/__cloud/stats` also shows what the bill is based on: invocations, rather than servers.

## A real lesson about data modeling

The first version of this folder kept all of a customer's notifications under **one key**, rewritten on every order. The
benchmark measured a p50 of about 1,350 µs, and it looked like serverless was 16 times slower than the rest.

It was not the architecture: it was the data model. With one key per notification, the p50 dropped to about 86 µs:

```go
// One key per notification: a single key holding the whole list would be rewritten on every event
```

Key-value databases have no joins and no easy way of adding to a list, so **you must design the keys around how the data is
read**. Modeling for them is one of the real learning costs of serverless. The same goes for uniqueness, since there is
no unique index:

```go
// R03: the email itself is a key, because a key-value database has no unique index
```

## Do not trust this folder's latency

The measured p50 of about 86 µs **says nothing about real serverless**. Here, the database is in the same process, with no
network between them, and no cold starts. In a real cloud, every call to the database crosses the network, as in
[folder 07](../07-microservices/README.md).

## How it is tested

Calling a function needs no server. Only the services it uses have to be set up:

```go
response := Handler(env, cloud.Request{Body: []byte(`{"customer_id":"c","coupon":"NOPE","items":[{"product_id":"p","quantity":2}]}`)})
```

## The gain and the price

These are the commonly described trade-offs. They could not be measured here, since nothing runs in a real cloud.

| You gain | You pay |
|---|---|
| No servers to run or patch | Cold starts |
| Scales automatically, including down to zero | Lock-in to the provider's services |
| You pay for use, so idle costs little | Costs that can surprise you under constant high traffic |
| Small units, each deployed alone | Limits on execution time, and hard local debugging |

## When to use it

- **Good for:** irregular traffic, event processing (an uploaded file, a queue message), scheduled jobs, and small APIs.
- **Avoid when:** traffic is constant and heavy (a server is usually cheaper), latency must always be low, work runs for a long time, or you must stay independent of a provider.

Serverless also combines with the others: the [event-driven](../09-event-driven/README.md) style fits it naturally, and each
function can be organized on the inside like a [vertical slice](../11-vertical-slice/README.md).

## How to run

```bash
go run .
COLD_START_MS=200 go run .     # simulates cold starts
curl localhost:8080/__cloud/stats
go test ./...
```

**Measurement:** 17 files, 678 lines of code and 13 packages. See the [full comparison](../RESULTS.md).

**Next step:** [14. SOA](../14-soa/README.md).
