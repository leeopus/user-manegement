package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
	"github.com/user-system/backend/internal/email"
	"github.com/user-system/backend/internal/handler"
	"github.com/user-system/backend/internal/middleware"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/redis"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 全局错误恢复机制
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Recovered from panic: %v", r)
			// 这里可以添加：记录到监控系统、发送告警、清理资源等
		}
	}()

	// Load configuration
	if err := config.Load(".env"); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	cfg := config.Get()
	// Note: Logger initialization is optional in dev mode
	_ = cfg

	// Connect to Redis
	if err := redis.InitRedis(cfg.Redis.URL); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		log.Println("Rate limiting will be disabled")
	}
	defer redis.Close()

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 配置数据库连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生存时间

	log.Println("Database connected successfully")

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
		&repository.PasswordResetToken{},
	); err != nil {
		zap.L().Fatal("Failed to migrate database")
	}

	zap.L().Info("Database migration completed")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	oauthAppRepo := repository.NewOAuthApplicationRepository(db)
	oauthTokenRepo := repository.NewOAuthTokenRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(db)

	// Initialize services
	// 根据环境变量选择邮件服务
	smtpConfig := email.GetSMTPConfig()
	var emailService email.EmailService
	if smtpConfig.Host != "" && smtpConfig.Username != "" {
		fmt.Println("📧 使用SMTP邮件服务")
		emailService = email.NewSMTPEmailService(smtpConfig)
	} else {
		fmt.Println("📧 使用开发环境邮件服务 (控制台输出)")
		emailService = email.NewDevelopmentEmailService()
	}

	authService := service.NewAuthService(userRepo, auditLogRepo)
	userService := service.NewUserService(userRepo, roleRepo, auditLogRepo)
	roleService := service.NewRoleService(roleRepo, permissionRepo, auditLogRepo)
	permissionService := service.NewPermissionService(permissionRepo, auditLogRepo)
	oauthService := service.NewOAuthService(oauthAppRepo, oauthTokenRepo, userRepo, auditLogRepo)
	passwordService := service.NewPasswordResetService(userRepo, passwordResetTokenRepo, auditLogRepo, emailService, cfg.Frontend.URL)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	permissionHandler := handler.NewPermissionHandler(permissionService)
	oauthHandler := handler.NewOAuthHandler(oauthService)
	csrfHandler := handler.NewCSRFHandler()
	passwordHandler := handler.NewPasswordHandler(passwordService)

	// Setup Gin
	if cfg.Server.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Recovery(zap.L())) // 使用我们的自定义恢复中间件
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// CSRF token endpoint (no auth required)
	r.GET("/api/csrf-token", csrfHandler.GetToken)

	// API routes
	v1 := r.Group("/api/v1")
	{
		// Auth routes with CSRF protection
		auth := v1.Group("/auth")
		auth.Use(middleware.CSRF())
		{
			auth.POST("/register", middleware.RegisterRateLimit(redis.Client), authHandler.Register)
			auth.POST("/login", middleware.LoginRateLimit(redis.Client), authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.GET("/me", middleware.Auth(), authHandler.GetCurrentUser)
		}

		// Password reset routes (no CSRF protection - users don't have valid sessions)
		v1.POST("/auth/password/reset-request", passwordHandler.RequestReset)
		v1.POST("/auth/password/reset", passwordHandler.ResetPassword)
		v1.POST("/auth/password/validate-token", passwordHandler.ValidateToken)

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
	zap.L().Info("Server starting", zap.String("port", cfg.Server.Port))
	if err := r.Run(addr); err != nil {
		zap.L().Fatal("Failed to start server", zap.Error(err))
	}
}
