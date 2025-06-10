package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	IPSetCount int    `json:"ip_set_count" env:"IP_SET_COUNT"`
	APIKey     string `json:"api_key" env:"API_KEY"`
	APISecret  string `json:"api_secret" env:"API_SECRET"`
	API        string `json:"api" env:"API"`
	SiteDomain string `json:"site_domain" env:"SITE_DOMAIN"`
}

type Site struct {
	ID     int    `json:"id"`
	Domain string `json:"domain"`
	// Add other fields as needed
}

type Backend struct {
	RowKey int    `json:"_rowKey"`
	State  string `json:"state"`
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
	Index  int    `json:"_index"`
}

type SiteData struct {
	Data struct {
		Backend string `json:"backend"`
	} `json:"data"`
}

type UpdateRequest struct {
	Backend []Backend `json:"backend"`
}

func loadConfigFromEnv() (*Config, error) {
	config := &Config{}

	// 从环境变量读取IPSetCount
	if val := os.Getenv("IP_SET_COUNT"); val != "" {
		count, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid IP_SET_COUNT: %v", err)
		}
		config.IPSetCount = count
	} else {
		config.IPSetCount = 2 // 默认值
	}

	config.APIKey = os.Getenv("API_KEY")
	config.APISecret = os.Getenv("API_SECRET")
	config.API = os.Getenv("API")
	config.SiteDomain = os.Getenv("SITE_DOMAIN")

	// 验证必要参数
	if config.APIKey == "" || config.APISecret == "" {
		return nil, fmt.Errorf("API_KEY and API_SECRET are required")
	}

	return config, nil
}

func loadConfig(path *string) (*Config, error) {
	// 1. 尝试从环境变量加载
	if config, err := loadConfigFromEnv(); err == nil && config.APIKey != "" {
		return config, nil
	}

	// 2. 尝试从配置文件加载
	if path == nil || *path == "" {
		return nil, fmt.Errorf("failed to load configuration from environment or file")
	}

	if _, err := os.Stat(*path); err == nil {
		if config, err := loadConfigFromFile(*path); err == nil {
			return config, nil
		}
	}

	// 3. 返回默认配置
	return nil, fmt.Errorf("failed to load configuration from environment or file")
}

func loadConfigFromFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func getRealIP() (string, error) {
	// 创建自定义Transport禁用连接复用
	tr := &http.Transport{
		DisableKeepAlives: true,
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get("https://ip.3322.net")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

func getIPSet(count int) (map[string]bool, error) {
	ipSet := make(map[string]bool)
	for i := 0; i < 30; i++ {
		ip, err := getRealIP()
		fmt.Println(ip)
		if err != nil {
			return nil, err
		}
		ipSet[ip] = true
		if len(ipSet) >= count {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return ipSet, nil
}

func checkMultiIP(siteID int, ipSet map[string]bool, headers map[string]string, api string) (bool, error) {
	url := fmt.Sprintf("%s/v1/sites/%d", api, siteID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var siteData SiteData
	err = json.NewDecoder(resp.Body).Decode(&siteData)
	if err != nil {
		return false, err
	}

	var backends []Backend
	err = json.Unmarshal([]byte(siteData.Data.Backend), &backends)
	if err != nil {
		return false, err
	}

	if len(ipSet) != len(backends) {
		return false, nil
	}

	for _, backend := range backends {
		if !ipSet[strings.TrimSpace(backend.Addr)] {
			return false, nil
		}
	}

	return true, nil
}

func updateMultiIP(siteID int, ipSet map[string]bool, headers map[string]string, api string) error {
	url := fmt.Sprintf("%s/v1/sites/%d", api, siteID)

	var backends []Backend
	for ip := range ipSet {
		backends = append(backends, Backend{
			RowKey: 21,
			State:  "up",
			Addr:   ip,
			Weight: 1,
			Index:  0,
		})
	}

	data := UpdateRequest{
		Backend: backends,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// You can process the response if needed
	// var result map[string]interface{}
	// json.NewDecoder(resp.Body).Decode(&result)

	return nil
}

func getSites(headers map[string]string, api string) ([]Site, error) {
	url := fmt.Sprintf("%s/v1/sites", api)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []Site `json:"data"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func getSite(sites []Site, domain string) *Site {
	for _, site := range sites {
		if strings.Contains(site.Domain, domain) {
			return &site
		}
	}
	return nil
}

func main() {
	configPath := flag.String("c", "config.json", "配置文件路径")
	flag.Parse()
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 更新headers
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3",
		"api-key":    config.APIKey,
		"api-secret": config.APISecret,
	}

	sites, err := getSites(headers, config.API)
	if err != nil {
		log.Fatal(err)
	}

	site := getSite(sites, config.SiteDomain)
	if site == nil {
		log.Fatalf("找不到匹配的站点: %s", config.SiteDomain)
	}

	ipSet, err := getIPSet(config.IPSetCount)
	if err != nil {
		log.Fatal(err)
	}

	ok, err := checkMultiIP(site.ID, ipSet, headers, config.API)
	if err != nil {
		log.Fatal(err)
	}

	if !ok {
		if err := updateMultiIP(site.ID, ipSet, headers, config.API); err != nil {
			log.Fatal(err)
		}
		log.Println("IP地址更新成功")
	} else {
		log.Println("IP地址无需更新")
	}
}
