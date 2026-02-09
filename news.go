package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-shiori/go-readability"
)

// Handler lấy danh sách tin
func getNewsHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}

	category := r.URL.Query().Get("category")
	if category == "" {
		category = "general"
	}

	mutex.RLock()
	entry, found := newsCache[category]
	mutex.RUnlock()

	if found && time.Now().Before(entry.ExpiresAt) {
		fmt.Printf("[CACHE HIT] Danh sách tin: %s\n", category)
		json.NewEncoder(w).Encode(entry.Data)
		return
	}

	fmt.Printf("[CACHE MISS] Gọi GNews API: %s\n", category)
	articles, err := fetchFromGNews(category)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mutex.Lock()
	newsCache[category] = CacheEntry{
		Data:      articles,
		ExpiresAt: time.Now().Add(CACHE_TTL),
	}
	mutex.Unlock()

	json.NewEncoder(w).Encode(articles)
}

// Handler cào nội dung chi tiết
func getArticleContentHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}

	articleURL := r.URL.Query().Get("url")
	if articleURL == "" {
		http.Error(w, "Missing URL parameter", http.StatusBadRequest)
		return
	}

	fmt.Printf("[SCRAPING] Đang cào nội dung từ: %s\n", articleURL)
	article, err := readability.FromURL(articleURL, 30*time.Second)
	if err != nil {
		fmt.Printf("Lỗi cào dữ liệu: %v\n", err)
		http.Error(w, "Không thể lấy nội dung bài viết", http.StatusInternalServerError)
		return
	}

	response := FullArticleContent{
		Title:       article.Title,
		Content:     article.Content,
		TextContent: article.TextContent,
		SiteName:    article.SiteName,
	}

	json.NewEncoder(w).Encode(response)
}

func fetchFromGNews(category string) ([]Article, error) {
	var targetURL string
	params := url.Values{}
	params.Add("apikey", GNEWS_API_KEY)
	params.Add("lang", "vi")

	switch category {
	case "vietnam":
		targetURL = GNEWS_BASE_URL
		params.Add("category", "nation")
		params.Add("country", "vn")
	case "world":
		targetURL = GNEWS_BASE_URL
		params.Add("category", "world")
		params.Add("country", "vn")
	case "business":
		targetURL = GNEWS_BASE_URL
		params.Add("category", "business")
		params.Add("country", "vn")
	case "science":
		targetURL = GNEWS_BASE_URL
		params.Add("category", "science")
		params.Add("country", "vn")
	case "health":
		targetURL = GNEWS_BASE_URL
		params.Add("category", "health")
		params.Add("country", "vn")
	case "sports":
		targetURL = GNEWS_BASE_URL
		params.Add("category", "sports")
		params.Add("country", "vn")
	case "entertainment":
		targetURL = GNEWS_BASE_URL
		params.Add("category", "entertainment")
		params.Add("country", "vn")
	case "education":
		targetURL = GNEWS_BASE_URL
		params.Add("country", "vn")
		params.Add("q", "trường học OR sinh viên OR giáo dục")
	case "traffic":
		targetURL = GNEWS_BASE_URL
		params.Add("country", "vn")
		params.Add("q", "xe OR giao thông OR đường")
	default:
		targetURL = GNEWS_BASE_URL
		params.Add("category", "general")
		params.Add("country", "vn")
	}

	fullURL := fmt.Sprintf("%s?%s", targetURL, params.Encode())
	fmt.Printf("[API CALL] %s\n", fullURL)

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GNews API Error: %s", string(bodyBytes))
	}

	var result GNewsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Articles, nil
}