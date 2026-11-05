package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

// A function call became a network call. Everything in this file exists only because of
// that: timeouts, serialization, status codes and the new question "what if the other side does not answer?".

var errUnavailable = errors.New("dependency unavailable")

type RemoteError struct {
	Status  int
	Code    string
	Message string
}

func (e *RemoteError) Error() string { return e.Message }

type Clients struct {
	http          *http.Client
	catalog       string
	customers     string
	payments      string
	notifications string
}

func NewClients(catalog, customers, payments, notifications string) *Clients {
	return &Clients{
		http:          &http.Client{Timeout: 2 * time.Second},
		catalog:       catalog,
		customers:     customers,
		payments:      payments,
		notifications: notifications,
	}
}

// send makes the call and returns the outcome, decoding the body into out on success.
func (c *Clients) send(method, url string, in, out any) error {
	var body bytes.Buffer
	if in != nil {
		if err := json.NewEncoder(&body).Encode(in); err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", errUnavailable, err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 500 {
		return fmt.Errorf("%w: status %d", errUnavailable, res.StatusCode)
	}
	if res.StatusCode >= 400 {
		var remote struct {
			Code  string `json:"code"`
			Error string `json:"error"`
		}
		_ = json.NewDecoder(res.Body).Decode(&remote)
		return &RemoteError{res.StatusCode, remote.Code, remote.Error}
	}
	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}

func (c *Clients) CustomerExists(id string) (bool, error) {
	err := c.send("GET", c.customers+"/customers/"+id, nil, nil)
	var remote *RemoteError
	if errors.As(err, &remote) && remote.Status == 404 {
		return false, nil
	}
	return err == nil, err
}

func (c *Clients) Reserve(lines []Line) ([]Item, error) {
	var items []Item
	err := c.send("POST", c.catalog+"/reservations", lines, &items)
	return items, err
}

func (c *Clients) Release(lines []Line) error {
	return c.send("POST", c.catalog+"/releases", lines, nil)
}

func (c *Clients) Charge(amountCents int) error {
	return c.send("POST", c.payments+"/charges", map[string]int{"amount_cents": amountCents}, nil)
}

// Notify runs in the background, with retries: a notification outage must not
// take down a sale. The price is possibly losing the notice if every attempt fails.
func (c *Clients) Notify(customerID, orderID, kind string) {
	payload := map[string]string{"customer_id": customerID, "order_id": orderID, "type": kind}
	go func() {
		for attempt := 0; attempt < 3; attempt++ {
			if err := c.send("POST", c.notifications+"/notifications", payload, nil); err == nil {
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
		log.Printf("notification lost: %s of order %s", kind, orderID)
	}()
}
