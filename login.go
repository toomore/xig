package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func login(cj http.CookieJar, user, pass string) {
	log.Println("Login user ...")
	client := &http.Client{
		Jar: cj,
	}
	fq, _ := client.Get("https://www.instagram.com/")
	var csrftoken string
	for _, v := range fq.Cookies() {
		if v.Name == "csrftoken" {
			csrftoken = v.Value
			break
		}
	}
	data := url.Values{}
	data.Set("username", user)
	data.Set("password", pass)
	req, _ := http.NewRequest(
		"POST",
		"https://www.instagram.com/accounts/login/ajax/",
		strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://www.instagram.com")
	req.Header.Set("Referer", "https://www.instagram.com/")
	req.Header.Set("User-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.28 Safari/537.36")
	req.Header.Set("x-csrftoken", csrftoken)
	//req.Header.Set("x-instagram-ajax", "1")
	//req.Header.Set("x-requested-with", "XMLHttpRequest")
	u, _ := url.Parse("https://www.instagram.com/")
	resp, _ := client.Do(req)
	text, _ := ioutil.ReadAll(resp.Body)
	log.Printf("Response: %s\n", text)
	resp.Body.Close()

	for _, v := range client.Jar.Cookies(u) {
		log.Println("Cookie:", v)
	}
}
