package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	apperrors "github.com/user-system/backend/pkg/errors"
	"github.com/user-system/backend/pkg/utils"
	"go.uber.org/zap"
)

const dnsResolveTimeout = 3 * time.Second

var dnsResolver = &net.Resolver{}

type OAuthService interface {
	CreateApplication(name, redirectURIs, scopes string, auditCtx dto.AuditContext) (*repository.OAuthApplication, string, error)
	GetApplication(id uint) (*repository.OAuthApplication, error)
	UpdateApplication(id uint, name, redirectURIs string, auditCtx dto.AuditContext) (*repository.OAuthApplication, error)
	DeleteApplication(id uint, auditCtx dto.AuditContext) error
	ListApplications(offset, pageSize int) ([]repository.OAuthApplication, int64, error)
	Authorize(userID uint, clientID, redirectURI, state, scope, codeChallenge, codeChallengeMethod, ipAddress, userAgent string) (string, error)
	Token(clientID, clientSecret, code, redirectURI, codeVerifier string) (string, string, error)
	UserinfoByID(userID uint) (*repository.User, error)
}

type oauthService struct {
	appRepo      repository.OAuthApplicationRepository
	tokenRepo    repository.OAuthTokenRepository
	userRepo     repository.UserRepository
	auditLogger  *AuditLogger
	blacklistMgr *auth.TokenBlacklistManager
	redis        *redis.Client
}

func NewOAuthService(appRepo repository.OAuthApplicationRepository, tokenRepo repository.OAuthTokenRepository, userRepo repository.UserRepository, auditLogger *AuditLogger, redisClient *redis.Client, blacklistMgr *auth.TokenBlacklistManager) OAuthService {
	return &oauthService{
		appRepo:      appRepo,
		tokenRepo:    tokenRepo,
		userRepo:     userRepo,
		auditLogger:  auditLogger,
		blacklistMgr: blacklistMgr,
		redis:        redisClient,
	}
}

type authorizationCodeData struct {
	ClientID           string `json:"client_id"`
	UserID             uint   `json:"user_id"`
	RedirectURI        string `json:"redirect_uri"`
	State              string `json:"state"`
	Scope              string `json:"scope"`
	CodeChallenge      string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
}

func (s *oauthService) CreateApplication(name, redirectURIs, scopes string, auditCtx dto.AuditContext) (*repository.OAuthApplication, string, error) {
	clientID, err := utils.GenerateRandomString(16)
	if err != nil {
		return nil, "", apperrors.ErrInternalServer
	}
	clientID = "client_" + clientID

	// 服务端自动生成 client_secret，仅在创建时返回明文
	rawSecret, err := utils.GenerateRandomString(24)
	if err != nil {
		return nil, "", apperrors.ErrInternalServer
	}

	hashedSecret, err := utils.HashSecret(rawSecret)
	if err != nil {
		zap.L().Error("Failed to hash client secret", zap.Error(err))
		return nil, "", apperrors.ErrInternalServer
	}

	for _, uri := range strings.Split(redirectURIs, ",") {
		trimmed := strings.TrimSpace(uri)
		if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
			return nil, "", apperrors.ErrOAuthInvalidRedirectURI.WithDetails(map[string]interface{}{
				"reason": "redirect URI must use http or https protocol",
			})
		}
		if err := validateRedirectURIIsPublic(trimmed); err != nil {
			return nil, "", apperrors.ErrOAuthInvalidRedirectURI.WithDetails(map[string]interface{}{
				"reason": err.Error(),
			})
		}
	}

	if scopes == "" {
		scopes = "read"
	}

	app := &repository.OAuthApplication{
		Name:         name,
		ClientID:     clientID,
		ClientSecret: hashedSecret,
		RedirectURIs: redirectURIs,
		Scopes:       scopes,
	}
	if err := s.appRepo.Create(app); err != nil {
		return nil, "", apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&auditCtx, "create_oauth_application", "oauth", map[string]interface{}{
		"application_name": name,
		"client_id":        clientID,
	})

	return app, rawSecret, nil
}

