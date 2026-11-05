# 15. Microkernel (plugin architecture)

> A small, stable core, with everything else added as plugins.

You use this every day: VS Code, browsers with extensions, WordPress, and Eclipse are all built like this. The core knows how
to run plugins, and it does not know which ones exist.

## Structure

```text
15-microkernel/
├── main.go                  ← picks the plugins
├── architecture_test.go     ← the kernel knows no plugin
├── kernel/                  ← THE CORE
│   ├── plugin.go            ← the extension points: the only ways in
│   ├── kernel.go            ← what every store has: stock, orders, states
│   └── http.go
└── plugins/                 ← EVERYTHING OPTIONAL
    ├── welcomecoupon/
    ├── bigordercoupon/
    ├── flatshipping/
    ├── fakepayment/
    └── notifications/       ← has its own data and its own route
```

## The dependencies

```text
   welcomecoupon ─┐
  bigordercoupon ─┤
    flatshipping ─┼──► kernel
     fakepayment ─┤
   notifications ─┘

   main picks which plugins to load. The kernel imports none of them.
```

## What is core, and what is a plugin?

This is the main design decision. Here, the question was: "what would every store need, and what varies?"

| In the kernel | In a plugin |
|---|---|
| Products, customers, stock | Each coupon |
| The order's states (R11) | The shipping policy |
| Reserving and returning stock | The payment provider |
| How to calculate the total (R10) | Notifications |

## What it looks like in practice

**A plugin is small, and says what it adds:**

```go
func (p Plugin) Install(registry *kernel.Registry) { registry.AddCoupon("BIG20", p) }
```

**The kernel only knows that coupons exist:**

```go
// R08: the kernel only knows that coupons exist. The plugins hold the rules.
```

**A plugin can bring a whole feature**, with its own data and its own endpoint:

```go
registry.OnOrderEvent(p.record)
registry.AddRoute("GET /customers/{id}/notifications", p.list)
```

**The kernel works alone.** With no plugins, there are no coupons, shipping is free and every payment is declined:

```go
// New creates a kernel that already works with no plugins at all: no coupons, free shipping,
// and every payment declined. A plugin only ever adds to this.
```

## See it running

Turning features off with no rebuild. This is real output, with two plugins disabled:

```text
$ DISABLED_PLUGINS=welcome-coupon,notifications go run .

$ curl localhost:8080/plugins
["big-order-coupon","flat-shipping","fake-payment"]

$ curl localhost:8080/orders -d '{..., "coupon":"WELCOME10", ...}'
{"code":"invalid_coupon","error":"invalid coupon"}

$ curl localhost:8080/customers/cust-2/notifications     → 404
```

With everything loaded, the same code passes the contract suite. So the store's complete behavior is the kernel **plus**
its plugins.

## It compares best with the hexagon

Both plug things into a core, but with different goals:

| | Hexagonal (05) | Microkernel (15) |
|---|---|---|
| What gets plugged in | Technology: database, gateway | Features: coupons, notifications |
| How many at once | Usually one per port | As many as you want |
| Without the plugin | The system does not run | The system runs, with less |
| The question it answers | How do I isolate technology? | How do I add features without touching the core? |

Compared with the [Open/Closed principle](../principles/solid/README.md), this is that same idea at the scale of a whole system.

## How it is tested

The proof of the architecture is a feature that does not exist in the code at all. It is written in the test, and
works with no change to the kernel:

```go
func (halfOff) Install(registry *kernel.Registry) { registry.AddCoupon("HALF", halfOff{}) }
```

## The limits

**Plugins can only do what the extension points allow.** A plugin that wanted to change how stock is reserved has no way in.
Designing those points is the hard part, and changing them later breaks every plugin. That stability is the real cost.

**Plugins here are chosen at build time.** Real systems often load them at runtime. Go has a `plugin` package for this, but it
is limited, so systems in Go often run plugins as separate processes instead. VS Code also isolates extensions from the
main editor, so a bad one cannot take down the core. Here, a plugin runs inside the kernel, and it is fully trusted.

**Plugins can conflict.** Two plugins calling `SetShipping` means the last one wins, silently. Real systems need ordering
and conflict rules.

## The gain and the price

| You gain | You pay |
|---|---|
| New features without touching the core | Extension points that are hard to get right |
| Features turned on and off per customer | Extension points that are hard to change later |
| A small, stable core that is easy to test | Plugin ordering and conflicts |
| Third parties can extend the product | With third-party code, security and isolation |

## When to use it

- **Good for:** products with many optional features, editors and IDEs, or systems sold with different feature sets to each customer.
- **Avoid when:** all features are always on and are maintained by one team. Plain modules, as in [folder 04](../04-modular-monolith/README.md), are simpler.

## How to run

```bash
go run .
DISABLED_PLUGINS=notifications go run .
curl localhost:8080/plugins
go test ./...
```

**Measurement:** 11 files, 506 lines of code and 7 packages. See the [full comparison](../RESULTS.md).

**Next step:** [code principles](../principles/README.md), which apply to any of the architectures.
