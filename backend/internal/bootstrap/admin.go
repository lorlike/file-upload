package bootstrap

import (
	"strings"

	"file-upload/backend/internal/auth"
	"file-upload/backend/internal/config"
	"file-upload/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedAdminUser(database *gorm.DB, cfg config.Config) error {
	username := strings.TrimSpace(cfg.AdminUsername)
	password := strings.TrimSpace(cfg.AdminPassword)
	if username == "" || password == "" {
		return nil
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	user := models.User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: passwordHash,
		IsAdmin:      true,
	}

	var existing models.User
	err = database.Where("username = ?", username).First(&existing).Error
	switch {
	case err == nil:
		return database.Model(&existing).Updates(map[string]any{
			"password_hash": passwordHash,
			"is_admin":      true,
		}).Error
	case err == gorm.ErrRecordNotFound:
		return database.Create(&user).Error
	default:
		return err
	}
}

