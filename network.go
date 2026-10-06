package main

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// 一个全局随机源,代理轮换不再每次重置种子(原代码同秒内可能选到同一代理)
var randSrc = rand.New(rand.NewSource(time.Now().UnixNano()))
var randMu sync.Mutex

func rand_range(max int) int {
	if max <= 0 {
		return 0
	}
	randMu.Lock()
	defer randMu.Unlock()
	return randSrc.Intn(max)
}
func get_socks5_proxy() (proxy.Dialer, error) {
	// 创建一个SOCKS5代理拨号器
	frozen := app.Proxy.Sockc5
	if len(frozen) == 0 {
		return nil, fmt.Errorf("没有可用的代理")
	}
	return proxy.SOCKS5("tcp", frozen[rand_range(len(frozen))], nil, proxy.Direct)
}

var (
	clientOnce    sync.Once
	defaultClient *http.Client
)

// 复用单个 http.Client(含连接池),不再每次请求新建 TCP/TLS 握手
func get_client() *http.Client {
	if !app.Proxy.Enable || len(app.Proxy.Sockc5) == 0 {
		clientOnce.Do(func() {
			defaultClient = &http.Client{Timeout: time.Second * 60}
		})
		return defaultClient
	}
	p, err := get_socks5_proxy()
	if err != nil {
		clientOnce.Do(func() {
			defaultClient = &http.Client{Timeout: time.Second * 60}
		})
		return defaultClient
	}
	return &http.Client{
		Transport: &http.Transport{
			Dial: p.Dial,
		},
		Timeout: time.Second * 60,
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
func request_get(url string, ua string) (string, error) {
	client := get_client()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		return "", err

	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("状态码:%d", resp.StatusCode)
	}

	resp_data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(resp_data), nil
}
