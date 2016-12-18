package main

import (
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	cookieJar = NewCookies()

	filterV = regexp.MustCompile(`<script type="text/javascript">window._sharedData = (.+);</script>`)
	sizeR   = regexp.MustCompile(`/[a-z][0-9]+x[0-9]+`)

	delay     = flag.Int64("d", 0, "Delay to start")
	finddel   = flag.Bool("f", false, "Find deleted")
	getAll    = flag.Bool("a", false, "Get all data")
	loginuser = flag.Bool("u", false, "Login someone to see private data")
	ncpu      = flag.Int("c", runtime.NumCPU()*20, "concurrency nums")
	qLook     = flag.Bool("i", false, "Quick look recently data")
)

func fetch(user string) *http.Response {
	log.Printf("Fetch data from `%s`\n", user)
	client := &http.Client{
		Jar: cookieJar,
	}
	resp, err := client.Get(fmt.Sprintf(`https://www.instagram.com/%s/?hl=zh-tw`, user))
	if err != nil {
		log.Fatal(err)
	}
	return resp
}

func filter1(html io.Reader) []byte {
	log.Println("Find json data ...")
	data, err := ioutil.ReadAll(html)
	if err != nil {
		log.Fatal(err)
	}
	if filterV.Match(data) {
		log.Println("Finded!!")
		for _, result := range filterV.FindAllSubmatch(data, -1) {
			return result[1]
		}
	}
	return nil
}

func downloadNodeImage(node Node, user string, wg *sync.WaitGroup) {
	runtime.Gosched()
	defer wg.Done()

	path := sizeR.ReplaceAllString(node.DisplaySrc, "")
	url, err := url.Parse(path)
	if err != nil {
		log.Fatal(err)
	}
	err = downloadAndSave(path,
		fmt.Sprintf("./%s/img/%s_%%x%s", user, node.Code, filepath.Ext(url.Path)),
		true,
	)

	if err != nil {
		log.Fatal(err)
	}
	log.Println(fmt.Sprintf("Saved `%s`, `%s`", node.Code, node.DisplaySrc))
}

func downloadAvatar(user string, path string, wg *sync.WaitGroup) {
	defer wg.Done()

	path = sizeR.ReplaceAllString(path, "")
	url, err := url.Parse(path)
	if err != nil {
		log.Fatal(err)
	}
	err = downloadAndSave(path,
		fmt.Sprintf("./%s/avatar/%s_%%x%s", user, user, filepath.Ext(url.Path)),
		true,
	)

	if err != nil {
		log.Fatal(err)
	}
	log.Println(fmt.Sprintf("Saved avatar `%s`, `%s`", user, path))
}

func downloadAndSave(url string, path string, withHex bool) error {
	data, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(data.Body)

	data.Body.Close()

	if err != nil {
		log.Println(err)
		return err
	}
	if withHex {
		path = fmt.Sprintf(path, md5.Sum(body))
	}
	if _, err := os.Stat(path); err == nil {
		log.Println("File existed:", path)
		return nil
	}
	log.Printf("[O] Save `%s`\n", path)
	return ioutil.WriteFile(path, body, 0644)
}

