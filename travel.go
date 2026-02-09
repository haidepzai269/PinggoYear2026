package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Handler tìm kiếm địa điểm
func getMapSearchHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	mutex.RLock()
	entry, found := mapCache[query]
	mutex.RUnlock()

	if found && time.Now().Before(entry.ExpiresAt) {
		fmt.Printf("[MAP CACHE HIT] %s\n", query)
		json.NewEncoder(w).Encode(entry.Data)
		return
	}

	fmt.Printf("[MAP MISS] Gọi TomTom Search: %s\n", query)
	results, err := fetchFromTomTom(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mutex.Lock()
	mapCache[query] = CacheEntry{Data: results, ExpiresAt: time.Now().Add(24 * time.Hour)}
	mutex.Unlock()

	json.NewEncoder(w).Encode(results)
}

// Handler lấy Tile bản đồ
func getMapTileHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}

	z := r.URL.Query().Get("z")
	x := r.URL.Query().Get("x")
	y := r.URL.Query().Get("y")

	if z == "" || x == "" || y == "" {
		http.Error(w, "Thiếu tham số z, x, y", http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("%s/%s/%s", z, x, y)

	tileMutex.RLock()
	cachedImage, found := tileCache[cacheKey]
	tileMutex.RUnlock()

	if found {
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Header().Set("Content-Type", "image/png")
		w.Write(cachedImage)
		return
	}

	tomtomURL := fmt.Sprintf("%s/%s/%s/%s.png?key=%s&tileSize=512",
		TOMTOM_TILE_URL, z, x, y, TOMTOM_API_KEY)

	resp, err := http.Get(tomtomURL)
	if err != nil {
		http.Error(w, "Lỗi kết nối TomTom: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	bodyData, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		errMsg := fmt.Sprintf("TomTom Error (%d): %s", resp.StatusCode, string(bodyData))
		fmt.Println(errMsg)
		http.Error(w, errMsg, resp.StatusCode)
		return
	}

	tileMutex.Lock()
	if len(tileCache) > 2000 {
		tileCache = make(map[string][]byte)
	}
	tileCache[cacheKey] = bodyData
	tileMutex.Unlock()

	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	w.Header().Set("Content-Type", "image/png")
	w.Write(bodyData)
}

// Handler tìm đường
func getRouteHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "OPTIONS" {
		return
	}

	points := r.URL.Query().Get("points")
	if points == "" {
		http.Error(w, "Thiếu tham số points (start:end)", http.StatusBadRequest)
		return
	}

	urlStr := fmt.Sprintf("%s/%s/json?key=%s&traffic=true", TOMTOM_ROUTE_URL, points, TOMTOM_API_KEY)
	fmt.Printf("[ROUTE API] %s\n", urlStr)

	resp, err := http.Get(urlStr)
	if err != nil {
		http.Error(w, "Lỗi kết nối TomTom", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}

func fetchFromTomTom(query string) (*TomTomResponse, error) {
	encodedQuery := url.PathEscape(query)
	urlStr := fmt.Sprintf("%s/%s.json?key=%s&countrySet=VN&limit=5&language=vi-VN",
		TOMTOM_SEARCH_URL, encodedQuery, TOMTOM_API_KEY)

	fmt.Printf("[DEBUG] Request URL: %s\n", urlStr)

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		fmt.Printf("[TOMTOM ERROR] Status: %d | Body: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("TomTom API Error: %s", string(body))
	}

	var result TomTomResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}