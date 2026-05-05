package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
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
			log.Printf("Recovered from panic: %v", r)
		}
	}()

	// Load configuration
	if err := config.Load(".env"); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	cfg := config.Get()
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

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

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
	smtpConfig := email.GetSMTPConfig()
	var emailService email.EmailService
	if smtpConfig.Host != "" && smtpConfig.Username != "" {
		fmt.Println("Using SMTP email service")
		emailService = email.NewSMTPEmailService(smtpConfig)
	} else {
		fmt.Println("Using development email service (console output)")
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
	r.Use(middleware.Recovery(zap.L()))
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Health check（包含依赖检查）
	r.GET("/health", func(c *gin.Context) {
		status := gin.H{"status": "ok"}

		// 检查数据库
		if sqlDB != nil {
			if err := sqlDB.Ping(); err != nil {
				status["database"] = "unhealthy"
				status["status"] = "degraded"
			} else {
				status["database"] = "ok"
			}
		}

		// 检查 Redis
		if redis.Client != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := redis.Client.Ping(ctx).Err(); err != nil {
				status["redis"] = "unhealthy"
				status["status"] = "degraded"
			} else {
				status["redis"] = "ok"
			}
		} else {
			status["redis"] = "disabled"
		}

		httpStatus := 200
		if status["status"] != "ok" {
			httpStatus = 503
		}
		c.JSON(httpStatus, status)
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
		}
		usersAdmin := v1.Group("/users")
		usersAdmin.Use(middleware.Auth(), middleware.RequireRole(db, "admin"))
		{
			usersAdmin.POST("", userHandler.CreateUser)
			usersAdmin.PUT("/:id", userHandler.UpdateUser)
			usersAdmin.DELETE("/:id", userHandler.DeleteUser)
		}

		// Role routes (protected, admin only)
		roles := v1.Group("/roles")
		roles.Use(middleware.Auth(), middleware.RequireRole(db, "admin"))
		{
			roles.GET("", roleHandler.ListRoles)
			roles.GET("/:id", roleHandler.GetRole)
			roles.POST("", roleHandler.CreateRole)
			roles.PUT("/:id", roleHandler.UpdateRole)
			roles.DELETE("/:id", roleHandler.DeleteRole)
		}

		// Permission routes (protected, admin only)
		permissions := v1.Group("/permissions")
		permissions.Use(middleware.Auth(), middleware.RequireRole(db, "admin"))
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

	// Graceful shutdown
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 在 goroutine 中启动服务器
	go func() {
		zap.L().Info("Server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 等待中断信号
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	zap.L().Info("Shutting down server...")

	// 给正在处理的请求 10 秒时间完成
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		zap.L().Fatal("Server forced to shutdown", zap.Error(err))
	}

	zap.L().Info("Server exited gracefully")
}
