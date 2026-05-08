package service

import (
	"os"
	"time"

	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type defaultPermission struct {
	Name        string
	Code        string
	Resource    string
	Action      string
	Description string
}

var defaultPermissions = []defaultPermission{
	{Name: "查看用户", Code: PermUserRead, Resource: "users", Action: "read", Description: "查看用户列表和详情"},
	{Name: "编辑用户", Code: PermUserWrite, Resource: "users", Action: "write", Description: "创建、编辑用户及分配角色"},
	{Name: "删除用户", Code: PermUserDelete, Resource: "users", Action: "delete", Description: "软删除和硬删除用户"},
	{Name: "管理角色", Code: PermRoleManage, Resource: "roles", Action: "manage", Description: "增删改角色及分配权限"},
	{Name: "管理权限", Code: PermPermissionManage, Resource: "permissions", Action: "manage", Description: "增删改权限"},
	{Name: "管理OAuth应用", Code: PermOAuthManage, Resource: "oauth", Action: "manage", Description: "管理 OAuth 应用"},
	{Name: "查看审计日志", Code: PermAuditRead, Resource: "audit", Action: "read", Description: "查看系统审计日志"},
}

// SeedDefaults 创建默认权限、角色，以及可选的初始管理员用户。
// 该函数幂等：已存在的记录不会被重复创建。
func SeedDefaults(
	db *gorm.DB,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	permissionRepo repository.PermissionRepository,
) {
	seedPermissions(db, permissionRepo)
	seedRoles(db, roleRepo, permissionRepo)
	seedAdminUser(db, userRepo, roleRepo)
}

func seedPermissions(db *gorm.DB, permissionRepo repository.PermissionRepository) {
	for _, p := range defaultPermissions {
		if _, err := permissionRepo.FindByCode(p.Code); err == nil {
			continue
		}
		perm := &repository.Permission{
			Name:        p.Name,
			Code:        p.Code,
			Resource:    p.Resource,
			Action:      p.Action,
			Description: p.Description,
		}
		if err := db.Create(perm).Error; err != nil {
			zap.L().Warn("Seed: failed to create permission", zap.String("code", p.Code), zap.Error(err))
		} else {
			zap.L().Info("Seed: created permission", zap.String("code", p.Code))
		}
	}
}

func seedRoles(db *gorm.DB, roleRepo repository.RoleRepository, permissionRepo repository.PermissionRepository) {
	// 创建 admin 角色
	adminRole := seedRole(db, RoleAdmin, "管理员", "系统管理员，拥有所有权限")
	if adminRole != nil {
		// 将所有权限分配给 admin 角色
		for _, p := range defaultPermissions {
			perm, err := permissionRepo.FindByCode(p.Code)
			if err != nil {
				continue
			}
			roleRepo.AssignPermission(adminRole.ID, perm.ID)
		}
	}

	// 创建 user 角色
	seedRole(db, RoleUser, "普通用户", "注册用户的默认角色")
}

func seedRole(db *gorm.DB, code, name, description string) *repository.Role {
	if err := db.Where("code = ?", code).First(&repository.Role{}).Error; err == nil {
		return nil
	}
	role := &repository.Role{
		Name:        name,
		Code:        code,
		Description: description,
	}
	if err := db.Create(role).Error; err != nil {
		zap.L().Warn("Seed: failed to create role", zap.String("code", code), zap.Error(err))
		return nil
	}
	zap.L().Info("Seed: created role", zap.String("code", code))
	return role
}

func seedAdminUser(db *gorm.DB, userRepo repository.UserRepository, roleRepo repository.RoleRepository) {
	adminEmail := os.Getenv("INITIAL_ADMIN_EMAIL")
	adminPassword := os.Getenv("INITIAL_ADMIN_PASSWORD")
	adminUsername := os.Getenv("INITIAL_ADMIN_USERNAME")
	if adminUsername == "" {
		adminUsername = "system_admin"
	}

	if adminEmail == "" || adminPassword == "" {
		zap.L().Info("Seed: INITIAL_ADMIN_EMAIL/PASSWORD not set, skipping admin user creation")
		return
	}

	// 校验管理员密码强度，防止弱密码
	if _, err := utils.ValidatePassword(adminPassword, adminUsername); err != nil {
		zap.L().Error("Seed: INITIAL_ADMIN_PASSWORD does not meet strength requirements",
			zap.Error(err),
			zap.String("hint", "use at least 8 chars with letters and numbers"),
		)
		return
	}

	// 检查是否已有任何用户（只在没有用户时创建初始管理员）
	users, _, err := userRepo.List(0, 1, repository.UserFilters{})
	if err != nil {
		zap.L().Warn("Seed: failed to check existing users", zap.Error(err))
		return
	}
	if len(users) > 0 {
		zap.L().Info("Seed: users already exist, skipping admin user creation")
		return
	}

	// 查找 admin 角色
	adminRole, roleErr := roleRepo.FindByCode(RoleAdmin)
	if roleErr != nil {
		zap.L().Warn("Seed: admin role not found, skipping admin user creation", zap.Error(roleErr))
		return
	}

	passwordHash, err := utils.HashPassword(adminPassword)
	if err != nil {
		zap.L().Error("Seed: failed to hash admin password", zap.Error(err))
		return
	}

	now := time.Now()
	adminUser := &repository.User{
		Username:          adminUsername,
		Email:             adminEmail,
		PasswordHash:      passwordHash,
		Status:            StatusActive,
		PasswordChangedAt: &now,
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(adminUser).Error; err != nil {
			return err
		}
		return tx.Create(&repository.UserRole{UserID: adminUser.ID, RoleID: adminRole.ID}).Error
	}); err != nil {
		zap.L().Error("Seed: failed to create admin user with role in transaction", zap.String("email", adminEmail), zap.Error(err))
		return
	}

	zap.L().Info("Seed: initial admin user created",
		zap.String("email", adminEmail),
		zap.Uint("user_id", adminUser.ID),
	)
}
