package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kelchy/go-lib/http/client"
)

func main() {
	// Create a rate-limited client with 10 requests per second and burst of 5
	// This means:
	// - Maximum 10 requests per second on average
	// - Can burst up to 5 requests immediately if tokens are available
	c, err := client.NewRateLimited(10.0, 5)
	if err != nil {
		fmt.Println("Error creating rate-limited client:", err)
		return
	}

	// Configure the client (same API as regular client)
	c.SetJSON(true)
	c.SetTimeout(30000)

	// Make requests - rate limiting is applied automatically
	ctx := context.Background()

	// These requests will be rate-limited to 10 per second
	for i := 0; i < 3; i++ {
		r := c.Get(ctx, "https://jsonplaceholder.typicode.com/todos/1", nil, nil)
		if r.Error != nil {
			fmt.Printf("Request %d error: %v\n", i+1, r.Error)
			continue
		}
		jsons, _ := json.Marshal(r.JSON)
		fmt.Printf("Request %d response: %s\n", i+1, string(jsons))
	}

	// Example with HTTP/2 rate-limited client
	c2, err := client.NewRateLimitedHTTP2(5.0, 2)
	if err != nil {
		fmt.Println("Error creating rate-limited HTTP/2 client:", err)
		return
	}

	fmt.Println("\nUsing HTTP/2 rate-limited client (5 req/sec, burst 2):")
	r2 := c2.Get(ctx, "https://jsonplaceholder.typicode.com/todos/2", nil, nil)
	if r2.Error == nil {
		jsons, _ := json.Marshal(r2.JSON)
		fmt.Println("HTTP/2 response:", string(jsons))
	}
}
