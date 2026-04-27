package admin

import (
	"errors"

	"github.com/google/uuid"
	"github.com/impa-hub/internal/auth"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/models"
	"gorm.io/gorm"
)

type CreateUserRequest struct {
	Name     string          `json:"name" binding:"required"`
	Email    string          `json:"email" binding:"required,email"`
	Password string          `json:"password" binding:"required,min=6"`
	Role     models.UserRole `json:"role" binding:"required"`
}

type UpdateUserRequest struct {
	Name   *string          `json:"name,omitempty"`
	Email  *string          `json:"email,omitempty"`
	Active *bool            `json:"is_active,omitempty"`
	Role   *models.UserRole `json:"role,omitempty"`
}

type UpdateQuotasRequest struct {
	MaxInstances     *int  `json:"max_instances,omitempty"`
	MaxChatwootConns *int  `json:"max_chatwoot_conns,omitempty"`
	MaxTypebotConns  *int  `json:"max_typebot_conns,omitempty"`
	MaxEvoServers    *int  `json:"max_evo_servers,omitempty"`
	CanUseChatwoot   *bool `json:"can_use_chatwoot,omitempty"`
	CanUseTypebot    *bool `json:"can_use_typebot,omitempty"`
}

type UserListResponse struct {
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
	InstanceCount    int64           `json:"instance_count"`
	ServerCount      int64           `json:"server_count"`
	ChatwootCount    int64           `json:"chatwoot_count"`
}

func CreateUser(req CreateUserRequest) (*models.User, error) {
	// Verifica se email já existe
	var count int64
	database.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		return nil, errors.New("email já está em uso")
	}

	// Não permite criar outro superadmin
	if req.Role == models.RoleSuperAdmin {
		return nil, errors.New("não é possível criar outro superadmin")
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Name:             req.Name,
		Email:            req.Email,
		Password:         hashed,
		Role:             req.Role,
		Active:           true,
		MaxInstances:     5,
		MaxChatwootConns: 5,
		MaxTypebotConns:  5,
		MaxEvoServers:    3,
		CanUseChatwoot:   true,
		CanUseTypebot:    true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func ListUsers() ([]UserListResponse, error) {
	var users []models.User
	if err := database.DB.Where("role != ?", models.RoleSuperAdmin).Find(&users).Error; err != nil {
		return nil, err
	}

	var result []UserListResponse
	for _, u := range users {
		var instanceCount, serverCount, chatwootCount int64
		database.DB.Model(&models.Instance{}).Where("user_id = ?", u.ID).Count(&instanceCount)
		database.DB.Model(&models.EvoServer{}).Where("user_id = ?", u.ID).Count(&serverCount)
		database.DB.Model(&models.ChatwootConfig{}).Where("user_id = ?", u.ID).Count(&chatwootCount)

		result = append(result, UserListResponse{
			ID:               u.ID,
			Name:             u.Name,
			Email:            u.Email,
			Role:             u.Role,
			Active:           u.Active,
			MaxInstances:     u.MaxInstances,
			MaxChatwootConns: u.MaxChatwootConns,
			MaxTypebotConns:  u.MaxTypebotConns,
			MaxEvoServers:    u.MaxEvoServers,
			CanUseChatwoot:   u.CanUseChatwoot,
			CanUseTypebot:    u.CanUseTypebot,
			InstanceCount:    instanceCount,
			ServerCount:      serverCount,
			ChatwootCount:    chatwootCount,
		})
	}

	return result, nil
}

func GetUser(userID uuid.UUID) (*UserListResponse, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("usuário não encontrado")
		}
		return nil, err
	}

	var instanceCount, serverCount, chatwootCount int64
	database.DB.Model(&models.Instance{}).Where("user_id = ?", user.ID).Count(&instanceCount)
	database.DB.Model(&models.EvoServer{}).Where("user_id = ?", user.ID).Count(&serverCount)
	database.DB.Model(&models.ChatwootConfig{}).Where("user_id = ?", user.ID).Count(&chatwootCount)

	return &UserListResponse{
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
		InstanceCount:    instanceCount,
		ServerCount:      serverCount,
		ChatwootCount:    chatwootCount,
	}, nil
}

func UpdateUser(userID uuid.UUID, req UpdateUserRequest) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return errors.New("usuário não encontrado")
	}

	if user.Role == models.RoleSuperAdmin {
		return errors.New("não é possível editar o superadmin")
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if req.Role != nil && *req.Role != models.RoleSuperAdmin {
		updates["role"] = *req.Role
	}

	return database.DB.Model(&user).Updates(updates).Error
}

func UpdateQuotas(userID uuid.UUID, req UpdateQuotasRequest) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return errors.New("usuário não encontrado")
	}

	updates := make(map[string]interface{})
	if req.MaxInstances != nil {
		updates["max_instances"] = *req.MaxInstances
	}
	if req.MaxChatwootConns != nil {
		updates["max_chatwoot_conns"] = *req.MaxChatwootConns
	}
	if req.MaxTypebotConns != nil {
		updates["max_typebot_conns"] = *req.MaxTypebotConns
	}
	if req.MaxEvoServers != nil {
		updates["max_evo_servers"] = *req.MaxEvoServers
	}
	if req.CanUseChatwoot != nil {
		updates["can_use_chatwoot"] = *req.CanUseChatwoot
	}
	if req.CanUseTypebot != nil {
		updates["can_use_typebot"] = *req.CanUseTypebot
	}

	return database.DB.Model(&user).Updates(updates).Error
}

func DeleteUser(userID uuid.UUID) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return errors.New("usuário não encontrado")
	}

	if user.Role == models.RoleSuperAdmin {
		return errors.New("não é possível excluir o superadmin")
	}

	// Soft delete - mantém registros
	return database.DB.Delete(&user).Error
}

func ResetUserPassword(userID uuid.UUID, newPassword string) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return errors.New("usuário não encontrado")
	}

	hashed, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return database.DB.Model(&user).Update("password", hashed).Error
}
