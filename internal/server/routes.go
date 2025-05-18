package server

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"travel-ar-backend/internal/model"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/markbates/goth/gothic"
)

// 假设你有一个 JWT secret
var jwtSecret = []byte("your_secret_key")

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api", s.HelloWorldHandler)
	r.Get("/api/health", s.healthHandler)

	r.Get("/api/auth/{provider}", s.beginAuthProviderCallback)
	r.Get("/api/auth/{provider}/callback", s.getAuthCallbackFunction)
	r.Get("/api/me", s.MeHandler)

	r.Post("/api/logout", s.LogoutHandler)
	return r
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, _ := json.Marshal(s.db.Health())
	_, _ = w.Write(jsonResp)
}

func (s *Server) getAuthCallbackFunction(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	fmt.Println("provider: ", provider)
	r = r.WithContext(context.WithValue(context.Background(), "provider", provider))

	// 这里拿到的就是model.User 结构数据
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		fmt.Fprintln(w, "auth error:", err)
		return
	}

	fmt.Println(user)

	// 1. 查找用户
	userInDB, err := s.db.GetUserByEmail(user.Email)
	if err == sql.ErrNoRows {
		// 2. 不存在则插入
		userInDB, err = s.db.CreateUser(model.User{
			Email:    user.Email,
			GoogleID: user.UserID,
			Name:     user.Name,
			Avatar:   user.AvatarURL,
			Provider: "google",
			Status:   "active",
		})
	} else {
		// 3. 存在则更新
		s.db.UpdateUserGoogleInfo(userInDB.UserID, user.UserID, user.AvatarURL)
	}

	// 生成 JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userInDB.UserID,
		"email":   userInDB.Email,
		"name":    userInDB.Name,
		"avatar":  userInDB.Avatar,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	// 设置 Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // 本地开发用 false，生产环境要 true
		SameSite: http.SameSiteLaxMode,
	})
	fmt.Println("token: ", tokenString)
	// 重定向到前端
	http.Redirect(w, r, "http://localhost:5173", http.StatusFound)
}

func (s *Server) beginAuthProviderCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	r = r.WithContext(context.WithValue(context.Background(), "provider", provider))
	gothic.BeginAuthHandler(w, r)
}

func (s *Server) MeHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 从 Cookie 读取 token
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tokenStr := cookie.Value

	// 2. 解析 JWT
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 3. 获取 claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, ok := claims["user_id"].(float64) // jwt 库会把 int 转成 float64
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 4. 查数据库
	user, err := s.db.GetUserByID(int(userID))
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// 5. 返回用户信息
	resp := map[string]interface{}{
		"user_id":  user.UserID,
		"email":    user.Email,
		"name":     user.Name,
		"avatar":   user.Avatar,
		"provider": user.Provider,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// 让 token 立即过期
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false, // 生产环境要 true
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"logged out"}`))
}

// func (s *Server) SaveRefreshToken(userID int, refreshToken string, expiresAt time.Time) error {
// 	query := `
//         INSERT INTO refresh_tokens (user_id, refresh_token, expires_at)
//         VALUES ($1, $2, $3)
//     `
// 	_, err := s.db.Exec(query, userID, refreshToken, expiresAt)
// 	return err
// }

// func (s *Server) GetRefreshToken(token string) (*model.RefreshToken, error) {
// 	var rt model.RefreshToken
// 	query := `
//         SELECT token_id, user_id, refresh_token, expires_at, created_at, revoked
//         FROM refresh_tokens
//         WHERE refresh_token = $1 AND revoked = FALSE
//     `
// 	err := s.db.QueryRow(query, token).Scan(
// 		&rt.TokenID, &rt.UserID, &rt.RefreshToken, &rt.ExpiresAt, &rt.CreatedAt, &rt.Revoked,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &rt, nil
// }

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
