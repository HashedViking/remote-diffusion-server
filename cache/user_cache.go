package cache

import (
	"sync"
	"time"
)

/*
UserCache is a simple in-memory cache for storing user data.
The cache is safe for concurrent use by multiple goroutines.
Maps a user key to a last activity time.Time value.
*/
type UserCache struct {
	sync.RWMutex
	m map[string]time.Time
}

func NewUserCache() UserCache {
	return UserCache{m: make(map[string]time.Time)}
}

func (users *UserCache) Get(key string) time.Time {
	users.RLock()
	defer users.RUnlock()
	return users.m[key]
}

func (users *UserCache) Set(key string, value time.Time) {
	users.Lock()
	defer users.Unlock()
	users.m[key] = value
}

func (users *UserCache) Remove(key string) {
	users.Lock()
	defer users.Unlock()
	delete(users.m, key)
}

func (users *UserCache) Range(f func(key string, value time.Time)) {
	users.RLock()
	defer users.RUnlock()
	for k, v := range users.m {
		f(k, v)
	}
}