func fetchAll(id string, username string, endCursor string, count int) {
	v := url.Values{}
	v.Set("q", fmt.Sprintf(`ig_user(%s) { media.after(%s, %d) {
  count,
  nodes {
    caption,
    code,
    comments {
      count
    },
    comments_disabled,
    date,
    dimensions {
      height,
      width
    },
    display_src,
    id,
    is_video,
    likes {
      count
    },
    owner {
      id
    },
    thumbnail_src,
    video_views
  },
  page_info
}
 }`, id, endCursor, count))
	v.Set("ref", "users::show")

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   0,
				KeepAlive: 0,
			}).DialContext,
			TLSHandshakeTimeout: 1 * time.Second,
		},
		Jar: cookieJar,
	}

	req, err := http.NewRequest("POST", "https://www.instagram.com/query/", strings.NewReader(v.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", fmt.Sprintf("https://www.instagram.com/%s/", username))

	u, _ := url.Parse("https://www.instagram.com/")
	for _, v := range cookieJar.Cookies(u) {
		if v.Name == "csrftoken" {
			req.Header.Set("x-csrftoken", v.Value)
			break
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var data = &queryData{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Fatal(err)
	}

	var wg = &sync.WaitGroup{}
	wg.Add(len(data.Media.Nodes) * 2)

	queueImg := make(chan Node, *ncpu)
	queueCon := make(chan Node, *ncpu)

	for _, node := range data.Media.Nodes {
		go func(node Node) {
			queueImg <- node
			queueCon <- node
		}(node)
	}

	go func() {
		for node := range queueImg {
			downloadNodeImage(node, username, wg)
		}
	}()
	go func() {
		for node := range queueCon {
			saveNodeContent(node, username, wg)
		}
	}()

	wg.Wait()
	close(queueImg)
	close(queueCon)
}

func saveNodeContent(node Node, user string, wg *sync.WaitGroup) {
	runtime.Gosched()
	defer wg.Done()

	jsonStr, err := json.Marshal(node)
	if err != nil {
		log.Fatal(err)
	}
	basePath := fmt.Sprintf("./%s/content/%d_%s_%s.%%s", user, node.Date, node.Code, node.ID)
	if _, err := os.Stat(fmt.Sprintf(basePath, "json")); err != nil {
		if err := ioutil.WriteFile(fmt.Sprintf(basePath, "json"), jsonStr, 0644); err != nil {
			log.Fatal(err)
		}
		log.Printf("[O] Save content.json `%s`\n", node.Code)
	} else {
		log.Println("Content `json` existed", node.Code)
	}

	if _, err := os.Stat(fmt.Sprintf(basePath, "txt")); err != nil {
		ioutil.WriteFile(fmt.Sprintf(basePath, "txt"),
			[]byte(
				fmt.Sprintf("Code: %s\nCaption: %s\nDate: %s\nDisplaySrc: %s\nID: %s",
					node.Code, node.Caption, time.Unix(int64(node.Date), 0).Format(time.RFC3339), node.DisplaySrc, node.ID)),
			0644)
		log.Printf("[O] Save content.txt `%s`\n", node.Code)
	} else {
		log.Println("Content `txt` existed", node.Code)
	}
}

func saveBiography(data profile, wg *sync.WaitGroup) {
	defer wg.Done()

	text := fmt.Sprintf("Username: %s\nFullName: %s\nID: %s\nIsPrivate: %t\nProfilePicURLHd: %s\nFollows: %d\nFollowsBy: %d\nBio: %s",
		data.Username, data.FullName, data.ID, data.IsPrivate, data.ProfilePicURLHd,
		data.Follows.Count, data.FollowedBy.Count, data.Biography)

	hex := md5.Sum([]byte(text))
	ioutil.WriteFile(fmt.Sprintf("./%s/profile/%s_%x.txt", data.Username, data.Username, hex), []byte(text), 0644)
	log.Printf("Save profile `%s` `%x`\n", data.Username, hex)
}

func fetchRecently(username string) *IGData {
	// Get nodes
	fetchData := fetch(username)
	defer fetchData.Body.Close()

	var data = &IGData{}
	if err := json.Unmarshal(filter1(fetchData.Body), &data); err != nil {
		log.Fatal(err)
	}
	return data
}

func start(user string) {
	prepareBox(user)
	data := fetchRecently(user)

	var wg = &sync.WaitGroup{}
	UserData := data.EntryData.ProfilePage[0].User

	wg.Add(len(UserData.Media.Nodes)*2 + 2)

	// Get avatar
	go downloadAvatar(user, UserData.ProfilePicURLHd, wg)
	go saveBiography(UserData, wg)

	queueImg := make(chan Node, *ncpu)
	queueCon := make(chan Node, *ncpu)
	for _, node := range UserData.Media.Nodes {
		go func(node Node) {
			queueImg <- node
			queueCon <- node
		}(node)
	}

	go func() {
		for node := range queueImg {
			go downloadNodeImage(node, user, wg)
		}
	}()
	go func() {
		for node := range queueCon {
			go saveNodeContent(node, user, wg)
		}
	}()

	wg.Wait()
	close(queueImg)
	close(queueCon)

	if *getAll {
		log.Println("Get all data!!!!")
		fetchAll(UserData.ID, UserData.Username, UserData.Media.PageInfo.EndCursor, UserData.Media.Count)
	}

	fmt.Println("Username:", UserData.Username)
	fmt.Println("Count:", UserData.Media.Count)
}

func quickLook(username string) {
	data := fetchRecently(username)
	UserData := data.EntryData.ProfilePage[0].User
	for i := len(UserData.Media.Nodes) - 1; i >= 0; i-- {
		node := UserData.Media.Nodes[i]
		fmt.Printf(`+----------------------------------------------------+
Code: https://www.instagram.com/p/%s
Date: %s IsVideo: %t
Caption: %s
DisplaySrc: %s
`,
			node.Code, time.Unix(int64(node.Date), 0).Format(time.RFC3339),
			node.IsVideo, node.Caption, node.DisplaySrc)
	}
}

func prepareBox(user string) {
	for _, path := range [5]string{"", "/img", "/avatar", "/content", "/profile"} {
		if err := os.Mkdir(fmt.Sprintf("./%s%s", user, path), 0755); err != nil {
			log.Println(err)
		}
	}
}

func findContentJSON(username string) {
	allJSON, err := filepath.Glob(fmt.Sprintf("./%s/content/*.json", username))
	if err != nil {
		log.Fatalln(err)
	}
	var wg sync.WaitGroup
	limit := make(chan struct{}, *ncpu)
	result := make([]string, len(allJSON))

	wg.Add(len(allJSON))
	starttime := time.Now()
	for i, path := range allJSON {
		go func(i int, path string) {
			defer wg.Done()
			limit <- struct{}{}

			data, err := ioutil.ReadFile(path)
			if err != nil {
				log.Println("Open files:", err)
				return
			}
			var node Node
			json.Unmarshal(data, &node)

			resp, err := http.Get(fmt.Sprintf("https://www.instagram.com/p/%s", node.Code))
			if err == nil {
				if resp.StatusCode > 300 || resp.StatusCode < 200 {
					result[i] = fmt.Sprintf("%d => %d %s", resp.StatusCode, node.Date, node.Code)
					fmt.Printf("%s", "x")
				} else {
					fmt.Printf("%s", ".")
				}
			} else {
				fmt.Printf("%s", "!")
				result[i] = fmt.Sprintf("[Err] %s => %s", node.Code, err)
			}

			<-limit
		}(i, path)
	}
	wg.Wait()
	done := time.Since(starttime)
	fmt.Println()
	var num int
	for _, v := range result {
		if v != "" {
			fmt.Println(num, v)
			num++
		}
	}
	log.Println("Done", done)
}

func login() {
	log.Println("Login user ...")
	client := &http.Client{
		Jar: cookieJar,
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
	data.Set("username", os.Getenv("IGUSER"))
	data.Set("password", os.Getenv("IGPASS"))
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

func main() {
	flag.Parse()
	if len(flag.Args()) > 0 {
		if *loginuser {
			login()
		}

		switch {
		case *finddel:
			log.Println("To find deleted", flag.Arg(0))
			findContentJSON(flag.Arg(0))
		case *qLook:
			quickLook(flag.Arg(0))
		default:
			log.Printf("Delay: %ds", *delay)
			time.Sleep(time.Duration(*delay) * time.Second)
			start(flag.Arg(0))
		}
	} else {
		fmt.Println("xig [options] {username}")
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
}
