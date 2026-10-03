package utilities

import (
	"context"
	"net/http"
	"time"

	"schoolify_be/configs"
)

type contextKey string

const ReqIDKey contextKey = "reqId"

// ReqIDMiddleware intercepts HTTP requests to verify or generate a unique X-Req-ID.
// It sets the ID in both the request/response headers and the request context.
func ReqIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqId := r.Header.Get("X-Req-ID")
		if reqId == "" {
			var err error
			reqId, err = GenerateUUIDv7()
			if err != nil {
				// Fallback if UUIDv7 generation fails (highly unlikely)
				reqId = "req_fallback_" + time.Now().Format("20060102150405.000")
			}
		}

		// Ensure it is present in request and response headers
		r.Header.Set("X-Req-ID", reqId)
		w.Header().Set("X-Req-ID", reqId)

		// Inject into context
		ctx := context.WithValue(r.Context(), ReqIDKey, reqId)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetReqID extracts the X-Req-ID string from context.Context.
func GetReqID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqId, ok := ctx.Value(ReqIDKey).(string); ok {
		return reqId
	}
	return ""
}

const (
	UserIDKey   contextKey = "userId"
	UserRoleKey contextKey = "userRole"
)

// AuthMiddleware extracts and validates the JWE token from the Authorization header,
// checks the session in Redis, and automatically refreshes the token if it is close to expiring.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":"Authorization token is required"}`))
			return
		}

		tokenStr := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenStr = authHeader[7:]
		}

		cfg := configs.GetConfig()
		claims, err := Decrypt(tokenStr, []byte(cfg.Service.EncryptKey))
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":"Invalid or expired token"}`))
			return
		}

		// 1. Verify token Expiry
		expiryVal, ok := claims["Expiry"]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":"Invalid token claims"}`))
			return
		}

		var expiryUnix int64
		switch v := expiryVal.(type) {
		case float64:
			expiryUnix = int64(v)
		case int64:
			expiryUnix = v
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":"Invalid token claims"}`))
			return
		}

		if time.Now().Unix() > expiryUnix {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":"Invalid or expired token"}`))
			return
		}

		userId, _ := claims["ID"].(string)
		username, _ := claims["Username"].(string)
		userRole, _ := claims["Role"].(string)

		if userId == "" || username == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":"Invalid token claims"}`))
			return
		}

		// 2. Check session in Redis
		redisKey1 := "TOKEN_" + username + "_1"
		redisKey2 := "TOKEN_" + username + "_2"

		val1, err1 := RedisInstance().GetValue(r.Context(), redisKey1)
		val2, err2 := RedisInstance().GetValue(r.Context(), redisKey2)

		if (err1 != nil || val1 != tokenStr) && (err2 != nil || val2 != tokenStr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":"Invalid or expired token"}`))
			return
		}

		// 3. Automatic session refresh mechanism
		remainingTime := expiryUnix - time.Now().Unix()
		refreshThreshold := int64(cfg.Service.RefreshTime) * 60

		// Only trigger refresh if the request token is the primary one in key1
		// to avoid redundant/overlapping refreshes from key2 requests.
		if tokenStr == val1 && remainingTime < refreshThreshold {
			// Get TTL and rotate key1 to key2
			ttlVal, getTtlErr := RedisInstance().GetTTL(r.Context(), redisKey1)
			if getTtlErr == nil && ttlVal > 0 {
				_ = RedisInstance().SaveValue(r.Context(), redisKey2, val1, ttlVal)
			} else if remainingTime > 0 {
				_ = RedisInstance().SaveValue(r.Context(), redisKey2, val1, time.Duration(remainingTime)*time.Second)
			}

			// Generate new JWE token with the same claims but updated Expiry and IssuedAt
			sessionMinutes := cfg.Service.SessionTime
			if sessionMinutes <= 0 {
				sessionMinutes = 60
			}
			newExpiry := time.Now().Add(time.Duration(sessionMinutes) * time.Minute)
			claims["Expiry"] = newExpiry.Unix()
			claims["IssuedAt"] = time.Now().Unix()

			newToken, encryptErr := Encrypt(claims, []byte(cfg.Service.EncryptKey))
			if encryptErr == nil {
				// Save new JWE to key1 with full TTL (SESSION_TIME)
				newTTL := time.Duration(sessionMinutes) * time.Minute
				saveErr := RedisInstance().SaveValue(r.Context(), redisKey1, newToken, newTTL)
				if saveErr == nil {
					// Set response header "xrfs-token" to newToken
					w.Header().Set("xrfs-token", newToken)
				} else {
					reqID := GetReqID(r.Context())
					Log(reqID, ERROR, r.URL.Path, "AuthMiddleware", nil, "failed to save refreshed session in Redis", saveErr)
					w.Header().Set("xrfs-token", tokenStr)
				}
			} else {
				reqID := GetReqID(r.Context())
				Log(reqID, ERROR, r.URL.Path, "AuthMiddleware", claims, "failed to encrypt refreshed JWE", encryptErr)
				w.Header().Set("xrfs-token", tokenStr)
			}
		} else {
			// If not refreshing or matching key2, just use the current request token
			w.Header().Set("xrfs-token", tokenStr)
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userId)
		ctx = context.WithValue(ctx, UserRoleKey, userRole)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts the authenticated user ID string from context.Context.
func GetUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if userId, ok := ctx.Value(UserIDKey).(string); ok {
		return userId
	}
	return ""
}

// GetUserRole extracts the authenticated user role string from context.Context.
func GetUserRole(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if role, ok := ctx.Value(UserRoleKey).(string); ok {
		return role
	}
	return ""
}

// BuyerMiddleware restricts access to users with the "buyer" role.
// It must be chained after AuthMiddleware.
func BuyerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := GetUserRole(r.Context())
		if role != "0190a0a0-0000-7000-8000-000000000004" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errors":"Forbidden: Only buyers are allowed"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
