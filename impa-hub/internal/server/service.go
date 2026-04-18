package server

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/impa-hub/internal/database"
	"github.com/impa-hub/internal/evoclient"
	"github.com/impa-hub/internal/models"
	"gorm.io/gorm"
)

type CreateServerRequest struct {
	Name   string `json:"name" binding:"required"`
	URL    string `json:"base_url" binding:"required"`
	APIKey string `json:"global_api_key" binding:"required"`
}

type UpdateServerRequest struct {
	Name   *string `json:"name,omitempty"`
	URL    *string `json:"base_url,omitempty"`
	APIKey *string `json:"global_api_key,omitempty"`
	Active *bool   `json:"is_active,omitempty"`
}

type ServerResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	URL           string    `json:"base_url"`
	Active        bool      `json:"is_active"`
	InstanceCount int64     `json:"instance_count"`
}

func CreateServer(userID uuid.UUID, req CreateServerRequest) (*models.EvoServer, error) {
	// Valida URL
	if _, err := url.ParseRequestURI(req.URL); err != nil {
		return nil, errors.New("URL inválida")
	}

	// Verifica quota
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, errors.New("usuário não encontrado")
	}

	var count int64
	database.DB.Model(&models.EvoServer{}).Where("user_id = ?", userID).Count(&count)
	if int(count) >= user.MaxEvoServers {
		return nil, fmt.Errorf("limite de servidores atingido (%d/%d)", count, user.MaxEvoServers)
	}

	// Testa conexão com o servidor
	client := evoclient.New(req.URL, req.APIKey)
	_, err := client.ListInstances()
	if err != nil {
		return nil, fmt.Errorf("não foi possível conectar ao servidor: %v", err)
	}

	srv := models.EvoServer{
		UserID: userID,
		Name:   req.Name,
		URL:    req.URL,
		APIKey: req.APIKey,
		Active: true,
	}

	if err := database.DB.Create(&srv).Error; err != nil {
		return nil, err
	}

	return &srv, nil
}

func ListServers(userID uuid.UUID) ([]ServerResponse, error) {
	var servers []models.EvoServer
	if err := database.DB.Where("user_id = ?", userID).Find(&servers).Error; err != nil {
		return nil, err
	}

	var result []ServerResponse
	for _, s := range servers {
		var count int64
		database.DB.Model(&models.Instance{}).Where("evo_server_id = ?", s.ID).Count(&count)

		result = append(result, ServerResponse{
			ID:            s.ID,
			Name:          s.Name,
			URL:           s.URL,
			Active:        s.Active,
			InstanceCount: count,
		})
	}
	return result, nil
}

func GetServer(userID, serverID uuid.UUID) (*ServerResponse, error) {
	var srv models.EvoServer
	if err := database.DB.Where("id = ? AND user_id = ?", serverID, userID).First(&srv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("servidor não encontrado")
		}
		return nil, err
	}

	var count int64
	database.DB.Model(&models.Instance{}).Where("evo_server_id = ?", srv.ID).Count(&count)

	return &ServerResponse{
		ID:            srv.ID,
		Name:          srv.Name,
		URL:           srv.URL,
		Active:        srv.Active,
		InstanceCount: count,
	}, nil
}

func UpdateServer(userID, serverID uuid.UUID, req UpdateServerRequest) error {
	var srv models.EvoServer
	if err := database.DB.Where("id = ? AND user_id = ?", serverID, userID).First(&srv).Error; err != nil {
		return errors.New("servidor não encontrado")
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.URL != nil {
		if _, err := url.ParseRequestURI(*req.URL); err != nil {
			return errors.New("URL inválida")
		}
		updates["url"] = *req.URL
	}
	if req.APIKey != nil {
		updates["api_key"] = *req.APIKey
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}

	return database.DB.Model(&srv).Updates(updates).Error
}

func DeleteServer(userID, serverID uuid.UUID) error {
	var srv models.EvoServer
	if err := database.DB.Where("id = ? AND user_id = ?", serverID, userID).First(&srv).Error; err != nil {
		return errors.New("servidor não encontrado")
	}

	// Verifica se há instâncias
	var count int64
	database.DB.Model(&models.Instance{}).Where("evo_server_id = ?", serverID).Count(&count)
	if count > 0 {
		return fmt.Errorf("não é possível remover servidor com %d instâncias ativas", count)
	}

	return database.DB.Delete(&srv).Error
}

func TestServerConnection(userID, serverID uuid.UUID) error {
	var srv models.EvoServer
	if err := database.DB.Where("id = ? AND user_id = ?", serverID, userID).First(&srv).Error; err != nil {
		return errors.New("servidor não encontrado")
	}

	client := evoclient.New(srv.URL, srv.APIKey)
	_, err := client.ListInstances()
	return err
}

// GetEvoClient retorna um client configurado para o servidor
func GetEvoClient(serverID uuid.UUID) (*evoclient.Client, error) {
	var srv models.EvoServer
	if err := database.DB.First(&srv, serverID).Error; err != nil {
		return nil, errors.New("servidor não encontrado")
	}
	return evoclient.New(srv.URL, srv.APIKey), nil
}
