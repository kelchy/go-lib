package main

import (
	"time"
	"fmt"
	"os"
	"github.com/kelchy/go-lib/redis"
)

func main() {
	uri := os.Getenv("REDIS_URI")
	redisclient, e := redis.New(uri)
        if e != nil {
		fmt.Println(e)
		return
	}

	keys, _ := redisclient.Keys(nil, "*")
	fmt.Println("keys", keys)

	res, _ := redisclient.Set(nil, "key", "value2", 10 * time.Second)
	fmt.Println("result", res)

	resi, _ := redisclient.Del(nil, "key")
	fmt.Println("result int", resi)

	resb, _ := redisclient.SetNX(nil, "key", "value2", 10 * time.Second)
	fmt.Println("result bool", resb)

	val, _ := redisclient.Get(nil, "key")
	fmt.Println("result value", val)

	lock, _ := redisclient.Lock(nil, "locktest", 20 * time.Second)
	fmt.Println("result lock", lock)

	unlock, _ := redisclient.Unlock(nil, "locktest")
	fmt.Println("result unlock", unlock)
}