func (s *oauthService) Authorize(userID uint, clientID, redirectURI, state, scope, codeChallenge, codeChallengeMethod, ipAddress, userAgent string) (string, error) {
	if state == "" {
		return "", apperrors.ErrOAuthInvalidState
	}

	// PKCE 校验：如果提供了 code_challenge，必须使用 S256（禁用不安全的 plain 方法）
	if codeChallenge != "" {
		if codeChallengeMethod != "S256" {
			return "", apperrors.ErrOAuthInvalidScope.WithDetails(map[string]interface{}{
				"reason": "code_challenge_method must be S256",
			})
		}
	}

	app, err := s.appRepo.FindByClientID(clientID)
	if err != nil {
		return "", apperrors.ErrOAuthInvalidClient
	}

	if !isValidRedirectURI(app.RedirectURIs, redirectURI) {
		return "", apperrors.ErrOAuthInvalidRedirectURI
	}

	// 重定向时再次校验目标地址的 DNS 解析，防止 DNS 重绑定攻击
	if err := validateRedirectURIIsPublic(redirectURI); err != nil {
		return "", apperrors.ErrOAuthInvalidRedirectURI.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	// 验证请求的 scope 是否在应用注册的 scopes 范围内
	if scope == "" {
		scope = "read" // 默认 scope
	}
	if !isValidScope(app.Scopes, scope) {
		return "", apperrors.ErrOAuthInvalidScope
	}

	code, err := utils.GenerateRandomString(32)
	if err != nil {
		return "", apperrors.ErrInternalServer
	}

	codeData := authorizationCodeData{
		ClientID:            clientID,
		UserID:              userID,
		RedirectURI:         redirectURI,
		State:               state,
		Scope:               scope,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	}
	codeDataJSON, _ := json.Marshal(codeData)

	if s.redis == nil {
		return "", apperrors.ErrInternalServer
	}

	ctx := context.Background()
	key := fmt.Sprintf("oauth:auth_code:%s", code)
	if err := s.redis.Set(ctx, key, string(codeDataJSON), 2*time.Minute).Err(); err != nil {
		zap.L().Error("Failed to store authorization code", zap.Error(err))
		return "", apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&dto.AuditContext{
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}, "oauth_authorize", "oauth", map[string]interface{}{
		"client_id": clientID,
	})

	return code, nil
}

func (s *oauthService) Token(clientID, clientSecret, code, redirectURI, codeVerifier string) (string, string, error) {
	app, err := s.appRepo.FindByClientID(clientID)
	if err != nil {
		return "", "", apperrors.ErrOAuthInvalidClient
	}

	if !utils.VerifySecret(clientSecret, app.ClientSecret) {
		return "", "", apperrors.ErrOAuthInvalidClientSecret
	}

	if s.redis == nil {
		return "", "", apperrors.ErrInternalServer
	}

	ctx := context.Background()
	key := fmt.Sprintf("oauth:auth_code:%s", code)

	// 原子性 GET + DEL，确保 authorization code 只能使用一次
	delScript := redis.NewScript(`
		local val = redis.call("GET", KEYS[1])
		if val == false then
			return nil
		end
		redis.call("DEL", KEYS[1])
		return val
	`)
	result, err := delScript.Run(ctx, s.redis, []string{key}).Result()
	if err != nil {
		if err == redis.Nil {
			return "", "", apperrors.ErrOAuthInvalidCode
		}
		return "", "", apperrors.ErrOAuthInvalidCode
	}
	codeDataStr, ok := result.(string)
	if !ok {
		return "", "", apperrors.ErrOAuthInvalidCode
	}

	var codeData authorizationCodeData
	if err := json.Unmarshal([]byte(codeDataStr), &codeData); err != nil {
		return "", "", apperrors.ErrOAuthInvalidCode
	}

	if codeData.ClientID != clientID {
		return "", "", apperrors.ErrOAuthInvalidCode
	}

	if codeData.RedirectURI != redirectURI {
		return "", "", apperrors.ErrOAuthInvalidRedirectURI
	}

	// 兑换时再次校验 redirect URI 的 DNS 解析，防止创建应用与实际兑换之间 DNS 被重绑定
	if err := validateRedirectURIIsPublic(redirectURI); err != nil {
		return "", "", apperrors.ErrOAuthInvalidRedirectURI.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	// PKCE 验证：如果 authorize 时设置了 code_challenge，token 兑换时必须提供 code_verifier
	if codeData.CodeChallenge != "" {
		if codeVerifier == "" {
			return "", "", apperrors.ErrOAuthInvalidCode.WithDetails(map[string]interface{}{
				"reason": "code_verifier is required",
			})
		}
		if !verifyPKCECodeVerifier(codeVerifier, codeData.CodeChallenge, codeData.CodeChallengeMethod) {
			return "", "", apperrors.ErrOAuthInvalidCode.WithDetails(map[string]interface{}{
				"reason": "code_verifier mismatch",
			})
		}
	}

	user, err := s.userRepo.FindByID(codeData.UserID)
	if err != nil {
		return "", "", apperrors.ErrUserNotFound
	}

	accessToken, _, err := utils.GenerateOAuthToken(user.ID, user.Username, user.Email, codeData.Scope, codeData.ClientID)
	if err != nil {
		return "", "", apperrors.ErrInternalServer
	}

	oauthToken := &repository.OAuthToken{
		ApplicationID: app.ID,
		UserID:        user.ID,
		AccessToken:   repository.HashOAuthToken(accessToken),
		ExpiresAt:     time.Now().Add(time.Hour),
	}
	if err := s.tokenRepo.Create(oauthToken); err != nil {
		zap.L().Error("Failed to store OAuth token", zap.Error(err))
		return "", "", apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&dto.AuditContext{UserID: user.ID}, "oauth_token", "oauth", map[string]interface{}{
		"client_id": clientID,
	})

	return accessToken, "", nil
}

// UserinfoByID 通过已验证的 userID 获取用户信息（中间件已校验 token 有效性）
func (s *oauthService) UserinfoByID(userID uint) (*repository.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	return user, nil
}

func (s *oauthService) GetApplication(id uint) (*repository.OAuthApplication, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, apperrors.ErrOAuthInvalidClient
	}
	return app, nil
}

func (s *oauthService) UpdateApplication(id uint, name, redirectURIs string, auditCtx dto.AuditContext) (*repository.OAuthApplication, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, apperrors.ErrOAuthInvalidClient
	}
	for _, uri := range strings.Split(redirectURIs, ",") {
		trimmed := strings.TrimSpace(uri)
		if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
			return nil, apperrors.ErrOAuthInvalidRedirectURI.WithDetails(map[string]interface{}{
				"reason": "redirect URI must use http or https protocol",
			})
		}
		if err := validateRedirectURIIsPublic(trimmed); err != nil {
			return nil, apperrors.ErrOAuthInvalidRedirectURI.WithDetails(map[string]interface{}{
				"reason": err.Error(),
			})
		}
	}
	app.Name = name
	app.RedirectURIs = redirectURIs
	if err := s.appRepo.Update(app); err != nil {
		return nil, apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&auditCtx, "update_oauth_application", "oauth", map[string]interface{}{
		"application_id":   id,
		"application_name": name,
	})
	return app, nil
}

