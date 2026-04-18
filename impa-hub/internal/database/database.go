package database

import (
	"log"

	"github.com/impa-hub/internal/config"
	"github.com/impa-hub/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) *gorm.DB {
	logLevel := logger.Warn
	if cfg.LogLevel == "debug" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("[IMPA HUB] Erro ao conectar ao banco de dados: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("[IMPA HUB] Erro ao obter DB pool: %v", err)
	}

	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)

	DB = db
	return db
}

func Migrate() {
	// Fix: tabela chatwoot_conversations tinha coluna remote_j_i_d (GORM naming bug)
	// Drop e recria — é apenas cache de conversas
	if DB.Migrator().HasTable("chatwoot_conversations") {
		DB.Exec(`DROP TABLE chatwoot_conversations`)
		log.Println("[IMPA HUB] Tabela chatwoot_conversations recriada (fix coluna remote_jid)")
	}

	err := DB.AutoMigrate(
		&models.User{},
		&models.EvoServer{},
		&models.Instance{},
		&models.ChatwootConfig{},
		&models.WebhookLog{},
		&models.ChatwootConversation{},
		&models.ChatwootMessageMap{},
		&models.TypebotConfig{},
		&models.TypebotSession{},
		&models.TypebotSetting{},
	)
	if err != nil {
		log.Fatalf("[IMPA HUB] Erro na migração do banco: %v", err)
	}

	log.Println("[IMPA HUB] Migração do banco concluída")
}
