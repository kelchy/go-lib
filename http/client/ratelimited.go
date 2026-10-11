package client

import (
	"context"

	"golang.org/x/time/rate"
)

// RateLimitedClient - rate limited http client instance
type RateLimitedClient struct {
	*Client
	limiter *rate.Limiter
}

// NewRateLimited - creates and returns a rate limited http client
// rateLimit: maximum number of requests per second
// burst: maximum burst size (allows short bursts above the rate limit)
func NewRateLimited(rateLimit float64, burst int) (RateLimitedClient, error) {
	client, err := New()
	if err != nil {
		return RateLimitedClient{}, err
	}

	limiter := rate.NewLimiter(rate.Limit(rateLimit), burst)
	return RateLimitedClient{
		Client:  &client,
		limiter: limiter,
	}, nil
}

// NewRateLimitedHTTP2 - creates and returns a rate limited http/2 client
// rateLimit: maximum number of requests per second
// burst: maximum burst size (allows short bursts above the rate limit)
func NewRateLimitedHTTP2(rateLimit float64, burst int) (RateLimitedClient, error) {
	client, err := NewHTTP2()
	if err != nil {
		return RateLimitedClient{}, err
	}

	limiter := rate.NewLimiter(rate.Limit(rateLimit), burst)
	return RateLimitedClient{
		Client:  &client,
		limiter: limiter,
	}, nil
}

// Get - http call using get method with rate limiting
func (c RateLimitedClient) Get(ctx context.Context, url string, data []byte, hdr map[string]string) Res {
	if err := c.limiter.Wait(ctx); err != nil {
		var res Res
		res.Error = err
		return res
	}
	return c.Client.Get(ctx, url, data, hdr)
}

// Post - http call using post method with rate limiting
func (c RateLimitedClient) Post(ctx context.Context, url string, data []byte, hdr map[string]string) Res {
	if err := c.limiter.Wait(ctx); err != nil {
		var res Res
		res.Error = err
		return res
	}
	return c.Client.Post(ctx, url, data, hdr)
}

// Put - http call using put method with rate limiting
func (c RateLimitedClient) Put(ctx context.Context, url string, data []byte, hdr map[string]string) Res {
	if err := c.limiter.Wait(ctx); err != nil {
		var res Res
		res.Error = err
		return res
	}
	return c.Client.Put(ctx, url, data, hdr)
}

// Patch - http call using patch method with rate limiting
func (c RateLimitedClient) Patch(ctx context.Context, url string, data []byte, hdr map[string]string) Res {
	if err := c.limiter.Wait(ctx); err != nil {
		var res Res
		res.Error = err
		return res
	}
	return c.Client.Patch(ctx, url, data, hdr)
}

// Delete - http call using delete method with rate limiting
func (c RateLimitedClient) Delete(ctx context.Context, url string, data []byte, hdr map[string]string) Res {
	if err := c.limiter.Wait(ctx); err != nil {
		var res Res
		res.Error = err
		return res
	}
	return c.Client.Delete(ctx, url, data, hdr)
}

// SetTimeout - changes the default timeout set on code
func (c *RateLimitedClient) SetTimeout(timeout int) {
	c.Client.SetTimeout(timeout)
}

// SetLogger - changes the logger mode
func (c *RateLimitedClient) SetLogger(logtype string) {
	c.Client.SetLogger(logtype)
}

// SetJSON - changes the default JSON true to false - HTML content
func (c *RateLimitedClient) SetJSON(enabled bool) {
	c.Client.SetJSON(enabled)
}