func (s *oauthService) DeleteApplication(id uint, auditCtx dto.AuditContext) error {
	if err := s.appRepo.Delete(id); err != nil {
		return apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&auditCtx, "delete_oauth_application", "oauth", map[string]interface{}{
		"application_id": id,
	})
	return nil
}

func (s *oauthService) ListApplications(offset, pageSize int) ([]repository.OAuthApplication, int64, error) {
	return s.appRepo.List(offset, pageSize)
}

func isValidRedirectURI(registeredURIs, requestedURI string) bool {
	if registeredURIs == "" || requestedURI == "" {
		return false
	}
	// 拒绝非 HTTP(S) 协议的 URI（防止 javascript: 等注入）
	if !strings.HasPrefix(requestedURI, "http://") && !strings.HasPrefix(requestedURI, "https://") {
		return false
	}
	requestedNormalized := normalizeURI(requestedURI)
	for _, uri := range strings.Split(registeredURIs, ",") {
		if normalizeURI(strings.TrimSpace(uri)) == requestedNormalized {
			return true
		}
	}
	return false
}

// normalizeURI 规范化 URI：去除尾部斜杠，统一 scheme+host 大小写
func normalizeURI(rawURI string) string {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return rawURI
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	path := strings.TrimRight(parsed.Path, "/")
	if path == "" {
		path = "/"
	}
	parsed.Path = path
	return parsed.String()
}

