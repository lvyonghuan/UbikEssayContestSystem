package token

import (
	"errors"
	"main/conf"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lvyonghuan/Ubik-Util/uerr"
)

type ResponseToken struct {
	Token        string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

var (
	// FIXME 为不同的role准备不同的密钥
	accessTokenKey  = []byte("your-256-bit-secret-access")
	refreshTokenKey = []byte("your-256-bit-secret-refresh")

	accessTokenExpiry  = time.Duration(24) * time.Hour
	refreshTokenExpiry = time.Duration(2*24) * time.Hour
)

func InitJWT(tokenConf conf.TokenConfig) error {
	accessTokenExpiry = time.Duration(tokenConf.AccessTokenExpire) * time.Hour
	refreshTokenExpiry = time.Duration(tokenConf.RefreshTokenExpire) * time.Hour

	accessTokenKey = []byte(os.Getenv("Ubik_JWT_Access_Key"))
	refreshTokenKey = []byte(os.Getenv("Ubik_JWT_Refresh_Key"))

	if len(accessTokenKey) == 0 || len(refreshTokenKey) == 0 {
		err := uerr.NewError(errors.New("JWT keys cannot be empty"))
		return err
	}
	return nil
}

// UserClaims 增加了 Role 字段
type UserClaims struct {
	ID   int64  `json:"id"`
	Role string `json:"role"` // 新增角色字段
	jwt.RegisteredClaims
}

// GenTokenAndRefreshToken 接收 userID 和 role
func GenTokenAndRefreshToken(userID int64, role string) (ResponseToken, error) {
	// 构造 Access Token 载荷
	accessClaims := UserClaims{
		ID:   userID,
		Role: role, // 赋值角色
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "ubik",
			Subject:   "access_token",
		},
	}

	// 构造 Refresh Token 载荷
	refreshClaims := UserClaims{
		ID:   userID,
		Role: role, // 通常 Refresh Token 也可以携带角色，方便静默刷新时校验
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "ubik",
			Subject:   "refresh_token",
		},
	}

	accessToken, err := genToken(accessClaims, accessTokenKey)
	if err != nil {
		return ResponseToken{}, uerr.NewError(err)
	}
	refreshToken, err := genToken(refreshClaims, refreshTokenKey)
	if err != nil {
		return ResponseToken{}, uerr.NewError(err)
	}

	return ResponseToken{
		Token:        accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func ParseAccessToken(tokenString string) (*UserClaims, error) {
	return parseJWT(tokenString, accessTokenKey)
}

func ParseRefreshToken(tokenString string) (*UserClaims, error) {
	return parseJWT(tokenString, refreshTokenKey)
}

func genToken(claims jwt.Claims, key []byte) (string, error) {
	if len(key) == 0 {
		// 建议此处返回 error 而不是直接 Fatal 退出进程，除非这是不可恢复的启动错误
		return "", uerr.NewError(errors.New("JWT key cannot be empty"))
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

func parseJWT(tokenString string, key []byte) (*UserClaims, error) {
	claims := &UserClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})

	if err != nil {
		return nil, uerr.NewError(err)
	}
	if !token.Valid {
		return nil, uerr.NewError(errors.New("JWT token is invalid"))
	}

	return claims, nil
}
