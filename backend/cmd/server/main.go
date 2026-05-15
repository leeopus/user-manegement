package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
	"github.com/user-system/backend/internal/email"
	"github.com/user-system/backend/internal/handler"
	"github.com/user-system/backend/internal/middleware"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/audit"
	"github.com/user-system/backend/pkg/logger"
	"github.com/user-system/backend/pkg/redis"
	"go.uber.org/zap"
	"github.com/user-system/backend/pkg/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Bootstrap logger first with a safe default, then reconfigure after loading config
	_ = logger.Initialize("info")
	defer logger.Logger.Sync()

	defer func() {
		if r := recover(); r != nil {
			zap.L().Error("Recovered from panic", zap.Any("panic", r))
		}
	}()

	if err := config.Load(".env"); err != nil {
		zap.L().Fatal("Failed to load config", zap.Error(err))
	}

	cfg := config.Get()

	logLevel := "info"
	if cfg.Server.GinMode == "debug" {
		logLevel = "debug"
	}
	if err := logger.Initialize(logLevel); err != nil {
		zap.L().Fatal("Failed to initialize logger", zap.Error(err))
	}

	if err := redis.InitRedis(cfg.Redis.URL); err != nil {
		if cfg.Server.GinMode == "release" {
			zap.L().Fatal("Redis connection required in release mode — rate limiting, token blacklist, CSRF, and account lockout all depend on it",
				zap.Error(err))
		}
		zap.L().Warn("Failed to connect to Redis, security features will be degraded", zap.Error(err))
	}
	defer redis.Close()

	db, err := gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{})
	if err != nil {
		zap.L().Fatal("Failed to connect to database", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		zap.L().Fatal("Failed to get database connection", zap.Error(err))
	}

	sqlDB.SetMaxIdleConns(cfg.GetIntEnv("DB_MAX_IDLE_CONNS", 10))
	sqlDB.SetMaxOpenConns(cfg.GetIntEnv("DB_MAX_OPEN_CONNS", 100))
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.GetIntEnv("DB_CONN_MAX_LIFETIME_MIN", 60)) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.GetIntEnv("DB_CONN_MAX_IDLE_MIN", 10)) * time.Minute)

	zap.L().Info("Database connected successfully")

	if cfg.GetBoolEnv("AUTO_MIGRATE", false) {
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
			&repository.PasswordHistory{},
				&repository.EmailVerificationToken{},
		); err != nil {
			zap.L().Fatal("Failed to migrate database", zap.Error(err))
		}
		zap.L().Info("Database migration completed")

			// Composite index for audit log queries by user + created_at
			if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_user_created_at ON audit_logs (user_id, created_at DESC)").Error; err != nil {
				zap.L().Warn("Failed to create audit_logs composite index", zap.Error(err))
			}

			// Backfill empty nicknames for existing users
			backfillNicknames(db)
	} else {
		zap.L().Info("AutoMigrate skipped (set AUTO_MIGRATE=true to enable)")
	}

	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)

	// Seed 默认权限、角色和初始管理员（幂等，可安全重复执行）
	service.SeedDefaults(db, userRepo, roleRepo, permissionRepo)

	oauthAppRepo := repository.NewOAuthApplicationRepository(db)
	oauthTokenRepo := repository.NewOAuthTokenRepository(db)
	asyncAuditRepo := audit.NewAsyncAuditLogRepository(repository.NewAuditLogRepository(db))
	auditLogRepo := asyncAuditRepo // implements repository.AuditLogRepository
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(db)
	passwordHistoryRepo := repository.NewPasswordHistoryRepository(db)
	emailVerifyTokenRepo := repository.NewEmailVerificationTokenRepository(db)

	smtpConfig := email.GetSMTPConfig()
	var emailService email.EmailService
	if smtpConfig.Host != "" && smtpConfig.Username != "" {
		zap.L().Info("Using SMTP email service")
		emailService = email.NewSMTPEmailService(smtpConfig)
	} else {
		zap.L().Info("Using development email service (console output)")
		emailService = email.NewDevelopmentEmailService()
	}

	// 统一创建共享依赖，避免重复实例化
	auditLogger := service.NewAuditLogger(auditLogRepo)
	rbacCache := auth.NewRBACCacheManager(redis.Client)
	blacklistMgr := auth.NewTokenBlacklistManager(redis.Client)
	refreshTokenMgr := auth.NewRefreshTokenManager(redis.Client)

	authService := service.NewAuthService(userRepo, passwordHistoryRepo, auditLogger, redis.Client, blacklistMgr, refreshTokenMgr)
	eventPublisher := service.NewUserEventPublisher(redis.Client)
	emailVerifyService := service.NewEmailVerificationService(emailVerifyTokenRepo, userRepo, emailService, redis.Client, cfg.Frontend.URL, eventPublisher)
	profileService := service.NewProfileService(userRepo, auditLogger, refreshTokenMgr, blacklistMgr, eventPublisher)
	uploadService := service.NewUploadService(cfg.Upload)
	userService := service.NewUserService(userRepo, roleRepo, auditLogger, rbacCache, blacklistMgr, refreshTokenMgr)
	roleService := service.NewRoleService(roleRepo, permissionRepo, auditLogger, rbacCache)
	permissionService := service.NewPermissionService(permissionRepo, auditLogger, rbacCache)
	oauthService := service.NewOAuthService(oauthAppRepo, oauthTokenRepo, userRepo, auditLogger, redis.Client, blacklistMgr)
	// 密码重置速率限制配置
	pwResetCooldown := time.Duration(cfg.GetIntEnv("PW_RESET_EMAIL_COOLDOWN_SEC", 60)) * time.Second
	pwResetMaxPerHour := cfg.GetIntEnv("PW_RESET_IP_MAX_PER_HOUR", 3)
	pwResetBlockHours := time.Duration(cfg.GetIntEnv("PW_RESET_IP_BLOCK_HOURS", 24)) * time.Hour

	passwordService := service.NewPasswordResetService(userRepo, passwordResetTokenRepo, passwordHistoryRepo, auditLogger, emailService, cfg.Frontend.URL, refreshTokenMgr, blacklistMgr, redis.Client, pwResetCooldown, eventPublisher)

	rbacCfg := middleware.RBACConfig{
		UserRepo:     userRepo,
		RedisClient:  redis.Client,
		BlacklistMgr: blacklistMgr,
	}

	authCfg := middleware.AuthConfig{
		BlacklistMgr: blacklistMgr,
		UserRepo:     userRepo,
	}

	authHandler := handler.NewAuthHandler(authService)
	emailVerifyHandler := handler.NewEmailVerificationHandler(emailVerifyService)
	profileHandler := handler.NewProfileHandler(profileService, uploadService)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	permissionHandler := handler.NewPermissionHandler(permissionService)
	oauthHandler := handler.NewOAuthHandler(oauthService)
	auditHandler := handler.NewAuditHandler(auditLogRepo)
	csrfHandler := handler.NewCSRFHandler(redis.Client)
	passwordHandler := handler.NewPasswordHandler(passwordService)

	if cfg.Server.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	if trustedProxies := os.Getenv("TRUSTED_PROXIES"); trustedProxies != "" {
		proxies := strings.Split(trustedProxies, ",")
		for i := range proxies {
			proxies[i] = strings.TrimSpace(proxies[i])
		}
		if err := r.SetTrustedProxies(proxies); err != nil {
			zap.L().Warn("Failed to set trusted proxies", zap.Error(err))
		}
		zap.L().Info("Trusted proxies configured", zap.Strings("proxies", proxies))
	} else {
		if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
			zap.L().Warn("Failed to set trusted proxies", zap.Error(err))
		}
	}

	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(zap.L()))
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.BodyLimit())
	r.Use(middleware.Timeout(30 * time.Second))

	// 健康检查：公开端点仅返回 200/503，不泄露基础设施信息
	r.GET("/health", healthCheckPublic(sqlDB))

	// Static file serving for uploaded avatars
	r.Static("/uploads/avatars", "./uploads/avatars")

	r.GET("/api/csrf-token", middleware.CSRFTokenRateLimit(redis.Client), csrfHandler.GetToken)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.APIRateLimit(redis.Client))
	{
		v1.GET("/health/detail", middleware.Auth(authCfg), healthCheckDetail(sqlDB))

		auth := v1.Group("/auth")
		auth.Use(middleware.CSRF(redis.Client))
		{
			auth.POST("/register", middleware.RegisterRateLimit(redis.Client), authHandler.Register)
			auth.POST("/login", middleware.LoginRateLimit(redis.Client), authHandler.Login)
			auth.POST("/logout", middleware.Auth(authCfg), authHandler.Logout)
			auth.POST("/refresh", middleware.CSRF(redis.Client), middleware.RefreshRateLimit(redis.Client), authHandler.RefreshToken)
			auth.GET("/me", middleware.Auth(authCfg), authHandler.GetCurrentUser)
			auth.PUT("/password/change", middleware.Auth(authCfg), middleware.PasswordChangeRateLimit(redis.Client), authHandler.ChangePassword)
		}

		v1.POST("/auth/password/reset-request", middleware.CSRF(redis.Client), middleware.PasswordResetRateLimit(redis.Client, pwResetMaxPerHour, pwResetBlockHours), passwordHandler.RequestReset)
		v1.POST("/auth/password/reset", middleware.CSRF(redis.Client), passwordHandler.ResetPassword)
		v1.POST("/auth/password/validate-token", middleware.CSRF(redis.Client), middleware.PasswordResetRateLimit(redis.Client, pwResetMaxPerHour, pwResetBlockHours), passwordHandler.ValidateToken)

			// Email verification (public endpoint)
			v1.POST("/auth/email/verify", middleware.CSRF(redis.Client), middleware.EmailVerifyRateLimit(redis.Client), emailVerifyHandler.VerifyEmail)

		users := v1.Group("/users")
		users.Use(middleware.Auth(authCfg), middleware.CSRF(redis.Client), middleware.RequirePermission(rbacCfg, service.PermUserRead))
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.POST("", middleware.RequirePermission(rbacCfg, service.PermUserWrite), userHandler.CreateUser)
			users.PUT("/:id", middleware.RequirePermission(rbacCfg, service.PermUserWrite), userHandler.UpdateUser)
				users.PUT("/:id/status", middleware.RequirePermission(rbacCfg, service.PermUserWrite), userHandler.UpdateUserStatus)
			users.DELETE("/:id", middleware.RequirePermission(rbacCfg, service.PermUserDelete), userHandler.DeleteUser)
			users.DELETE("/:id/hard", middleware.RequirePermission(rbacCfg, service.PermUserDelete), userHandler.HardDeleteUser)
			users.POST("/:id/roles", middleware.RequirePermission(rbacCfg, service.PermUserWrite), userHandler.AssignRole)
			users.DELETE("/:id/roles/:roleId", middleware.RequirePermission(rbacCfg, service.PermUserWrite), userHandler.RemoveRole)
		}

		roles := v1.Group("/roles")
		roles.Use(middleware.Auth(authCfg), middleware.CSRF(redis.Client), middleware.RequirePermission(rbacCfg, service.PermRoleManage))
		{
			roles.GET("", roleHandler.ListRoles)
			roles.GET("/:id", roleHandler.GetRole)
			roles.POST("", roleHandler.CreateRole)
			roles.PUT("/:id", roleHandler.UpdateRole)
			roles.DELETE("/:id", roleHandler.DeleteRole)
			roles.POST("/:id/permissions", roleHandler.AssignPermission)
			roles.DELETE("/:id/permissions/:permissionId", roleHandler.RemovePermission)
		}

		permissions := v1.Group("/permissions")
		permissions.Use(middleware.Auth(authCfg), middleware.CSRF(redis.Client), middleware.RequirePermission(rbacCfg, service.PermPermissionManage))
		{
			permissions.GET("", permissionHandler.ListPermissions)
			permissions.GET("/:id", permissionHandler.GetPermission)
			permissions.POST("", permissionHandler.CreatePermission)
			permissions.PUT("/:id", permissionHandler.UpdatePermission)
			permissions.DELETE("/:id", permissionHandler.DeletePermission)
		}

		oauth := v1.Group("/oauth")
		{
				oauth.GET("/authorize", middleware.OptionalAuth(authCfg), oauthHandler.AuthorizePage)
			oauth.POST("/authorize", middleware.Auth(authCfg), middleware.CSRF(redis.Client), oauthHandler.Authorize)
			oauth.POST("/token", middleware.OAuthTokenRateLimit(redis.Client), oauthHandler.Token)
			oauth.GET("/userinfo", middleware.OAuthAuth(authCfg), oauthHandler.Userinfo)
				oauth.GET("/frontchannel-logout-uris", oauthHandler.FrontchannelLogoutURIs)
		}

		oauthApps := v1.Group("/oauth/applications")
		oauthApps.Use(middleware.Auth(authCfg), middleware.CSRF(redis.Client), middleware.RequirePermission(rbacCfg, service.PermOAuthManage))
		{
			oauthApps.GET("", oauthHandler.ListApplications)
			oauthApps.GET("/:id", oauthHandler.GetApplication)
			oauthApps.POST("", oauthHandler.CreateApplication)
			oauthApps.PUT("/:id", oauthHandler.UpdateApplication)
			oauthApps.DELETE("/:id", oauthHandler.DeleteApplication)
		}

		auditLogs := v1.Group("/audit-logs")
		auditLogs.Use(middleware.Auth(authCfg), middleware.RequirePermission(rbacCfg, service.PermAuditRead))
		{
			auditLogs.GET("", auditHandler.ListAuditLogs)
		}

		profileGroup := v1.Group("/profile")
		profileGroup.Use(middleware.Auth(authCfg), middleware.CSRF(redis.Client))
		{
			profileGroup.GET("", profileHandler.GetProfile)
			profileGroup.PUT("", middleware.ProfileUpdateRateLimit(redis.Client), profileHandler.UpdateProfile)
			profileGroup.POST("/avatar", middleware.ProfileUpdateRateLimit(redis.Client), profileHandler.UploadAvatar)
			profileGroup.POST("/email/resend", middleware.EmailVerifyRateLimit(redis.Client), emailVerifyHandler.ResendVerification)
			profileGroup.DELETE("/account", middleware.ProfileUpdateRateLimit(redis.Client), profileHandler.DeleteAccount)
		}
	}

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		zap.L().Info("Server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("Failed to start server", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 审计日志定时清理：每天执行，删除超过保留期的记录
	retentionDays := cfg.Security.AuditRetentionDays
	if retentionDays <= 0 {
		retentionDays = 90
	}
	go func() {
		// 启动时立即执行一次清理
		if deleted, err := auditLogRepo.CleanupOlderThan(retentionDays); err != nil {
			zap.L().Error("Audit log initial cleanup failed", zap.Error(err))
		} else if deleted > 0 {
			zap.L().Info("Audit log initial cleanup completed", zap.Int64("deleted", deleted), zap.Int("retention_days", retentionDays))
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if deleted, err := auditLogRepo.CleanupOlderThan(retentionDays); err != nil {
					zap.L().Error("Audit log cleanup failed", zap.Error(err))
				} else if deleted > 0 {
					zap.L().Info("Audit log cleanup completed", zap.Int64("deleted", deleted), zap.Int("retention_days", retentionDays))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	zap.L().Info("Audit log cleanup scheduled", zap.Int("retention_days", retentionDays))

	<-ctx.Done()

	zap.L().Info("Shutting down server...")

	// 优雅关闭审计日志队列
	asyncAuditRepo.Shutdown(5 * time.Second)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		zap.L().Fatal("Server forced to shutdown", zap.Error(err))
	}

	zap.L().Info("Server exited gracefully")
}

// healthCheckPublic 公开健康检查，仅返回 200/503，不泄露基础设施细节
func healthCheckPublic(sqlDB *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		healthy := true

		if sqlDB != nil {
			if err := sqlDB.Ping(); err != nil {
				healthy = false
			}
		}

		if redis.IsAvailable() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := redis.Client.Ping(ctx).Err(); err != nil {
				healthy = false
			}
		}

		if healthy {
			c.JSON(200, gin.H{"status": "ok"})
		} else {
			c.JSON(503, gin.H{"status": "unhealthy"})
		}
	}
}

