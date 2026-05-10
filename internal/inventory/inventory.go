package inventory

import (
	"sync"
)

// Inventory implements a simple in-memory inventory with atomic check-and-decrement.
// This is a demo replacement for a Redis Lua script + Postgres in production.
type Inventory struct {
	mu      sync.Mutex
	stock   map[string]int
	version map[string]int64
}

func New() *Inventory {
	return &Inventory{
		stock:   make(map[string]int),
		version: make(map[string]int64),
	}
}

// Initialize sets starting stock levels for items.
func (i *Inventory) Initialize(items map[string]int) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for k, v := range items {
		i.stock[k] = v
		i.version[k] = 1
	}
}

// CheckAndDecrement atomically checks stock > 0 and decrements it.
// Returns (true, remaining) if successful, or (false, 0) if sold out.
func (i *Inventory) CheckAndDecrement(itemID string) (bool, int) {
	i.mu.Lock()
	defer i.mu.Unlock()

	cur, ok := i.stock[itemID]
	if !ok || cur <= 0 {
		return false, 0
	}
	i.stock[itemID] = cur - 1
	i.version[itemID]++
	return true, i.stock[itemID]
}

// RedisLuaScript is an example Lua script you can use with Redis to perform
// the atomic check-and-decrement in one round-trip in production.
const RedisLuaScript = `
local key = KEYS[1]
local stock = tonumber(redis.call('GET', key) or '-1')
if stock <= 0 then
  return -1
end
redis.call('DECR', key)
return stock - 1
`
