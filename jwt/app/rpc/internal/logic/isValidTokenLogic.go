package logic

import (
	"Mini-Tiktok/jwt/app/rpc/Jwt"
	"Mini-Tiktok/jwt/app/rpc/internal/svc"
	"context"
	"github.com/golang-jwt/jwt/v4"

	"github.com/zeromicro/go-zero/core/logx"
)

type IsValidTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIsValidTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IsValidTokenLogic {
	return &IsValidTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// secret 用于获取密钥
func (l *IsValidTokenLogic) secret() jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		return []byte(l.svcCtx.Config.JwtConfig.AccessSecret), nil
	}
}

func (l *IsValidTokenLogic) IsValidToken(in *Jwt.IsValidTokenReq) (*Jwt.IsValidTokenResp, error) {
	t := in.Token
	_, err := jwt.ParseWithClaims(t, &Claims{}, l.secret())
	if err != nil {
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
	}
	return &Jwt.IsValidTokenResp{
		IsValid: true,
	}, nil
}
