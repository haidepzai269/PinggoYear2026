package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-shiori/go-readability" // Thư viện mới
)

// --- CẤU HÌNH ---
const (
	// GNEWS CONFIG
	GNEWS_API_KEY = "968a2702a428ba46862223f52f48ca56" // Key cũ của bạn
	GNEWS_BASE_URL = "https://gnews.io/api/v4/top-headlines"
	
	// TOMTOM CONFIG (BẠN CẦN THAY KEY CỦA BẠN VÀO ĐÂY)
	TOMTOM_API_KEY = "QHwbpg3Xwemf5VlooFK4bk6tQ0PYtiaf" 
	TOMTOM_SEARCH_URL = "https://api.tomtom.com/search/2/search"
// URL để lấy ảnh bản đồ (Raster Tile)
TOMTOM_TILE_URL   = "https://api.tomtom.com/map/1/tile/basic/main"

PORT      = ":8080"
CACHE_TTL = 2 * time.Hour
)

// --- STRUCTS ---
type Article struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	URL         string `json:"url"`
	Image       string `json:"image"`
	PublishedAt string `json:"publishedAt"`
	Source      struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"source"`
}

type GNewsResponse struct {
	TotalArticles int       `json:"totalArticles"`
	Articles      []Article `json:"articles"`
}
type CacheEntry struct {
	Data      interface{} // <-- Sửa thành interface{} để nhận cả Map lẫn News
	ExpiresAt time.Time
}

// Struct trả về cho Client khi cào nội dung chi tiết
type FullArticleContent struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	TextContent string `json:"textContent"`
	SiteName    string `json:"siteName"`
}
// Struct cho TomTom Search
type TomTomResponse struct {
	Results []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Score    float64 `json:"score"`
		Address  struct {
			FreeformAddress string `json:"freeformAddress"`
			Country        string `json:"country"`
		} `json:"address"`
		Position struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"position"`
	} `json:"results"`
}

// --- GLOBAL CACHE ---
var (
	// Tách cache news và map để quản lý dễ hơn
	newsCache = make(map[string]CacheEntry)
	mapCache  = make(map[string]CacheEntry) // Cache cho tìm kiếm địa điểm
	mutex     = &sync.RWMutex{}

	// 🔥 CACHE ẢNH BẢN ĐỒ (RAM) - ĐÂY LÀ PHẦN QUAN TRỌNG BẠN ĐANG THIẾU 🔥
	tileCache = make(map[string][]byte)
	tileMutex = &sync.RWMutex{}
)

// --- MIDDLEWARE ---
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// Handler 1: Lấy danh sách tin (Giữ nguyên logic cũ)
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

// Handler 2: Cào nội dung chi tiết (MỚI)
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

	// Sử dụng thư viện go-readability để lấy nội dung chính
	// Timeout 30s để tránh treo server
	article, err := readability.FromURL(articleURL, 30*time.Second)
	if err != nil {
		fmt.Printf("Lỗi cào dữ liệu: %v\n", err)
		http.Error(w, "Không thể lấy nội dung bài viết", http.StatusInternalServerError)
		return
	}

	response := FullArticleContent{
		Title:       article.Title,
		Content:     article.Content, // HTML sạch đã lọc quảng cáo
		TextContent: article.TextContent,
		SiteName:    article.SiteName,
	}

	json.NewEncoder(w).Encode(response)
}

// 3. Handler TomTom Search (MỚI - Tối ưu Quota)
func getMapSearchHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" { return }

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	// Kiểm tra Cache bản đồ
	mutex.RLock()
	entry, found := mapCache[query]
	mutex.RUnlock()

	if found && time.Now().Before(entry.ExpiresAt) {
		fmt.Printf("[MAP CACHE HIT] %s\n", query)
		json.NewEncoder(w).Encode(entry.Data)
		return
	}

	// Gọi TomTom API nếu không có cache
	fmt.Printf("[MAP MISS] Gọi TomTom Search: %s\n", query)
	results, err := fetchFromTomTom(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Lưu cache (Cache lâu hơn tin tức vì địa điểm ít thay đổi - 24h)
	mutex.Lock()
	mapCache[query] = CacheEntry{Data: results, ExpiresAt: time.Now().Add(24 * time.Hour)}
	mutex.Unlock()

	json.NewEncoder(w).Encode(results)
}

func getMapTileHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" { return }

	z := r.URL.Query().Get("z")
	x := r.URL.Query().Get("x")
	y := r.URL.Query().Get("y")

	if z == "" || x == "" || y == "" {
		http.Error(w, "Thiếu tham số", http.StatusBadRequest)
		return
	}

	// Tạo key cache
	cacheKey := fmt.Sprintf("%s/%s/%s", z, x, y)

	// 1. Kiểm tra RAM xem có ảnh chưa
	tileMutex.RLock()
	cachedImage, found := tileCache[cacheKey]
	tileMutex.RUnlock()

	if found {
		// Có rồi -> Trả luôn
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Header().Set("Content-Type", "image/png")
		w.Write(cachedImage)
		return
	}

	// 2. Chưa có -> Gọi TomTom
	tomtomURL := fmt.Sprintf("%s/%s/%s/%s.png?key=%s&tileSize=512&view=Unified&language=vi-VN",
		TOMTOM_TILE_URL, z, x, y, TOMTOM_API_KEY)

	resp, err := http.Get(tomtomURL)
	if err != nil {
		http.Error(w, "Lỗi kết nối TomTom", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		http.Error(w, "TomTom Error", resp.StatusCode)
		return
	}

	imgData, _ := io.ReadAll(resp.Body)

	// 3. Lưu vào RAM
	tileMutex.Lock()
	// Nếu RAM đầy (2000 ảnh) thì xóa bớt đi
	if len(tileCache) > 2000 {
		tileCache = make(map[string][]byte)
	}
	tileCache[cacheKey] = imgData
	tileMutex.Unlock()

	// 4. Trả về Client
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	w.Header().Set("Content-Type", "image/png")
	w.Write(imgData)
}

func fetchFromTomTom(query string) (*TomTomResponse, error) {
	// Encode query (TomTom thích %20 hơn dấu +)
	encodedQuery := url.PathEscape(query) 
	
	// Tạo URL
	urlStr := fmt.Sprintf("%s/%s.json?key=%s&countrySet=VN&limit=5&language=vi-VN", 
		TOMTOM_SEARCH_URL, encodedQuery, TOMTOM_API_KEY)

	fmt.Printf("[DEBUG] Request URL: %s\n", urlStr) // In URL ra để kiểm tra

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Đọc body trả về dù thành công hay thất bại
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		// In lỗi chi tiết từ TomTom ra Terminal của Go
		fmt.Printf("[TOMTOM ERROR] Status: %d | Body: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("TomTom API Error: %s", string(body))
	}

	var result TomTomResponse
	if err := json.Unmarshal(body, &result); err != nil { // Dùng Unmarshal an toàn hơn
		return nil, err
	}
	return &result, nil
}

func fetchFromGNews(category string) ([]Article, error) {
	var targetURL string
	params := url.Values{}
	params.Add("apikey", GNEWS_API_KEY)
	params.Add("lang", "vi") // Luôn tìm tiếng Việt
	
	// Lưu ý: Mặc định không Add country=vn ngay, chỉ Add cho các mục Category chuẩn
	
	switch category {
	case "vietnam":
		targetURL = GNEWS_BASE_URL
		params.Add("category", "nation")
		params.Add("country", "vn") // Chỉ mục Việt Nam mới ép buộc country VN
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
		// Thay đổi chiến thuật: Dùng top-headlines (tin nóng) + lọc từ khóa
		targetURL = GNEWS_BASE_URL 
		params.Add("country", "vn") // Quan trọng: Ép tìm nguồn Việt Nam
		params.Add("q", "trường học OR sinh viên OR giáo dục") // Từ khóa phổ biến hơn
	
	case "traffic":
		targetURL = GNEWS_BASE_URL
		params.Add("country", "vn")
		params.Add("q", "xe OR giao thông OR đường") // "xe" và "đường" xuất hiện trong hầu hết tin giao thông

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

func main() {
	http.HandleFunc("/news", getNewsHandler)
	http.HandleFunc("/article", getArticleContentHandler)
	
	// API Mới
	http.HandleFunc("/travel/search", getMapSearchHandler)
	http.HandleFunc("/tiles", getMapTileHandler)
	fmt.Printf("Backend Go đang chạy tại http://localhost%s\n", PORT)
	log.Fatal(http.ListenAndServe(PORT, nil))
}