package main

import (
	"fmt"
	"time"
	"context"
	"encoding/json"
        "github.com/kelchy/go-lib/http/client"
)

func main() {
	c, _ := client.New()
	c.SetJSON(false)
	r := c.Get(nil, "https://www.google.com", nil, nil)
	fmt.Println("SAMPLE HTML HEADER", r.Response)
	fmt.Println("SAMPLE HTML FIRST 64 CHARS", r.HTML[:64])

	// get json payload
	c.SetJSON(true)
	j := c.Get(nil, "https://jsonplaceholder.typicode.com/todos/1", nil, nil)
	fmt.Println("SAMPLE JSON HEADER", j.Response)
	jsons, _ := json.Marshal(j.JSON)
	fmt.Println("SAMPLE JSON PAYLOAD", string(jsons))

	// simulate a timeout, but we dont want to see the error stack
	c.SetLogger("empty")
	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Millisecond)
	defer cancel()
	e := c.Get(ctx, "https://www.google.com", nil, nil)
	fmt.Println("SAMPLE ERROR TIMEOUT", e.Error)
}
