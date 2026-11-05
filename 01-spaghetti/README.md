# 01. Spaghetti (the baseline)

> Everything in one file, with no separation at all. It serves as a ruler: the "before" of every other folder.

This is not a model to follow. It exists because every comparison needs a starting point, and because it teaches the
most important lesson in the repository:

**This code passes 100% of the contract suite, just like every other architecture.**

Architecture does not change what the system does today. It changes how much it costs to change it tomorrow.

## Structure

```text
01-spaghetti/
└── main.go    ← everything: routes, validation, rules, data and JSON
```

There is nothing to draw: a single function, `handle`, does the whole job.

## What it looks like in practice

State is global, data is untyped maps, and business rules are mixed with HTTP:

```go
var products = map[string]map[string]any{}
var customers = map[string]map[string]any{}
var orders = map[string]map[string]any{}
```

Look at the coupon rule. To understand the discount, you also have to read HTTP statuses and error JSON:

```go
if cp == "WELCOME10" {
	for _, o := range orders {
		if o["customer_id"] == cid && o["status"] != "cancelled" {
			w.WriteHeader(422)
			w.Write([]byte(`{"code":"coupon_not_eligible","error":"only on the first order"}`))
			return
		}
	}
	disc = float64(int(st) * 10 / 100)
```

And the "payment gateway" is just an `if` in the middle of the handler:

```go
if o["total_cents"].(float64) > 1000000 {
	w.WriteHeader(402)
```

## What hurts here

Picture these requests, one by one:

| Request | What happens |
|---|---|
| Test the coupon rule | Impossible without starting HTTP: the rule does not exist outside the handler |
| Swap the payment gateway | There is no gateway to swap: you have to find the `if` |
| Store the data in Postgres | Every line that touches a map changes, which is almost the whole file |
| Two people working at once | Guaranteed merge conflict: there is only one file |
| Rename the `price_cents` field | A loose string, with no help from the compiler; a typo becomes a production bug |

One more thing: **the function has access to everything**. Nothing stops customer code from changing an order.
With no boundaries in the code, the only boundary is the memory of whoever is working on it.

## But it has merits too

| Merit | Why |
|---|---|
| It is the smallest of all | 279 lines of code, against 891 in clean architecture ([measurement](../RESULTS.md)) |
| It reads top to bottom | No jumping between files |
| It can be written in minutes | There are no structural decisions to make |

So a throwaway script or a one-afternoon prototype can be written this way without a problem. The mistake is letting
this code survive and grow.

Note that it is not faster: across three runs of the measurement, it was always the slowest monolith, by a margin
that varied from run to run. "No layers" does not mean "more efficient". The likely cause is the untyped maps, but
that was not investigated in depth.

## When to use it

- **Fine for:** a one-off script, a throwaway prototype, a one-day proof of concept.
- **Avoid:** in anything someone else will maintain, or that will last more than a few weeks.

## How to run

```bash
go run .      # port 8080; use PORT=9000 to change it
```

**Next step:** [02. Layered monolith](../02-layered/README.md), the first separation almost every project makes.