// healthCheckDetail 详细健康检查，需认证，返回各组件状态
func healthCheckDetail(sqlDB *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{"status": "ok"}

		if sqlDB != nil {
			if err := sqlDB.Ping(); err != nil {
				status["database"] = "unhealthy"
				status["status"] = "degraded"
			} else {
				status["database"] = "ok"
			}
		}

		if redis.IsAvailable() {
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
	}
}

// backfillNicknames generates unique nicknames for users who have empty ones.
// Safe to run multiple times (idempotent).
func backfillNicknames(db *gorm.DB) {
	type user struct {
		ID uint
	}
	var users []user
	if err := db.Model(&repository.User{}).
		Where("(nickname = '' OR nickname IS NULL) AND deleted_at IS NULL").
		FindInBatches(&users, 100, func(tx *gorm.DB, batch int) error {
			for _, u := range users {
				nickname, err := utils.GenerateUniqueNickname()
				if err != nil {
					zap.L().Error("backfill: generate nickname failed", zap.Uint("user_id", u.ID), zap.Error(err))
					continue
				}
				// Check uniqueness within batch and DB
				var count int64
				tx.Model(&repository.User{}).Where("nickname = ? AND deleted_at IS NULL", nickname).Count(&count)
				if count > 0 {
					// Collision, try again with timestamp suffix
					nickname = nickname + "_" + fmt.Sprintf("%d", time.Now().UnixNano()%10000)
				}
				if err := tx.Model(&repository.User{}).Where("id = ?", u.ID).Update("nickname", nickname).Error; err != nil {
					zap.L().Error("backfill: update nickname failed", zap.Uint("user_id", u.ID), zap.Error(err))
				}
			}
			return nil
		}).Error; err != nil {
		zap.L().Error("backfill nicknames failed", zap.Error(err))
		return
	}
	if len(users) > 0 {
		zap.L().Info("Backfilled nicknames", zap.Int("count", len(users)))
	}
}
