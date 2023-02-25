```
	c, _ := client.New()
	r := c.Get("https://www.google.com", nil, nil, 0)
	fmt.Println("SAMPLE HTML HEADER", r.Response)
	html, _ := r.Html()
	fmt.Println("SAMPLE HTML FIRST 64 CHARS", html[:64])
```
