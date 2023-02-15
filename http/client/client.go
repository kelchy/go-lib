package client

import (
	"bytes"
	"context"
	"time"
	"strings"
	"net"
	"net/http"
	"crypto/tls"
	"golang.org/x/net/http2"
	"github.com/kelchy/go-lib/log"
)

// TIMEOUT - default timeout
const TIMEOUT = 30000

// Client - initiated client instance
type Client struct {
	Client	*http.Client
	timeout	int
	log	log.Log
	JSON	bool
}

// New - creates an returns http client
func New() (Client, error) {
	var client Client
	c := &http.Client{
		Transport: &http.Transport{
/*
			CloseIdleConnections:
			MaxIdleConnsPerHost:
			DisableKeepAlives:
*/
		},
	}
	client.Client = c
	client.timeout = TIMEOUT
	l, e := log.New("")
	if e != nil {
		return client, e
	}
	client.log = l
	client.JSON = true
	return client, nil
}

// NewHTTP2 - creates and returns http/2 client
func NewHTTP2() (Client, error) {
	var client Client
	c := &http.Client{
		Transport: &http2.Transport{
			// So http2.Transport doesn't complain the URL scheme isn't 'https'
			AllowHTTP: true,
			// Pretend we are dialing a TLS endpoint.
			// Note, we ignore the passed tls.Config
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				// h2c
				return net.Dial(network, addr)
			},
		},
	}
	client.Client = c
	client.timeout = TIMEOUT
	l, e := log.New("")
	client.log = l
	if e != nil {
		return client, e
	}
	return client, nil
}

// SetTimeout - changes the default timeout set on code
func (c *Client) SetTimeout(timeout int) {
	c.timeout = timeout
}

// SetLogger - changes the logger mode
func (c *Client) SetLogger(logtype string) {
	l, e := log.New(logtype)
	if e == nil {
		c.log = l
	}
}

// SetJSON - changes the default JSON true to false - HTML content
func (c *Client) SetJSON(enabled bool) {
	c.JSON = enabled
}

// Get - http call using get method, timeout in milli
func (c Client) Get(ctx context.Context, url string, data []byte, hdr map[string]string) Res {
	return c.req(ctx, "GET", url, data, hdr)
}

// Post - http call using post method, timeout in milli
func (c Client) Post(ctx context.Context, url string, data []byte, hdr map[string]string) Res {
	return c.req(ctx, "POST", url, data, hdr)
}

// Put - http call using put method, timeout in milli
func (c Client) Put(ctx context.Context, url string, data []byte, hdr map[string]string) Res {
	return c.req(ctx, "PUT", url, data, hdr)
}

// Patch - http call using patch method, timeout in milli
func (c Client) Patch(ctx context.Context, url string, data []byte, hdr map[string]string) Res {
	return c.req(ctx, "PATCH", url, data, hdr)
}

// Delete - http call using delete method, timeout in milli
func (c Client) Delete(ctx context.Context, url string, data []byte, hdr map[string]string) Res {
	return c.req(ctx, "DELETE", url, data, hdr)
}

func (c Client) req(ctx context.Context, method string, url string, data []byte, hdr map[string]string) Res {
	var cancel context.CancelFunc
	if ctx == nil {
		timesec := c.timeout
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timesec) * time.Millisecond)
		defer cancel()
	}

	var res Res
	res.log = c.log
	req, e := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, bytes.NewBuffer(data))
	if e != nil {
		c.log.Error("HTTPC_NEW", e)
		res.Error = e
		return res
	}

	// default json for RESTful
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}

	resp, e := c.Client.Do(req)
	if e != nil {
		c.log.Error("HTTPC_DO", e)
		res.Error = e
		return res
	}
	res.Response = *resp
	if c.JSON == true {
		res.JSONparse()
	} else {
		res.HTMLparse()
	}
	return res
}
