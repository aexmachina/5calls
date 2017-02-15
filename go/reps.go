package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

// RepFinder provides a mechanism to find local reps given an address.
type RepFinder interface {
	GetReps(address string) (*LocalReps, *Address, error)
}

// APIError is an error returned by the Google civic API, which also
// implements the error interface.
type APIError struct {
	Code    int
	Message string
	Errors  []struct {
		Domain  string
		Reason  string
		Message string
	}
}

func (ae *APIError) Error() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%d %s", ae.Code, ae.Message)
	for _, e := range ae.Errors {
		if e.Message != ae.Message { // don't duplicate messages
			fmt.Fprintf(&buf, ";[domain=%s, reason=%s: %s]", e.Domain, e.Reason, e.Message)
		}
	}
	return buf.String()
}

// Office represents a government office.
type Office struct {
	Name            string
	DivisionID      string
	Levels          []string
	Roles           []string
	OfficialIndices []int
}

// Official represents a government official.
type Official struct {
	Name     string
	Address  []Address
	Party    string
	Phones   []string
	PhotoURL string
	Channels []struct {
		ID   string
		Type string
	}
}

// repCache implements a cache layer on top of a delegate rep finder.
type repCache struct {
	delegate RepFinder
	cache    *cache.Cache
}

type cacheItem struct {
	reps LocalReps
	addr Address
}

// NewRepCache returns a repCache value for a delegate.
func NewRepCache(delegate RepFinder, ttl time.Duration, gc time.Duration) RepFinder {
	return &repCache{
		delegate: delegate,
		cache:    cache.New(ttl, gc),
	}
}

// GetReps returns local representatives for the supplied address.
func (r *repCache) GetReps(address string) (*LocalReps, *Address, error) {
	data, ok := r.cache.Get(address)
	if ok {
		ci := data.(*cacheItem)
		reps := ci.reps
		addr := ci.addr
		return &reps, &addr, nil
	}
	reps, addr, err := r.delegate.GetReps(address)
	if err != nil {
		return nil, nil, err
	}
	ci := &cacheItem{reps: *reps, addr: *addr}
	r.cache.Set(address, ci, cache.DefaultExpiration)
	return reps, addr, nil
}
