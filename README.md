# Casbin Redis Adapter

Casbin Redis Adapter is an adapter for [Casbin](https://github.com/casbin/casbin) based on [Redis](https://redis.io).

## Installation

    go get github.com/mlsen/casbin-redis-adapter

## Usage

```go
package main

import (
	"github.com/casbin/casbin/v2"
	"github.com/mlsen/casbin-redis-adapter/v2"
)

func main() {
	// The adapter can be initialized from a Redis URL
	a, _ := redisadapter.NewFromURL("redis://:123@localhost:6379/0")

	// Initialize a new Enforcer with the redis adapter
	e, _ := casbin.NewEnforcer("model.conf", a)

	// Load policy from redis
	e.LoadPolicy()

	// Add a policy to redis
	e.AddPolicy("alice", "data1", "read")

	// Check for permissions
	e.Enforce("alice", "data1", "read")

	// Delete a policy from redis
	e.RemovePolicy("alice", "data1", "read")

	// Save all policies to redis
	e.SavePolicy()
}
```

## Initialize Adapter from existing [go-redis](https://github.com/go-redis/redis) Client

```go
package main

import (
	"github.com/casbin/casbin/v2"
	"github.com/go-redis/redis/v8"
	"github.com/mlsen/casbin-redis-adapter/v2"
)

func main() {
	// Initialize the redis client
	rc := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		DB:       0,
		Password: "123",
	})
	// ...

	// The adapter can be initialized from a Redis URL
	a := redisadapter.NewFromClient(rc)

	// ...
}
```

## Main differences to the [official Redis Adapter](https://github.com/casbin/redis-adapter/)

- This Adapter uses [go-redis](https://github.com/go-redis/redis) instead
  of [Redigo](https://github.com/gomodule/redigo)
- This Adapter uses the original CSV format for saving rules to Redis, instead of marshalling them to JSON.

## License

This project is licensed under
the [Apache 2.0 license](https://github.com/mlsen/casbin-redis-adapter/blob/master/LICENSE).