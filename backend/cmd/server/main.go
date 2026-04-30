package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
	"github.com/user-system/backend/internal/handler"
	"github.com/user-system/backend/internal/middleware"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	if err := config.Load(".env"); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	cfg := config.Get()
	if err := logger.Initialize("info"); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{})
	if err != nil {
		logger.Fatal("Failed to connect to database", logger.Logger.WithError(err))
	}

	logger.Info("Database connected successfully")

	// Auto migrate
	if err := db.AutoMigrate(
		&repository.User{},
		&repository.Role{},
		&repository.Permission{},
		&repository.UserRole{},
		&repository.RolePermission{},
		&repository.OAuthApplication{},
		&repository.OAuthToken{},
		&repository.AuditLog{},
	); err != nil {
		logger.Fatal("Failed to migrate database")
	}

	logger.Info("Database migration completed")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	oauthAppRepo := repository.NewOAuthApplicationRepository(db)
	oauthTokenRepo := repository.NewOAuthTokenRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, auditLogRepo)
	userService := service.NewUserService(userRepo, roleRepo, auditLogRepo)
	roleService := service.NewRoleService(roleRepo, permissionRepo, auditLogRepo)
	permissionService := service.NewPermissionService(permissionRepo, auditLogRepo)
	oauthService := service.NewOAuthService(oauthAppRepo, oauthTokenRepo, userRepo, auditLogRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	permissionHandler := handler.NewPermissionHandler(permissionService)
	oauthHandler := handler.NewOAuthHandler(oauthService)

	// Setup Gin
	if cfg.Server.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	v1 := r.Group("/api/v1")
	{
		// Auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.GET("/me", middleware.Auth(), authHandler.GetCurrentUser)
		}

		// User routes (protected)
		users := v1.Group("/users")
		users.Use(middleware.Auth())
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.POST("", userHandler.CreateUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}

		// Role routes (protected)
		roles := v1.Group("/roles")
		roles.Use(middleware.Auth())
		{
			roles.GET("", roleHandler.ListRoles)
			roles.GET("/:id", roleHandler.GetRole)
			roles.POST("", roleHandler.CreateRole)
			roles.PUT("/:id", roleHandler.UpdateRole)
			roles.DELETE("/:id", roleHandler.DeleteRole)
		}

		// Permission routes (protected)
		permissions := v1.Group("/permissions")
		permissions.Use(middleware.Auth())
		{
			permissions.GET("", permissionHandler.ListPermissions)
			permissions.GET("/:id", permissionHandler.GetPermission)
			permissions.POST("", permissionHandler.CreatePermission)
			permissions.PUT("/:id", permissionHandler.UpdatePermission)
			permissions.DELETE("/:id", permissionHandler.DeletePermission)
		}

		// OAuth routes
		oauth := v1.Group("/oauth")
		{
			oauth.POST("/authorize", oauthHandler.Authorize)
			oauth.POST("/token", oauthHandler.Token)
			oauth.GET("/userinfo", middleware.OAuthAuth(), oauthHandler.Userinfo)
		}

		// OAuth Application management (protected)
		oauthApps := v1.Group("/oauth/applications")
		oauthApps.Use(middleware.Auth())
		{
			oauthApps.GET("", oauthHandler.ListApplications)
			oauthApps.GET("/:id", oauthHandler.GetApplication)
			oauthApps.POST("", oauthHandler.CreateApplication)
			oauthApps.PUT("/:id", oauthHandler.UpdateApplication)
			oauthApps.DELETE("/:id", oauthHandler.DeleteApplication)
		}
	}

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	logger.Info("Server starting", logger.Logger.With("port", cfg.Server.Port))
	if err := r.Run(addr); err != nil {
		logger.Fatal("Failed to start server", logger.Logger.WithError(err))
	}
}
