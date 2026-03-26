/*
我们正在开发一个 Go Web 应用（如 Gin 框架），用户登录后需要生成 JWT，后续请求通过携带 JWT 进行身份验证。

1. 初始化 JWTManager
在应用启动时，从配置中读取密钥和有效期，创建 JWTManager 实例：
```golang
jwtManager := auth.NewJWTManager(
    config.JWTSecret,          // "my-secret-key"
    config.AccessTokenTTL,     // 15 * time.Minute
    config.RefreshTokenTTL,    // 7 * 24 * time.Hour
)
```

2. 登录接口：签发令牌
用户发送登录请求（用户名/密码），验证通过后，获取用户身份信息（UserID、Username、Role），调用 Issue 签发一对令牌：
```golang
func loginHandler(c *gin.Context) {
    // 1. 校验用户名密码
    user := validateLogin(c)
    if user == nil {
        c.JSON(401, gin.H{"error": "invalid credentials"})
        return
    }

    // 2. 构建身份信息
    identity := domainauth.Identity{
        UserID:   user.ID,
        Username: user.Username,
        Role:     user.Role,
    }

    // 3. 签发令牌
    tokenPair, err := jwtManager.Issue(identity)
    if err != nil {
        c.JSON(500, gin.H{"error": "token issuance failed"})
        return
    }

    // 4. 返回给客户端
    c.JSON(200, tokenPair) // 包含 accessToken, refreshToken, expires_in 等
}
```

3. 访问保护路由：解析访问令牌

```golang
func AuthMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
            c.AbortWithStatusJSON(401, gin.H{"error": "missing token"})
            return
        }
        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

        claims, err := jwtManager.Parse(tokenStr)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }

        // 检查令牌类型是否为 access
        if claims.TokenType != domainauth.TokenTypeAccess {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token type"})
            return
        }

        // 将用户信息存入上下文，供后续处理使用
        c.Set("userID", claims.UserID)
        c.Set("username", claims.Username)
        c.Set("role", claims.Role)
        c.Next()
    }
}
```

4. 刷新令牌接口
当访问令牌过期，客户端可以使用刷新令牌换取新的令牌对。后端接收刷新令牌，调用 Parse 验证，并确保 TokenType 为 refresh，然后重新签发新的令牌对：

```json
func refreshHandler(c *gin.Context) {
    var req struct {
        RefreshToken string `json:"refreshToken"`
    }
    if err := c.BindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "invalid request"})
        return
    }

    claims, err := jwtManager.Parse(req.RefreshToken)
    if err != nil || claims.TokenType != domainauth.TokenTypeRefresh {
        c.JSON(401, gin.H{"error": "invalid refresh token"})
        return
    }

    // 根据 claims 中的用户信息重新签发（可能需要验证用户是否仍有效）
    identity := domainauth.Identity{
        UserID:   claims.UserID,
        Username: claims.Username,
        Role:     claims.Role,
    }
    newTokenPair, err := jwtManager.Issue(identity)
    if err != nil {
        c.JSON(500, gin.H{"error": "token refresh failed"})
        return
    }

    c.JSON(200, newTokenPair)
}
```

5. 注销/退出登录
```json
如果需要使令牌失效，常见的做法是在服务端维护一个黑名单或使用短有效期令牌。本实现中未提供显式注销，通常配合访问令牌的较短有效期（如15分钟）和刷新令牌的可撤销机制实现。
```
*/

package auth

import (
	"fmt"
	"time"

	domainauth "kubeclaw/backend/internal/domain/auth"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager 使用 HMAC-SHA256 签发和校验 JWT。
type JWTManager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

type jwtClaims struct {
	UserID    int64  `json:"uid"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	TokenType string `json:"tokenType"`
	jwt.RegisteredClaims
}

// NewJWTManager 创建 JWT 管理器。
func NewJWTManager(secret string, accessTokenTTL, refreshTokenTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// Issue 生成访问令牌和刷新令牌。
func (m *JWTManager) Issue(identity domainauth.Identity) (domainauth.TokenPair, error) {
	now := time.Now()

	accessToken, err := m.signToken(identity, domainauth.TokenTypeAccess, now, m.accessTokenTTL)
	if err != nil {
		return domainauth.TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken, err := m.signToken(identity, domainauth.TokenTypeRefresh, now, m.refreshTokenTTL)
	if err != nil {
		return domainauth.TokenPair{}, fmt.Errorf("sign refresh token: %w", err)
	}

	return domainauth.TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresIn:  int64(m.accessTokenTTL.Seconds()),
		RefreshExpiresIn: int64(m.refreshTokenTTL.Seconds()),
		TokenType:        "Bearer",
	}, nil
}

// Parse 解析并校验 JWT，失败时统一返回领域错误。
func (m *JWTManager) Parse(token string) (*domainauth.Claims, error) {
	claims := &jwtClaims{}

	parsedToken, err := jwt.ParseWithClaims(token, claims, func(parsed *jwt.Token) (any, error) {
		if _, ok := parsed.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %s", parsed.Method.Alg())
		}

		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !parsedToken.Valid || claims.ExpiresAt == nil {
		return nil, domainauth.ErrInvalidToken
	}

	tokenType := domainauth.TokenType(claims.TokenType)
	if tokenType != domainauth.TokenTypeAccess && tokenType != domainauth.TokenTypeRefresh {
		return nil, domainauth.ErrInvalidToken
	}

	return &domainauth.Claims{
		UserID:    claims.UserID,
		Username:  claims.Username,
		Role:      claims.Role,
		TokenType: tokenType,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func (m *JWTManager) signToken(identity domainauth.Identity, tokenType domainauth.TokenType, now time.Time, ttl time.Duration) (string, error) {
	claims := jwtClaims{
		UserID:    identity.UserID,
		Username:  identity.Username,
		Role:      identity.Role,
		TokenType: string(tokenType),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   identity.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

var _ domainauth.TokenManager = (*JWTManager)(nil)
