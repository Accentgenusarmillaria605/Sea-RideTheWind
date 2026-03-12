package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type CheckBlacklistMiddleware struct {
	Redis *redis.Redis
}

func NewCheckBlacklistMiddleware(r *redis.Redis) *CheckBlacklistMiddleware {
	return &CheckBlacklistMiddleware{
		Redis: r,
	}
}

func (m *CheckBlacklistMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next(w, r)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			next(w, r)
			return
		}
		blackListKey := fmt.Sprintf("user:jwt_blacklist:%s", token)
		exists, err := m.Redis.ExistsCtx(r.Context(), blackListKey)
		if err == nil && exists {
			w.Header().Set("Content-Type", "appliaction/json;charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"code":401, "msg":"登录已失效,请重新登陆"}`))
			return
		}
		ctx := context.WithValue(r.Context(), "jwt_token", token)
		next(w, r.WithContext(ctx))
	}
}
