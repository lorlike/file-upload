package httpserver

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"file-upload/backend/internal/auth"
	"file-upload/backend/internal/config"
	"file-upload/backend/internal/models"
	"file-upload/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handlers struct {
	DB     *gorm.DB
	Config config.Config
	Store  *storage.Local
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      userDTO   `json:"user"`
}

type userDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"isAdmin"`
}

type fileDTO struct {
	ID           string    `json:"id"`
	OriginalName string    `json:"originalName"`
	SizeBytes    int64     `json:"sizeBytes"`
	UploadedAt   time.Time `json:"uploadedAt"`
	DownloadURL  string    `json:"downloadUrl"`
}

type renameFileRequest struct {
	OriginalName string `json:"originalName"`
}

type adminUserDTO struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	IsAdmin   bool      `json:"isAdmin"`
	CreatedAt time.Time `json:"createdAt"`
	FileCount int64     `json:"fileCount"`
	TotalSize int64     `json:"totalSize"`
}

type adminFileDTO struct {
	ID           string    `json:"id"`
	OriginalName string    `json:"originalName"`
	SizeBytes    int64     `json:"sizeBytes"`
	UploadedAt   time.Time `json:"uploadedAt"`
	DownloadURL  string    `json:"downloadUrl"`
	UserID       string    `json:"userId"`
}

func (h *Handlers) RegisterRoutes(router *gin.Engine) {
	router.GET("/healthz", h.healthz)

	api := router.Group("/api")
	api.POST("/auth/register", h.register)
	api.POST("/auth/login", h.login)

	protected := api.Group("")
	protected.Use(auth.Middleware(h.Config.JWTSecret, h.DB))
	protected.GET("/me", h.me)
	protected.GET("/files", h.listFiles)
	protected.POST("/files", h.uploadFile)
	protected.PATCH("/files/:id", h.renameFile)
	protected.DELETE("/files/:id", h.deleteFile)
	protected.GET("/files/:id/download", h.downloadFile)

	admin := api.Group("/admin")
	admin.Use(auth.Middleware(h.Config.JWTSecret, h.DB))
	admin.Use(auth.RequireAdmin())
	admin.GET("/users", h.adminListUsers)
	admin.GET("/users/:id/files", h.adminListUserFiles)
	admin.DELETE("/users/:id", h.adminDeleteUser)
	admin.DELETE("/files/:id", h.adminDeleteFile)
}

func (h *Handlers) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) register(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "username required and password must be at least 6 characters"})
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to hash password"})
		return
	}

	user := models.User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: passwordHash,
	}

	if err := h.DB.Create(&user).Error; err != nil {
		if isDuplicateError(err) {
			c.JSON(http.StatusConflict, gin.H{"message": "username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create user"})
		return
	}

	h.issueToken(c, user)
}

func (h *Handlers) login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "username and password are required"})
		return
	}

	var user models.User
	if err := h.DB.Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid credentials"})
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid credentials"})
		return
	}

	h.issueToken(c, user)
}

func (h *Handlers) issueToken(c *gin.Context, user models.User) {
	token, expiresAt, err := auth.SignToken(h.Config.JWTSecret, user.ID, user.Username, h.Config.TokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create token"})
		return
	}

	c.JSON(http.StatusOK, authResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      userDTO{ID: user.ID, Username: user.Username, IsAdmin: user.IsAdmin},
	})
}

func (h *Handlers) me(c *gin.Context) {
	userID := c.GetString(auth.ContextUserIDKey)
	username := c.GetString(auth.ContextUsernameKey)
	isAdmin := c.GetBool(auth.ContextIsAdminKey)
	c.JSON(http.StatusOK, userDTO{ID: userID, Username: username, IsAdmin: isAdmin})
}

func (h *Handlers) listFiles(c *gin.Context) {
	userID := c.GetString(auth.ContextUserIDKey)

	var records []models.FileRecord
	if err := h.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load files"})
		return
	}

	items := make([]fileDTO, 0, len(records))
	for _, record := range records {
		items = append(items, fileDTO{
			ID:           record.ID,
			OriginalName: record.OriginalName,
			SizeBytes:    record.SizeBytes,
			UploadedAt:   record.CreatedAt,
			DownloadURL:  fmt.Sprintf("/files/%s/download", record.ID),
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handlers) uploadFile(c *gin.Context) {
	userID := c.GetString(auth.ContextUserIDKey)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "file is required"})
		return
	}

	if fileHeader.Size > h.Config.MaxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"message": "file too large"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to open file"})
		return
	}
	defer src.Close()

	storedName := uuid.NewString() + filepath.Ext(fileHeader.Filename)
	if _, _, err := h.Store.Save(storedName, src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save file"})
		return
	}

	record := models.FileRecord{
		ID:           uuid.NewString(),
		UserID:       userID,
		OriginalName: sanitizeDisplayName(fileHeader.Filename),
		StoredName:   storedName,
		MimeType:     fileHeader.Header.Get("Content-Type"),
		SizeBytes:    fileHeader.Size,
	}

	if err := h.DB.Create(&record).Error; err != nil {
		_ = h.Store.Remove(storedName)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to record file"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           record.ID,
		"originalName": record.OriginalName,
		"sizeBytes":    record.SizeBytes,
		"uploadedAt":   record.CreatedAt,
		"downloadUrl":  fmt.Sprintf("/files/%s/download", record.ID),
	})
}

func (h *Handlers) downloadFile(c *gin.Context) {
	userID := c.GetString(auth.ContextUserIDKey)
	fileID := strings.TrimSpace(c.Param("id"))

	var record models.FileRecord
	if err := h.DB.Where("id = ? AND user_id = ?", fileID, userID).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "file not found"})
		return
	}

	fullPath := h.Store.Path(record.StoredName)
	if _, err := os.Stat(fullPath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "file not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, record.OriginalName))
	c.File(fullPath)
}

func (h *Handlers) renameFile(c *gin.Context) {
	userID := c.GetString(auth.ContextUserIDKey)
	fileID := strings.TrimSpace(c.Param("id"))

	var req renameFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	if strings.TrimSpace(req.OriginalName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "originalName is required"})
		return
	}

	newName := sanitizeDisplayName(req.OriginalName)

	var record models.FileRecord
	if err := h.DB.Where("id = ? AND user_id = ?", fileID, userID).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "file not found"})
		return
	}

	record.OriginalName = newName
	if err := h.DB.Save(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to rename file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           record.ID,
		"originalName": record.OriginalName,
		"sizeBytes":    record.SizeBytes,
		"uploadedAt":   record.CreatedAt,
		"downloadUrl":  fmt.Sprintf("/files/%s/download", record.ID),
	})
}

func (h *Handlers) deleteFile(c *gin.Context) {
	userID := c.GetString(auth.ContextUserIDKey)
	fileID := strings.TrimSpace(c.Param("id"))

	var record models.FileRecord
	if err := h.DB.Where("id = ? AND user_id = ?", fileID, userID).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "file not found"})
		return
	}

	if err := h.Store.Remove(record.StoredName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to remove file"})
		return
	}

	if err := h.DB.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete file record"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handlers) adminListUsers(c *gin.Context) {
	type userRow struct {
		ID        string
		Username  string
		IsAdmin   bool
		CreatedAt time.Time
	}
	type statRow struct {
		UserID    string
		FileCount int64
		TotalSize int64
	}

	var users []userRow
	if err := h.DB.Model(&models.User{}).
		Select("id, username, is_admin, created_at").
		Order("created_at desc").
		Scan(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load users"})
		return
	}

	var stats []statRow
	if err := h.DB.Model(&models.FileRecord{}).
		Select("user_id, count(*) as file_count, coalesce(sum(size_bytes), 0) as total_size").
		Group("user_id").
		Scan(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load user stats"})
		return
	}

	statMap := make(map[string]statRow, len(stats))
	for _, item := range stats {
		statMap[item.UserID] = item
	}

	items := make([]adminUserDTO, 0, len(users))
	for _, user := range users {
		stat := statMap[user.ID]
		items = append(items, adminUserDTO{
			ID:        user.ID,
			Username:  user.Username,
			IsAdmin:   user.IsAdmin,
			CreatedAt: user.CreatedAt,
			FileCount: stat.FileCount,
			TotalSize: stat.TotalSize,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handlers) adminListUserFiles(c *gin.Context) {
	userID := strings.TrimSpace(c.Param("id"))

	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		return
	}

	var records []models.FileRecord
	if err := h.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load files"})
		return
	}

	items := make([]adminFileDTO, 0, len(records))
	for _, record := range records {
		items = append(items, adminFileDTO{
			ID:           record.ID,
			OriginalName: record.OriginalName,
			SizeBytes:    record.SizeBytes,
			UploadedAt:   record.CreatedAt,
			DownloadURL:  fmt.Sprintf("/files/%s/download", record.ID),
			UserID:       record.UserID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  userDTO{ID: user.ID, Username: user.Username, IsAdmin: user.IsAdmin},
		"items": items,
	})
}

func (h *Handlers) adminDeleteUser(c *gin.Context) {
	userID := strings.TrimSpace(c.Param("id"))

	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		return
	}

	var files []models.FileRecord
	if err := h.DB.Where("user_id = ?", userID).Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load user files"})
		return
	}

	for _, file := range files {
		_ = h.Store.Remove(file.StoredName)
	}

	if err := h.DB.Where("user_id = ?", userID).Delete(&models.FileRecord{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete user files"})
		return
	}

	if err := h.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete user"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handlers) adminDeleteFile(c *gin.Context) {
	fileID := strings.TrimSpace(c.Param("id"))

	var record models.FileRecord
	if err := h.DB.First(&record, "id = ?", fileID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "file not found"})
		return
	}

	if err := h.Store.Remove(record.StoredName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to remove file"})
		return
	}

	if err := h.DB.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete file record"})
		return
	}

	c.Status(http.StatusNoContent)
}

func sanitizeDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "untitled"
	}
	return filepath.Base(name)
}

func isDuplicateError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint"))
}

