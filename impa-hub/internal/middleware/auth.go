package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/impa-hub/internal/config"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/models"
)

type Claims struct {
	UserID uuid.UUID       `json:"userId"`
	Email  string          `json:"email"`
	Role   models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token não fornecido"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Formato de token inválido"})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(config.AppConfig.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido ou expirado"})
			c.Abort()
			return
		}

		// Verifica se o usuário ainda existe e está ativo
		var user models.User
		if err := database.DB.Where("id = ? AND active = true", claims.UserID).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não encontrado ou inativo"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", claims.Role)
		c.Set("user", &user)
		c.Next()
	}
}

func SuperAdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("userRole")
		if !exists || role.(models.UserRole) != models.RoleSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Acesso restrito a SuperAdmin"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func AdminOrAbove() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Acesso negado"})
			c.Abort()
			return
		}
		r := role.(models.UserRole)
		if r != models.RoleSuperAdmin && r != models.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Acesso restrito a Admin ou superior"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) *models.User {
	user, exists := c.Get("user")
	if !exists {
		return nil
	}
	return user.(*models.User)
}

func GetCurrentUserID(c *gin.Context) uuid.UUID {
	id, exists := c.Get("userID")
	if !exists {
		return uuid.Nil
	}
	return id.(uuid.UUID)
}
