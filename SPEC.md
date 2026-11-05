# Functional specification: the store

Every architecture in this repository implements **exactly** this specification.
The suite in [`contract/`](contract/) tests each one from the outside, over HTTP only, with no knowledge of the code.

The domain is a small store, on purpose: it has enough rules for the architectures to differ, and few enough
that you can read them all.

## Why the rules are numbered

Each rule appears in the code as a `// R08` comment. That lets you ask "where does the coupon rule live in each
architecture?":

```bash
./scripts/where-rule.sh R08
```

Comparing the answers across folders is the fastest way to feel the difference between them.

## Endpoints

| Method | Route                           | Success | What it does                 |
|--------|---------------------------------|---------|------------------------------|
| GET    | `/health`                       | 200     | Reports the service is up    |
| POST   | `/products`                     | 201     | Creates a product            |
| GET    | `/products/{id}`                | 200     | Gets a product (and stock)   |
| POST   | `/customers`                    | 201     | Creates a customer           |
| GET    | `/customers/{id}`               | 200     | Gets a customer              |
| GET    | `/customers/{id}/notifications` | 200     | Lists a customer's notices   |
| POST   | `/orders`                       | 201     | Creates an order             |
| GET    | `/orders/{id}`                  | 200     | Gets an order                |
| POST   | `/orders/{id}/pay`              | 200     | Pays an order                |
| POST   | `/orders/{id}/cancel`           | 200     | Cancels an order             |

## Formats

Money is always an integer in cents, never floating point. IDs are opaque strings.

```jsonc
// Product
{ "id": "...", "name": "Keyboard", "price_cents": 15000, "stock": 10 }

// Customer
{ "id": "...", "name": "Ana", "email": "ana@example.com" }

// POST /orders (input)
{ "customer_id": "...", "items": [{ "product_id": "...", "quantity": 2 }], "coupon": "WELCOME10" }

// Order
{
  "id": "...", "customer_id": "...", "status": "pending",
  "items": [{ "product_id": "...", "quantity": 2, "unit_price_cents": 15000 }],
  "coupon": "WELCOME10",
  "subtotal_cents": 30000, "discount_cents": 3000, "shipping_cents": 0, "total_cents": 27000
}

// Notification
{ "customer_id": "...", "order_id": "...", "type": "order_paid" }

// Error (any)
{ "code": "insufficient_stock", "error": "human-readable message" }
```

## Business rules

| #   | Rule | Failure |
|-----|------|---------|
| R01 | A product needs a name, a price above zero and stock of zero or more. | 422 `validation` |
| R02 | A customer needs a name and an email in the form `something@something`. | 422 `validation` |
| R03 | The email is unique, ignoring case. | 409 `email_taken` |
| R04 | An order has at least 1 item, a quantity from 1 to 10 per item and no repeated product. | 422 `validation` |
| R05 | The order's customer and products must exist. | 422 `customer_not_found` / `product_not_found` |
| R06 | Stock is reserved when the order is created, **all or nothing**: if one item is short, none is reserved. | 409 `insufficient_stock` |
| R07 | The order keeps the price at purchase time. Changing the product later does not change the order. | |
| R08 | Coupons: `WELCOME10` gives 10% on the first order only (cancelled orders do not count). `BIG20` gives 20% on a subtotal of $500.00 or more. A failed coupon does **not** consume stock. | 422 `invalid_coupon` / `coupon_not_eligible` |
| R09 | Free shipping if `subtotal - discount >= $200.00`. Otherwise, $20.00. | |
| R10 | `total = subtotal - discount + shipping`. Percentages round down, in cents. | |
| R11 | States: `pending → paid`, `pending → cancelled`, `paid → cancelled`. `cancelled` is final, and paying twice is an error. | 409 `invalid_transition` |
| R12 | The payment gateway declines totals above $10,000.00. The order stays `pending` and nothing is notified. | 402 `payment_declined` |
| R13 | Cancelling returns the stock. | |
| R14 | The customer is notified of `order_created`, `order_paid` and `order_cancelled`. Delivery may be asynchronous (up to 2 s). | |

## General errors

| Situation                  | Response |
|----------------------------|----------|
| Malformed JSON             | 400 `invalid_json` |
| Resource in the URL does not exist | 404 `not_found` |
| Dependency down (only meaningful in microservices) | 503 `service_unavailable` |

`GET /customers/{id}/notifications` never returns 404: with no notifications, it answers `[]`.

## Left out on purpose

Authentication, a real database, pagination and observability are out of scope.
All of them matter, but here they would only add noise to a comparison of **structure**.
That is why data stays in memory, and each implementation is free to organize it however it likes.
