package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
)

// GOBPATH is path to save gob file
const GOBPATH = "./cookies.gob"

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

// All show all cookies
func (c *Cookies) All() {
	for k, v := range c.entry {
		fmt.Println("Host:", k)
		for ck, cv := range v {
			fmt.Printf("[%s] %s\n", ck, cv)
		}
	}
}

// Dumps data
func (c *Cookies) Dumps() {
	if _, err := os.Stat(GOBPATH); err != nil {
		if os.IsNotExist(err) {
			os.Create(GOBPATH)
		}
	}
	file, err := os.OpenFile(GOBPATH, os.O_WRONLY, os.ModePerm)
	defer file.Close()

	if err != nil {
		log.Fatal("[Dumps]", err)
	}

	enc := gob.NewEncoder(file)
	enc.Encode(c.entry)
}

// Loads data from files
func (c *Cookies) Loads() {
	if _, err := os.Stat(GOBPATH); err != nil {
		log.Println("[Load cookies file]", err)
		return
	}
	file, err := os.OpenFile(GOBPATH, os.O_RDONLY, os.ModePerm)
	defer file.Close()

	if err != nil {
		log.Fatal("[Loads]", err)
	}

	dec := gob.NewDecoder(file)
	dec.Decode(&c.entry)
}
