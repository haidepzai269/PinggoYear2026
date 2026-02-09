package main

import (
	"os"
	"time"
)


const (
	// GNEWS CONFIG
	GNEWS_API_KEY  = "968a2702a428ba46862223f52f48ca56"
	GNEWS_BASE_URL = "https://gnews.io/api/v4/top-headlines"

	// TOMTOM CONFIG
	TOMTOM_API_KEY    = "QHwbpg3Xwemf5VlooFK4bk6tQ0PYtiaf"
	TOMTOM_SEARCH_URL = "https://api.tomtom.com/search/2/search"
	TOMTOM_TILE_URL   = "https://api.tomtom.com/map/1/tile/basic/main"
	TOMTOM_ROUTE_URL  = "https://api.tomtom.com/routing/1/calculateRoute"
	
	
	CACHE_TTL = 2 * time.Hour
	// JWT CONFIG (Bí mật)
	JWT_SECRET        = "pinggo_super_secret_key_2026" // Nên đổi chuỗi này phức tạp hơn
	ACCESS_TOKEN_TTL  = 15 * time.Minute               // Hết hạn nhanh
	REFRESH_TOKEN_TTL = 7 * 24 * time.Hour
)

var PORT = ":8080" // Đổi thành biến var để có thể thay đổi

func init() {
	// Nếu Render cung cấp PORT thì dùng, không thì dùng 8080
	envPort := os.Getenv("PORT")
	if envPort != "" {
		PORT = ":" + envPort
	}
}