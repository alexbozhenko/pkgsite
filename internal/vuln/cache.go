// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vuln

import (
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	vulnc "golang.org/x/vuln/client"
	"golang.org/x/vuln/osv"
)

// cache implements the golang.org/x/vulndb/client.Cache interface. It
// stores in memory the index and a limited number of path entries for one DB.
type cache struct {
	mu         sync.Mutex
	dbName     string // support only one DB
	index      vulnc.DBIndex
	retrieved  time.Time
	entryCache *lru.Cache
}

func newCache() *cache {
	const size = 100
	ec, err := lru.New(size)
	if err != nil {
		// Can only happen if size is bad, and we control it.
		panic(err)
	}
	return &cache{entryCache: ec}
}

// ReadIndex returns the index for dbName from the cache, or returns zero values
// if it is not present.
func (c *cache) ReadIndex(dbName string) (vulnc.DBIndex, time.Time, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkDB(dbName); err != nil {
		return nil, time.Time{}, err
	}
	return c.index, c.retrieved, nil
}

// WriteIndex puts the index and retrieved time into the cache.
func (c *cache) WriteIndex(dbName string, index vulnc.DBIndex, retrieved time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkDB(dbName); err != nil {
		return err
	}
	c.index = index
	c.retrieved = retrieved
	return nil
}

// ReadEntries returns the vulndb entries for path from the cache, or
// nil if not prsent.
func (c *cache) ReadEntries(dbName, path string) ([]*osv.Entry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkDB(dbName); err != nil {
		return nil, err
	}
	if entries, ok := c.entryCache.Get(path); ok {
		return entries.([]*osv.Entry), nil
	}
	return nil, nil
}

// WriteEntries puts the entries for path into the cache.
func (c *cache) WriteEntries(dbName, path string, entries []*osv.Entry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.checkDB(dbName); err != nil {
		return err
	}
	c.entryCache.Add(path, entries)
	return nil
}

func (c *cache) checkDB(name string) error {
	if c.dbName == "" {
		c.dbName = name
		return nil
	}
	if c.dbName != name {
		return fmt.Errorf("vulndbCache: called with db name %q, expected %q", name, c.dbName)
	}
	return nil
}