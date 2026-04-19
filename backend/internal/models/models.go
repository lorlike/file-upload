package models

import "time"

type User struct {
	ID           string    `gorm:"primaryKey;size:36"`
	Username     string    `gorm:"uniqueIndex;not null;size:64"`
	PasswordHash string    `gorm:"not null;size:255"`
	IsAdmin      bool      `gorm:"not null;default:false"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

type FileRecord struct {
	ID           string    `gorm:"primaryKey;size:36"`
	UserID       string    `gorm:"index;not null;size:36"`
	OriginalName string    `gorm:"not null;size:255"`
	StoredName   string    `gorm:"not null;size:255"`
	MimeType     string    `gorm:"size:255"`
	SizeBytes    int64     `gorm:"not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

