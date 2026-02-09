package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

var (
	// Cache tin tức
	newsCache = make(map[string]CacheEntry)
	
	// Cache tìm kiếm địa điểm
	mapCache  = make(map[string]CacheEntry)
	
	// Mutex cho news và map search
	mutex = &sync.RWMutex{}

	// Cache ảnh bản đồ (Tile Cache)
	tileCache = make(map[string][]byte)
	tileMutex = &sync.RWMutex{}

	// --- FIREBASE CLIENT ---
	authClient *auth.Client
)

// Hàm khởi tạo kết nối Firebase
func initFirebase() {
	var opt option.ClientOption

	// 1. Ưu tiên đọc từ biến môi trường (Dùng cho Render)
	jsonKey := os.Getenv("FIREBASE_CREDENTIALS")
	if jsonKey != "" {
		log.Println("Đang sử dụng Firebase Key từ biến môi trường...")
		opt = option.WithCredentialsJSON([]byte(jsonKey))
	} else {
		// 2. Nếu không có, đọc từ file (Dùng cho Localhost)
		log.Println("Đang sử dụng Firebase Key từ file serviceAccountKey.json...")
		opt = option.WithCredentialsFile(filepath.Join(".", "serviceAccountKey.json"))
	}

	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Lỗi kết nối Firebase: %v\n", err)
	}

	client, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("Lỗi tạo Auth Client: %v\n", err)
	}

	authClient = client
	log.Println("Đã kết nối tới Firebase Auth thành công!")
}