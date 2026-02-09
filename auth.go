package main

import (
	"context"
	"net/http"
	"strings"
)

// --- MIDDLEWARE FIREBASE ---

// AuthMiddleware chặn request nếu không có Firebase Token hợp lệ
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w) // Đảm bảo CORS header luôn có
		if r.Method == "OPTIONS" {
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization Header", http.StatusUnauthorized)
			return
		}

		// Lấy token từ chuỗi "Bearer <token>"
		idToken := strings.TrimPrefix(authHeader, "Bearer ")
		
		// Xác thực token với Firebase Server
		_, err := authClient.VerifyIDToken(context.Background(), idToken) 
		if err != nil {
			http.Error(w, "Invalid or Expired Token", http.StatusUnauthorized)
			return
		}

		// (Tùy chọn) Có thể lấy thông tin user từ token để dùng sau này
		// fmt.Printf("User ID: %s đang gọi API\n", token.UID)

		// Token hợp lệ, cho phép đi tiếp
		next(w, r)
	}
}