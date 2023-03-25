package logic

import (
	"Mini-Tiktok/jwt/app/rpc/Jwt"
	"Mini-Tiktok/jwt/app/rpc/internal/svc"
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v4"

	"github.com/zeromicro/go-zero/core/logx"
)

type ParseTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewParseTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ParseTokenLogic {
	return &ParseTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// secret 用于获取密钥
func (l *ParseTokenLogic) secret() jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		return []byte(l.svcCtx.Config.JwtConfig.AccessSecret), nil
	}
}

func (l *ParseTokenLogic) ParseToken(in *Jwt.ParseTokenReq) (*Jwt.ParseTokenResp, error) {
	t := in.Token
	token, err := jwt.ParseWithClaims(t, &Claims{}, l.secret())
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, TokenMalformed
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, TokenExpired
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, TokenNotValidYet
			} else {
				return nil, TokenInvalid
			}
		}
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return &Jwt.ParseTokenResp{
			UserID:       claims.UserID,
			AccessExpire: claims.ExpiresAt.Unix(),
		}, nil
	}
	return nil, errors.New("couldn't handle this token")
}
