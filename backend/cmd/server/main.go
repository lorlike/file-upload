package main

import (
	"log"
	"net/http"
	"time"

	"file-upload/backend/internal/bootstrap"
	"file-upload/backend/internal/config"
	"file-upload/backend/internal/db"
	httpserver "file-upload/backend/internal/http"
	"file-upload/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	var (
		database *gorm.DB
		err      error
	)
	for attempt := 0; attempt < 30; attempt++ {
		database, err = db.Open(cfg.DatabaseDSN)
		if err == nil {
			break
		}
		log.Printf("database not ready, retrying: %v", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	if err := bootstrap.SeedAdminUser(database, cfg); err != nil {
		log.Fatalf("seed admin user: %v", err)
	}

	store := storage.NewLocal(cfg.UploadDir)
	if err := store.Ensure(); err != nil {
		log.Fatalf("prepare upload dir: %v", err)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(httpserver.CORS(cfg.CORSAllowedOrigins))

	handlers := &httpserver.Handlers{
		DB:     database,
		Config: cfg,
		Store:  store,
	}
	handlers.RegisterRoutes(router)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("server listening on %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("run server: %v", err)
	}
}
