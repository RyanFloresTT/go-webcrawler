package main

import (
	"fmt"
	"net/url"
	"sort"
	"sync"
)

type config struct {
	pages              map[string]int
	baseURL            *url.URL
	mu                 *sync.Mutex
	concurrencyControl chan struct{}
	wg                 *sync.WaitGroup
	maxPages           int
}

func (cfg *config) crawlPage(rawCurrentURL string) {
	defer cfg.wg.Done()
	cfg.mu.Lock()
	if len(cfg.pages) >= cfg.maxPages {
		cfg.mu.Unlock()
		return
	}
	cfg.mu.Unlock()

	fmt.Println("Starting Crawl of : " + rawCurrentURL)

	cfg.concurrencyControl <- struct{}{}

	defer func() {
		<-cfg.concurrencyControl
	}()

	baseURL, err := url.Parse(cfg.baseURL.String())
	if err != nil || baseURL == nil {
		fmt.Printf("Error parsing base URL: %v\n", err)
		return
	}

	baseHost := baseURL.Hostname()

	currentURL, err := url.Parse(rawCurrentURL)
	if err != nil || currentURL == nil {
		fmt.Printf("Error parsing current URL: %v\n", err)
		return
	}

	currentHost := currentURL.Hostname()

	if baseHost != currentHost {
		return
	}

	normalCurrentURL, err := NormalizeURL(rawCurrentURL)
	if err != nil {
		fmt.Printf("Error normalizing URL: %v\n", err)
		return
	}

	cfg.mu.Lock()
	val, ok := cfg.pages[normalCurrentURL]
	if ok {
		cfg.pages[normalCurrentURL] = val + 1
		cfg.mu.Unlock()
		return
	}
	cfg.pages[normalCurrentURL] = 1
	cfg.mu.Unlock()

	cfg.mu.Lock()
	if len(cfg.pages) > cfg.maxPages {
		cfg.mu.Unlock()
		return
	}
	cfg.mu.Unlock()

	rawHTML, err := getHTML(normalCurrentURL)
	if err != nil {
		fmt.Printf("Error getting HTML from %s: %v\n", normalCurrentURL, err)
		return
	}

	urls, err := GetURLsFromHTML(rawHTML, cfg.baseURL.String())
	if err != nil {
		fmt.Printf("Error getting URLs from HTML: %v\n", err)
		return
	}

	for _, u := range urls {
		nextURL, err := url.Parse(u)
		if err != nil {
			fmt.Println("Error parsing URL:", u)
			continue
		}

		absoluteURL := baseURL.ResolveReference(nextURL)
		cfg.wg.Add(1)
		go cfg.crawlPage(absoluteURL.String())

	}
}

func printReport(pages map[string]int, baseURL string) {
	fmt.Println("=============================")
	fmt.Println("REPORT for " + baseURL)
	fmt.Println("=============================")

	type kv struct {
		Key   string
		Value int
	}

	var kvs []kv
	for key, value := range pages {
		kvs = append(kvs, kv{Key: key, Value: value})
	}

	sort.Slice(kvs, func(i, j int) bool {
		if kvs[i].Value == kvs[j].Value {
			return kvs[i].Key < kvs[j].Key
		}
		return kvs[i].Value > kvs[j].Value
	})

	for _, kv := range kvs {
		fmt.Printf("Found %v internal links to %s\n", kv.Value, kv.Key)
	}
}
