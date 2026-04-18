package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/impa-hub/internal/config"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/middleware"
	"github.com/impa-hub/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID               uuid.UUID       `json:"id"`
	Name             string          `json:"name"`
	Email            string          `json:"email"`
	Role             models.UserRole `json:"role"`
	Active           bool            `json:"is_active"`
	MaxInstances     int             `json:"max_instances"`
	MaxChatwootConns int             `json:"max_chatwoot_conns"`
	MaxTypebotConns  int             `json:"max_typebot_conns"`
	MaxEvoServers    int             `json:"max_evo_servers"`
	CanUseChatwoot   bool            `json:"can_use_chatwoot"`
	CanUseTypebot    bool            `json:"can_use_typebot"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"current_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func Login(req LoginRequest) (*LoginResponse, error) {
	var user models.User
	if err := database.DB.Where("email = ? AND active = true", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("credenciais inválidas")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("credenciais inválidas")
	}

	token, err := generateToken(&user)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: token,
		User:  toUserResponse(&user),
	}, nil
}

func toUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:               user.ID,
		Name:             user.Name,
		Email:            user.Email,
		Role:             user.Role,
		Active:           user.Active,
		MaxInstances:     user.MaxInstances,
		MaxChatwootConns: user.MaxChatwootConns,
		MaxTypebotConns:  user.MaxTypebotConns,
		MaxEvoServers:    user.MaxEvoServers,
		CanUseChatwoot:   user.CanUseChatwoot,
		CanUseTypebot:    user.CanUseTypebot,
	}
}

func ChangePassword(userID uuid.UUID, req ChangePasswordRequest) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return errors.New("usuário não encontrado")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return errors.New("senha atual incorreta")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return database.DB.Model(&user).Update("password", string(hashed)).Error
}

func generateToken(user *models.User) (string, error) {
	claims := &middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.AppConfig.JWTExpirationHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}

func CreateSuperAdminIfNotExists(cfg *config.Config) error {
	var count int64
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleSuperAdmin).Count(&count)
	if count > 0 {
		return nil
	}

	hashed, err := HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}

	admin := models.User{
		Name:             "Super Admin",
		Email:            cfg.AdminEmail,
		Password:         hashed,
		Role:             models.RoleSuperAdmin,
		Active:           true,
		MaxInstances:     999,
		MaxChatwootConns: 999,
		MaxTypebotConns:  999,
		MaxEvoServers:    999,
		CanUseChatwoot:   true,
		CanUseTypebot:    true,
	}

	return database.DB.Create(&admin).Error
}