// isValidScope 验证请求的 scope 是否全部在应用注册的 scopes 范围内
func isValidScope(registeredScopes, requestedScope string) bool {
	if registeredScopes == "" {
		return false
	}
	registered := make(map[string]bool)
	for _, s := range strings.Split(registeredScopes, ",") {
		registered[strings.TrimSpace(s)] = true
	}
	// OAuth 标准用空格分隔，同时兼容逗号分隔
	requested := strings.FieldsFunc(requestedScope, func(r rune) bool {
		return r == ' ' || r == ','
	})
	for _, s := range requested {
		if !registered[strings.TrimSpace(s)] {
			return false
		}
	}
	return true
}

// validateRedirectURIIsPublic 检查 redirect URI 的 host 是否为公网地址，防止 SSRF
func validateRedirectURIIsPublic(rawURI string) error {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return fmt.Errorf("invalid redirect URI: %s", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("redirect URI must have a host")
	}

	// 拒绝明显的主机名模式（开发环境允许 localhost 等本地地址）
	lowerHost := strings.ToLower(host)
	isDev := os.Getenv("GIN_MODE") != "release"
	internalHostnames := []string{"localhost", "0.0.0.0", "::1", "metadata.google.internal", "169.254.169.254"}
	for _, h := range internalHostnames {
		if lowerHost == h {
			if isDev {
				return nil
			}
			return fmt.Errorf("redirect URI must not point to internal address: %s", host)
		}
	}

	// 尝试直接解析 IP（标准格式）
	if ip := net.ParseIP(host); ip != nil {
		if err := checkIPNotInternal(ip, host); err != nil {
			return err
		}
		return nil
	}

	// DNS 解析后检查实际 IP，防止 DNS 重绑定攻击（带超时，避免慢速 DNS 阻塞 goroutine）
	resolveCtx, cancel := context.WithTimeout(context.Background(), dnsResolveTimeout)
	defer cancel()

	addrs, err := dnsResolver.LookupIPAddr(resolveCtx, host)
	if err != nil {
		return fmt.Errorf("could not resolve redirect URI host: %s", host)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("redirect URI host resolved to no addresses: %s", host)
	}
	for _, addr := range addrs {
		if err := checkIPNotInternal(addr.IP, host); err != nil {
			return err
		}
	}

	return nil
}

// checkIPNotInternal 检查 IP 是否为内部/私有地址
func checkIPNotInternal(originalIP net.IP, host string) error {
	ip := originalIP.To4()
	if ip == nil {
		ip = originalIP.To16()
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("redirect URI must not point to internal/private IP: %s", host)
	}
	return nil
}

// verifyPKCECodeVerifier 验证 PKCE code_verifier 是否匹配 code_challenge（RFC 7636, S256 only）
func verifyPKCECodeVerifier(codeVerifier, codeChallenge, method string) bool {
	if method != "S256" {
		return false
	}
	hash := sha256.Sum256([]byte(codeVerifier))
	encoded := base64.RawURLEncoding.EncodeToString(hash[:])
	return encoded == codeChallenge
}
