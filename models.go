package main

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// --- STRUCTS CHO AUTH ---
type User struct {
	Username string `json:"username"`
	Password string `json:"password"` // Lưu hash, không lưu plain text
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// --- STRUCTS CHO TIN TỨC ---
type Article struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	URL         string `json:"url"`
	Image       string `json:"image"`
	PublishedAt string `json:"publishedAt"`
	Source struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"source"`
}

type GNewsResponse struct {
	TotalArticles int       `json:"totalArticles"`
	Articles      []Article `json:"articles"`
}

type FullArticleContent struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	TextContent string `json:"textContent"`
	SiteName    string `json:"siteName"`
}

// --- STRUCTS CHO BẢN ĐỒ ---
type TomTomResponse struct {
	Results []struct {
		ID    string  `json:"id"`
		Type  string  `json:"type"`
		Score float64 `json:"score"`
		Address struct {
			FreeformAddress string `json:"freeformAddress"`
			Country         string `json:"country"`
		} `json:"address"`
		Position struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"position"`
	} `json:"results"`
}

// --- STRUCT CACHE CHUNG ---
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}