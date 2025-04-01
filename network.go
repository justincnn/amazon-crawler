package main

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

func rangdom_range(max int) int {
	rand.NewSource(time.Now().UnixNano())
	return rand.Intn(max)
}
func get_socks5_proxy() (proxy.Dialer, error) {
	// 创建一个SOCKS5代理拨号器
	len := len(app.Proxy.Sockc5)
	if len == 0 {
		return nil, fmt.Errorf("没有可用的代理")
	}
	return proxy.SOCKS5("tcp", app.Proxy.Sockc5[rangdom_range(len)], nil, proxy.Direct)
}
func get_client() http.Client {

	proxy, err := get_socks5_proxy()
	if err != nil {
		return http.Client{Timeout: time.Second * 60}
	}
	if app.Proxy.Enable {
		return http.Client{
			Transport: &http.Transport{
				Dial: proxy.Dial,
			},

			Timeout: time.Second * 60,
		}
	} else {
		return http.Client{Timeout: time.Second * 60}
	}
}

func telnet(ip string) bool {
	conn, err := net.DialTimeout("tcp", ip, 5*time.Second)
	if err != nil {
		return false
	} else {
		if conn != nil {
			_ = conn.Close()
			return true
		} else {
			return false
		}
	}
}

// 请求HTTP GET方法，增加随机用户代理
func request_get(url, cookie string) (string, error) {
	// 随机User-Agent列表
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36",
	}
	
	// 随机选择一个User-Agent
	randomUserAgent := userAgents[rand.Intn(len(userAgents))]
	
	log.Infof("GET:%s", url)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	
	// 设置随机请求头，模拟真实浏览器
	req.Header.Set("User-Agent", randomUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Cache-Control", "max-age=0")
	
	// 设置随机引用页面，提高真实性
	referers := []string{
		"https://www.google.com/",
		"https://www.bing.com/",
		"https://www.yahoo.com/",
		"https://www.amazon.com/",
		"",  // 有时不设置引用页
	}
	if rand.Intn(5) > 0 { // 80%概率设置引用页
		req.Header.Set("Referer", referers[rand.Intn(len(referers))])
	}
	
	if len(cookie) > 0 {
		req.Header.Set("Cookie", cookie)
	}
	
	client := get_client()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		return "", ERROR_NOT_404
	}
	if resp.StatusCode == 503 {
		return "", ERROR_NOT_503
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	// 模拟人类阅读页面的行为，添加随机延迟
	// 根据页面长度计算一个合理的"阅读时间"
	readingTime := 3 + rand.Intn(5) + (len(body) / 50000)  // 基础时间 + 内容长度因子
	log.Infof("模拟阅读页面，停留 %d 秒", readingTime)
	time.Sleep(time.Duration(readingTime) * time.Second)
	
	return string(body), nil
}
