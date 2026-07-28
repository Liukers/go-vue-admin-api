package util

import (
	"errors"
	"time"
	"go-vue-admin/conf"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 自定义声明结构体并内嵌jwt.RegisteredClaims
type CustomClaims struct {
	UserID          uint   `json:"userId"`
	Username        string `json:"username"`
	RoleID          uint   `json:"roleId"`
	PasswordVersion int    `json:"passwordVersion"`
	jwt.RegisteredClaims
}

type JWT struct {
	SigningKey []byte
}

var (
	TokenExpired     = errors.New("token已过期")
	TokenNotValidYet = errors.New("token尚未生效")
	TokenMalformed   = errors.New("非法token")
	TokenInvalid     = errors.New("无效token")
)

func NewJWT() *JWT {
	return &JWT{
		SigningKey: []byte(conf.GetConfig().JWT.SigningKey),
	}
}

// CreateToken 创建一个新的token
func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

// CreateClaims 创建Claims
func (j *JWT) CreateClaims(baseClaims CustomClaims) CustomClaims {
	expiresTime := time.Duration(conf.GetConfig().JWT.ExpiresTime) * time.Hour
	claims := CustomClaims{
		UserID:          baseClaims.UserID,
		Username:        baseClaims.Username,
		RoleID:          baseClaims.RoleID,
		PasswordVersion: baseClaims.PasswordVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresTime)),
			Issuer:    conf.GetConfig().JWT.Issuer,
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	return claims
}

// ParseToken 解析token
func (j *JWT) ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, TokenMalformed
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, TokenExpired
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, TokenNotValidYet
		default:
			return nil, TokenInvalid
		}
	}
	
	if token != nil {
		if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
			return claims, nil
		}
	}
	return nil, TokenInvalid
}

// RefreshGracePeriod token过期后仍允许刷新的最大宽限期
const RefreshGracePeriod = 24 * time.Hour

// ParseTokenIgnoreExpiry 解析token（校验签名，但忽略过期时间等claims校验，用于刷新宽限场景）
func (j *JWT) ParseTokenIgnoreExpiry(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	}, jwt.WithoutClaimsValidation(), jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		return nil, TokenInvalid
	}
	if claims, ok := token.Claims.(*CustomClaims); ok {
		return claims, nil
	}
	return nil, TokenInvalid
}

// RefreshToken 更新token
func (j *JWT) RefreshToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			token, parseErr := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
				return j.SigningKey, nil
			}, jwt.WithoutClaimsValidation(), jwt.WithValidMethods([]string{"HS256"}))
			if parseErr != nil {
				return "", parseErr
			}
			if claims, ok := token.Claims.(*CustomClaims); ok {
				claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Duration(conf.GetConfig().JWT.ExpiresTime) * time.Hour))
				claims.RegisteredClaims.NotBefore = jwt.NewNumericDate(time.Now())
				return j.CreateToken(*claims)
			}
		}
		return "", err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Duration(conf.GetConfig().JWT.ExpiresTime) * time.Hour))
		claims.RegisteredClaims.NotBefore = jwt.NewNumericDate(time.Now())
		return j.CreateToken(*claims)
	}
	return "", TokenInvalid
}
