# 14. SOA (Service-Oriented Architecture)

> Large services, a shared data model, and a central **bus** in the middle of every conversation.

SOA came before microservices, and it is still common in banks, insurers, governments and other large companies. Each
service exposes business capabilities for the whole enterprise, and an **Enterprise Service Bus (ESB)** connects them.

Today, it is often remembered as what microservices reacted against. Comparing the two is the main reason this folder exists.

## Structure

```text
14-soa/
├── build.sh  run.sh  test.sh
├── canonical/canonical.go   ← the shared model: EVERY service imports this
└── services/
    ├── esb/                 ← the bus: routes, translates, validates (8080)
    ├── database/            ← the database the whole company shares (8081)
    ├── customer/            ← customer records (8082)
    ├── sales/               ← catalog AND orders together (8083)
    └── billing/             ← charging AND notifications together (8084)
```

## How the parts talk

```text
  client ──► ESB ──┬──► customer ──┐
             ▲     ├──► sales ─────┼──► database (shared by all)
             │     └──► billing ───┘
             │            ▲
             └── sales ───┘   every call between services goes through the bus
```

Services never call each other. They name an operation, and the bus decides who answers it.

## SOA and microservices, side by side

These are the same kind of system, with opposite decisions:

| | SOA (14) | Microservices (07) |
|---|---|---|
| Size of a service | Large: `sales` has catalog and orders | Small: one capability each |
| Middle of the system | A smart bus | A dumb gateway that only routes |
| Business logic of integration | In the bus | In the services |
| Data | One shared database | One store per service |
| Code shared between services | A canonical model | None, on purpose |
| Go modules | 1 | 6 |
| Main goal | Reuse across the enterprise | Independence between teams |

## What it looks like in practice

**The canonical model is imported by everyone.** No translation between services, and a single change touches them all:

```go
// Package canonical is the enterprise's shared vocabulary: every service imports it.
```

**The bus knows the whole system.** Moving an operation to another service changes one line here, and none of the callers:

```go
"PlaceOrder":        "SALES_URL",
"ChargePayment":     "BILLING_URL",
```

**A service asks for an operation, not a service:**

```go
if err := bus.Call("ChargePayment", canonical.Charge{AmountCents: order.TotalCents}, nil); err != nil {
```

**The shared database is a tempting shortcut.** `sales` needs to know whether a customer exists, so it reads the table
directly, rather than asking the service in charge of it:

```go
// R05: straight to the customers table, skipping the customer service. Quick, and coupling:
// if that team changes the table, this breaks.
```

## What happens when something goes down

**Real result**, killing processes while the system runs:

| Situation | Effect |
|---|---|
| `billing` down | `POST /orders` returns **503**, while reads such as `GET /customers` still work |
| `esb` down | **Everything stops**: connection refused, even with all the services running |

The first row hides a real inconsistency, seen in the same test: **the client got a 503, but the order was saved and the
stock went down.** With 5 units and two orders, one accepted and one failed, the stock ended at 3. The order is written
first, and the failure only comes at the notification:

```go
return order, notify(order, "order_created")
```

With no transaction across the calls, nothing undoes that write. A client that retries would buy twice. This is left here on
purpose, because it is the kind of failure this style invites. Folder 07 avoids this one by notifying in the
background, and [folder 09](../09-event-driven/README.md) by reacting to events.

The second row is the classic criticism of the ESB: a single point of failure, and a bottleneck that every change has to go
through, often owned by a separate team.

## The measured cost

Each order makes many trips: to the bus, to `sales`, several to the database, back through the bus to `billing`, and to the
database again. Same rules, same machine, on localhost:

| | Monoliths | Microservices (07) | SOA (14) |
|---|---|---|---|
| p50 of `POST /orders` | about 80 µs | about 487 µs | **about 1,025 µs** |
| Memory | about 18 MB | about 105 MB | about 100 MB |

On a real network, each of those trips would cost far more.

## How it is tested

Since all routing knowledge lives in the bus, a typo there would break everything, so it gets its own test:

```go
func TestEveryPublicRouteHasAProvider(t *testing.T) {
```

## The gain and the price

| You gain | You pay |
|---|---|
| Services reused across the company | A central bus that can stop everything |
| One shared vocabulary | Any change to the model touches everyone |
| Callers independent of who provides a service | Latency from going through the middle |
| Central place for control and auditing | Shared database, so teams are coupled through tables |

## When to use it

- **Good for:** integrating many existing systems in a large organization, especially legacy systems that cannot change much, and where central governance is required.
- **Avoid in new systems** with no such need. Knowing it matters mostly because you will meet it in large companies, and because it explains why microservices look the way they do.

## How to run

```bash
./run.sh                     # builds and starts the 5 processes; ESB on port 8080
BASE_PORT=9000 ./run.sh      # uses other ports
pkill -x billing             # orders fail, but look at the stock afterwards
pkill -x esb                 # everything stops
```

**Measurement:** 9 files, 791 lines of code and 5 processes. See the [full comparison](../RESULTS.md).

**Next step:** [15. Microkernel](../15-microkernel/README.md).
