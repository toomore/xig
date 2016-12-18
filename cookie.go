package main

import (
	"log"
	"net/http"
	"net/url"
	"sync"
)

// Cookies struct
type Cookies struct {
	entry map[string]map[string]*http.Cookie
	mu    sync.Mutex
}

// NewCookies a Cookies
func NewCookies() *Cookies {
	return &Cookies{
		entry: make(map[string]map[string]*http.Cookie),
	}
}

// SetCookies set cookies
func (c *Cookies) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.entry[u.Host]; !ok {
		log.Println("[Cookies] New host:", u.Host)
		c.entry[u.Host] = make(map[string]*http.Cookie)
	}
	hostCookie := c.entry[u.Host]
	for _, v := range cookies {
		hostCookie[v.Name] = v
	}
}

// Cookies get cookies
func (c *Cookies) Cookies(u *url.URL) []*http.Cookie {
	c.mu.Lock()
	defer c.mu.Unlock()

	if hostCookie, ok := c.entry[u.Host]; ok {
		if len(hostCookie) > 0 {
			result := make([]*http.Cookie, len(hostCookie))
			var i int
			for _, v := range hostCookie {
				result[i] = v
				i++
			}
			return result
		}
	}
	return nil
}
