package auth

import (
	"errors"
	"time"
)

// TokenType 用来区分访问令牌和刷新令牌。
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

var (
	// ErrInvalidToken 表示令牌签名、格式或有效期校验失败。
	ErrInvalidToken = errors.New("invalid token")
)

// Identity 是发放令牌时需要写入的最小身份信息。
type Identity struct {
	UserID   int64
	Username string
	Role     string
}

// Claims 是应用层关心的令牌载荷。
type Claims struct {
	UserID    int64
	Username  string
	Role      string
	TokenType TokenType
	ExpiresAt time.Time
}

// TokenPair 是登录或刷新接口返回的令牌对。
type TokenPair struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	AccessExpiresIn  int64  `json:"accessExpiresIn"`
	RefreshExpiresIn int64  `json:"refreshExpiresIn"`
	TokenType        string `json:"tokenType"`
}

// TokenManager 定义令牌签发与解析能力。
type TokenManager interface {
	Issue(identity Identity) (TokenPair, error)
	Parse(token string) (*Claims, error)
}
