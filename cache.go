package hhdhcp

import (
	"sync"
	"time"
)

type cache struct {
	data map[string]string
	sync.RWMutex
}

type Cache interface {
	Get(key string) (string, error)
	Add(key string, value string)
	AddWithTTL(key string, value string, ttl time.Duration, callBackFunc CallbackFunc)
	Delete(key string)
	Keys() []string
}

type CallbackFunc func(key string, val string)

func NewCache() Cache {
	return &cache{
		data: map[string]string{},
	}
}
func (c *cache) Add(key string, value string) {
	c.Lock()
	c.data[key] = value
	c.Unlock()
}

func (c *cache) Get(key string) (string, error) {
	c.RLock()
	defer c.RUnlock()
	if val, ok := c.data[key]; ok {
		return val, nil
	}
	return "", nil
}

func (c *cache) AddWithTTL(key string, value string, ttl time.Duration, callbackFunc CallbackFunc) {
	c.Lock()
	c.data[key] = value
	go func() {
		<-time.After(ttl)
		c.Lock()
		delete(c.data, key)
		c.Unlock()
		if callbackFunc != nil {
			callbackFunc(key, value)
		}
	}()
	c.Unlock()

}

func (c *cache) Delete(key string) {
	c.Lock()
	delete(c.data, key)
	c.Unlock()
}

func (c *cache) Keys() []string {
	c.Lock()
	defer c.Unlock()
	keys := make([]string, len(c.data))
	i := 0
	for k, _ := range c.data {
		keys[i] = k
		i++
	}
	return keys
}
