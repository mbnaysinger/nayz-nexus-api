package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Roles da aplicação NAYZTECH no nayz-auth
const (
	RoleAdmin = "ADMIN"
)

// CustomClaims espelha o JWT HS256 emitido pelo nayz-auth
type CustomClaims struct {
	AppID       string   `json:"app_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	PersonID    string   `json:"person_id"`
	Name        string   `json:"name"`
	jwt.RegisteredClaims
}

// Tipo customizado para a chave do contexto para evitar colisões
type contextKey string

const ClaimsContextKey contextKey = "user_claims"

// ClaimsFromRequest recupera os claims injetados pelo RequireAuth
func ClaimsFromRequest(r *http.Request) (*CustomClaims, bool) {
	claims, ok := r.Context().Value(ClaimsContextKey).(*CustomClaims)
	return claims, ok
}

type AuthMiddleware struct {
	jwtSecret     []byte
	expectedAppID string
}

// NewAuthMiddleware valida assinatura (secret idêntica à do nayz-auth) e,
// quando expectedAppID é informado, recusa tokens emitidos para outras aplicações.
func NewAuthMiddleware(secret string, expectedAppID string) *AuthMiddleware {
	return &AuthMiddleware{jwtSecret: []byte(secret), expectedAppID: expectedAppID}
}

// RequireAuth intercepta a requisição, valida o JWT e injeta os claims no contexto
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error": "Token ausente ou em formato inválido"}`, http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims := &CustomClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return m.jwtSecret, nil
		}, jwt.WithValidMethods([]string{"HS256"}))

		if err != nil || !token.Valid {
			http.Error(w, `{"error": "Token inválido, expirado ou assinatura corrompida"}`, http.StatusUnauthorized)
			return
		}

		// Token de outra aplicação do nayz-auth não vale aqui (mesmo com a mesma secret)
		if m.expectedAppID != "" && claims.AppID != m.expectedAppID {
			http.Error(w, `{"error": "Token não pertence à aplicação NAYZTECH"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// RequireRole exige, além do login, uma role específica presente no claim "roles"
func (m *AuthMiddleware) RequireRole(requiredRole string, next http.HandlerFunc) http.HandlerFunc {
	return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromRequest(r)
		if !ok {
			http.Error(w, `{"error": "Falha crítica ao ler o contexto da requisição"}`, http.StatusInternalServerError)
			return
		}

		for _, role := range claims.Roles {
			if role == requiredRole {
				next(w, r)
				return
			}
		}
		http.Error(w, `{"error": "Acesso Negado: Você não possui a permissão necessária"}`, http.StatusForbidden)
	})
}
