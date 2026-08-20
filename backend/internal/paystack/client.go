package paystack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.paystack.co"

type Client struct {
	secretKey string
	publicKey string
	dvaBank   string
	http      *http.Client
}

func NewClient(secretKey, publicKey, dvaBank string) *Client {
	if dvaBank == "" {
		dvaBank = "wema-bank"
	}
	return &Client{
		secretKey: strings.TrimSpace(secretKey),
		publicKey: strings.TrimSpace(publicKey),
		dvaBank:   dvaBank,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.secretKey != ""
}

func (c *Client) PublicKey() string { return c.publicKey }
func (c *Client) DVABank() string   { return c.dvaBank }

type apiEnvelope struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type Customer struct {
	ID           int64  `json:"id"`
	CustomerCode string `json:"customer_code"`
	Email        string `json:"email"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
}

type DedicatedAccount struct {
	ID            int64  `json:"id"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	Bank          struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
		ID   int    `json:"id"`
	} `json:"bank"`
	Currency string `json:"currency"`
	Active   bool   `json:"active"`
	Customer struct {
		CustomerCode string `json:"customer_code"`
		Email        string `json:"email"`
	} `json:"customer"`
}

func (c *Client) CreateCustomer(ctx context.Context, email, firstName, lastName, phone string) (*Customer, error) {
	body := map[string]any{
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
	}
	if phone != "" {
		body["phone"] = phone
	}
	var cust Customer
	if err := c.post(ctx, "/customer", body, &cust); err != nil {
		// Idempotent: fetch existing by email if already created
		if existing, ferr := c.GetCustomerByEmail(ctx, email); ferr == nil {
			return existing, nil
		}
		return nil, err
	}
	return &cust, nil
}

func (c *Client) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	var cust Customer
	if err := c.get(ctx, "/customer/"+email, &cust); err != nil {
		return nil, err
	}
	return &cust, nil
}

func (c *Client) ValidateCustomer(ctx context.Context, customerCode, country, accountType, accountNumber, bvn, bankCode string) error {
	body := map[string]any{
		"country":        country,
		"type":           accountType,
		"account_number": accountNumber,
		"bvn":            bvn,
		"bank_code":      bankCode,
		"first_name":     "Customer",
		"last_name":      "HomeGauge",
	}
	return c.post(ctx, "/customer/"+customerCode+"/identification", body, nil)
}

func (c *Client) CreateDedicatedAccount(ctx context.Context, customerCode, preferredBank, firstName, lastName, phone string) (*DedicatedAccount, error) {
	body := map[string]any{
		"customer":      customerCode,
		"preferred_bank": preferredBank,
	}
	if firstName != "" {
		body["first_name"] = firstName
	}
	if lastName != "" {
		body["last_name"] = lastName
	}
	if phone != "" {
		body["phone"] = phone
	}
	var acc DedicatedAccount
	if err := c.post(ctx, "/dedicated_account", body, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

func (c *Client) AssignDedicatedAccount(ctx context.Context, email, firstName, lastName, phone, preferredBank, country string) (*DedicatedAccount, error) {
	body := map[string]any{
		"email":          email,
		"first_name":     firstName,
		"last_name":      lastName,
		"preferred_bank": preferredBank,
		"country":        country,
	}
	if phone != "" {
		body["phone"] = phone
	}
	var acc DedicatedAccount
	if err := c.post(ctx, "/dedicated_account/assign", body, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var env apiEnvelope
	if err := json.Unmarshal(b, &env); err != nil {
		return fmt.Errorf("paystack: invalid response (%d)", res.StatusCode)
	}
	if res.StatusCode >= 300 || !env.Status {
		msg := env.Message
		if msg == "" {
			msg = string(b)
		}
		return fmt.Errorf("paystack: %s", msg)
	}
	if out == nil || len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}
