# 08. MVC (Model-View-Controller)

> Three roles: the **model** holds data and rules, the **view** decides how things look, and the **controller** connects them.

Created by Trygve Reenskaug in the 1970s, for Smalltalk desktop interfaces. It became the structure of the best-known web
frameworks: Rails, Django, Laravel and Spring MVC. For many developers, it is the first architecture they ever meet.

## Structure

```text
08-mvc/
├── main.go
├── architecture_test.go       ← ensures the model stays independent
├── models/                    ← M: data and every business rule
│   ├── db.go
│   ├── product.go
│   ├── customer.go
│   ├── order.go
│   └── notification.go
├── views/                     ← V: JSON and HTML output
│   ├── views.go
│   └── templates/*.html
└── controllers/               ← C: receives the request, asks the model, picks the view
    ├── controller.go
    ├── products.go
    ├── customers.go
    └── orders.go
```

## The dependencies

```text
    request
       │
       ▼
  controller ──────► view ──► response
       │              │
       ▼              ▼
     model ◄──────────┘   (the view only reads the model's data)
```

The model knows neither of the other two. That is the one rule that really matters here.

## Path of a `POST /orders`

1. `controllers/orders.go`: decodes the request and calls the model.
2. `models/order.go`: `db.PlaceOrder` applies all the rules and saves the order.
3. `controllers/orders.go`: picks the view.
4. `views/views.go`: renders JSON or HTML.

## What it looks like in practice

**The controller is thin.** It asks, and it does not decide:

```go
order, err := c.db.PlaceOrder(in.CustomerID, lines, in.Coupon)
if err != nil {
	fail(w, r, err)
	return
}
views.Order(w, r, 201, order)
```

**The model is fat.** With no service layer, the rules live on the model itself:

```go
func (db *DB) PlaceOrder(customerID string, lines []Line, coupon string) (Order, error) {
```

**The view is a real choice.** This is the idea other folders lack: the same model and controller, with two ways of showing
the result. This is the real output of the same route, with only the `Accept` header changing:

```text
$ curl localhost:8080/products/prod-1
{"id":"prod-1","name":"Keyboard","price_cents":30000,"stock":5}

$ curl -H 'Accept: text/html' localhost:8080/products/prod-1
<!doctype html>
<title>Keyboard</title>
<h1>Keyboard</h1>
<p>Price: 30000 cents</p>
<p>In stock: 5</p>
```

The choice is made in one place:

```go
if !strings.Contains(r.Header.Get("Accept"), "text/html") {
```

## How it differs from layers (folder 02)

They look alike, and many projects mix the two. The difference is what each one divides:

| | Layered (02) | MVC (08) |
|---|---|---|
| Main concern | The whole system, from HTTP to data | Mainly the presentation |
| Where the rules live | In a service layer | On the model |
| Presentation | One handler layer does everything | Split into controller and view |
| Output format | JSON tags on the shared model | Only in the view |

MVC says a lot about the interface and little about the back end: it does not say how to organize the model. That is
why, as projects grow, it is common to add a service layer. At that point, the project has become MVC at the top and
layers underneath.

## The most common problems

- **Fat controller:** business rules creep into the controller, where they cannot be reused or tested without HTTP.
- **God model:** with everything on the model, `models/order.go` grows without limit. Here it is already the largest file in this folder.
- **Database at the center:** frameworks tie the model to the ORM, so the rules end up depending on the database. This is the problem [folder 05](../05-hexagonal/README.md) solves.

## How it is tested

The model is tested with no HTTP at all, and a fake gateway is just a function:

```go
db := NewDB(func(int) bool { return false })
```

The view is tested through the controller, including that user data is escaped in HTML:

```go
if strings.Contains(got, "<b>Keyboard") {
```

## The gain and the price

| You gain | You pay |
|---|---|
| A structure almost everyone already knows | Little guidance as the model grows |
| Several output formats with the same rules | The model tends to become a giant |
| Rules testable without HTTP | Nothing but discipline keeps controllers thin |
| Strong support in frameworks | The model is often tied to the database |

## When to use it

- **Good for:** web applications with server-rendered pages, CRUDs, and projects using a framework built around it.
- **Avoid as the only structure when:** the domain has many rules. Keep MVC at the edge, and organize the rules with [folder 04](../04-modular-monolith/README.md) or [folder 05](../05-hexagonal/README.md).

## How to run

```bash
go run .
curl -H 'Accept: text/html' localhost:8080/products/prod-1
go test ./...
```

**Measurement:** 14 files, 557 lines of code and 4 packages. See the [full comparison](../RESULTS.md). Its binary is
larger (12.9 MB against 9.5 MB), most likely because of `html/template`, which no other folder uses.

**Next step:** [09. Event-driven](../09-event-driven/README.md).
