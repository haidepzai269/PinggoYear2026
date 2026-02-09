package main

import (
	"fmt"
	"log"
	"net/http"
)

// Middleware CORS dùng chung
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func main() {
	// 1. Khởi tạo Firebase trước khi chạy server
	initFirebase()

	// 2. Định nghĩa Route

	// Route News (Đã bảo vệ bằng Firebase)
	http.HandleFunc("/news", AuthMiddleware(getNewsHandler))
	http.HandleFunc("/article", AuthMiddleware(getArticleContentHandler))

	// Route Map (Đã bảo vệ bằng Firebase)
	http.HandleFunc("/travel/search", AuthMiddleware(getMapSearchHandler))
	http.HandleFunc("/tiles", getMapTileHandler)
	http.HandleFunc("/travel/route", AuthMiddleware(getRouteHandler))

	// Lưu ý: Không cần route /auth/login hay /register nữa
	// vì Frontend sẽ làm việc đó trực tiếp với Google Firebase

	fmt.Printf("Backend Go đang chạy tại http://localhost%s\n", PORT)
	log.Fatal(http.ListenAndServe(PORT, nil))
}