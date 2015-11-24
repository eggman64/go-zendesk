package zendesk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type Client struct {
	username string
	password string

	Client    *http.Client
	BaseURL   *url.URL
	UserAgent string

	Tickets *TicketService
}

func NewEnvClient() (*Client, error) {
	domain := os.Getenv("ZENDESK_DOMAIN")
	if domain == "" {
		return nil, errors.New("ZENDESK_DOMAIN not found")
	}

	username := os.Getenv("ZENDESK_USERNAME")
	if username == "" {
		return nil, errors.New("ZENDESK_USERNAME not found")
	}

	password := os.Getenv("ZENDESK_PASSWORD")
	if password == "" {
		return nil, errors.New("ZENDESK_PASSWORD not found")
	}

	return NewClient(domain, username, password)
}

func NewClient(domain, username, password string) (*Client, error) {
	baseURL, err := url.Parse(fmt.Sprintf("https://%s.zendesk.com", domain))

	client := &Client{
		BaseURL:   baseURL,
		UserAgent: "Go-Zendesk",
		username:  username,
		password:  password,
	}

	client.Tickets = NewTicketService(client)

	return client, err
}

func (c *Client) do(method, endpoint string, in interface{}, out interface{}) error {
	rel, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	url := c.BaseURL.ResolveReference(rel)
	req, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("User-Agent", c.UserAgent)

	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return err
		}

		buf := bytes.NewBuffer(payload)
		req.Body = ioutil.NopCloser(buf)

		req.ContentLength = int64(len(payload))
		req.Header.Set("Content-Length", strconv.Itoa(len(payload)))
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if code := res.StatusCode; 200 <= code && code <= 299 {
		if out != nil {
			return json.NewDecoder(res.Body).Decode(out)
		} else {
			return nil
		}
	}

	errRes := new(ErrorResponse)
	errRes.Response = res
	json.NewDecoder(res.Body).Decode(errRes)

	return errRes
}

func (c *Client) Get(endpoint string, out interface{}) error {
	return c.do("GET", endpoint, nil, out)
}

type ErrorResponse struct {
	Response *http.Response

	Type        *string `json:"error,omitmepty"`
	Description *string `json:"description,omitempty"`
}

func (e *ErrorResponse) Error() string {
	msg := fmt.Sprintf("%v %v: %d", e.Response.Request.Method, e.Response.Request.URL, e.Response.StatusCode)

	if e.Type != nil {
		msg = fmt.Sprintf("%s %v", msg, *e.Type)
	}

	if e.Description != nil {
		msg = fmt.Sprintf("%s %v", msg, *e.Description)
	}

	return msg
}
