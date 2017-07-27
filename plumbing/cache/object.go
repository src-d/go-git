package cache

import (
	"github.com/golang/groupcache/lru"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

const (
	MaxSize = 96 * MiByte
)

type ObjectLRU struct {
	// Max file size allowed into this cache
	MaxSize FileSize

	cache      *lru.Cache
	actualSize FileSize
}

// NewObjectLRU returns an Object cache that keeps the most used objects that fit
// into the specific memory size
func NewObjectLRU(size FileSize) *ObjectLRU {
	olru := &ObjectLRU{
		MaxSize: size,
	}

	lc := lru.New(0)
	lc.OnEvicted = func(key lru.Key, value interface{}) {
		obj := getAsObject(value)
		olru.actualSize -= FileSize(obj.Size())
	}
	olru.cache = lc

	return olru
}

func getAsObject(value interface{}) plumbing.EncodedObject {
	v, ok := value.(plumbing.EncodedObject)
	if !ok {
		panic("unreachable")
	}

	return v
}

// Add adds a new object to the cache. If the object size is greater than the
// cache size, the less used objects will be discarded until the cache fits into
// the specified size again.
func (c *ObjectLRU) Add(o plumbing.EncodedObject) {
	// if the size of the object is bigger or equal than the cache size,
	// skip it
	if FileSize(o.Size()) >= c.MaxSize {
		return
	}

	oldLen := c.cache.Len()
	c.cache.Add(o.Hash(), o)
	if oldLen < c.cache.Len() {
		c.actualSize += FileSize(o.Size())
	}

	if c.actualSize > c.MaxSize {
		for {
			if c.actualSize <= c.MaxSize {
				break
			}

			c.cache.RemoveOldest()
		}
	}
}

// Get returns an object by his hash. If the object is not found in the cache, it
// returns nil
func (c *ObjectLRU) Get(k plumbing.Hash) plumbing.EncodedObject {
	val, ok := c.cache.Get(k)
	if !ok {
		return nil
	}

	return getAsObject(val)
}

// Clear the content of this object cache
func (c *ObjectLRU) Clear() {
	c.cache.Clear()
}
