package logic

import (
	"Mini-Tiktok/jwt/app/rpc/Jwt"
	"Mini-Tiktok/jwt/app/rpc/internal/svc"
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

type Claims struct {
	UserID               string
	jwt.RegisteredClaims //jwt 自带载荷
}

func NewCreateTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTokenLogic {
	return &CreateTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// BuildClaims 构建 payload
func (l *CreateTokenLogic) BuildClaims(userID string, ttl int64) Claims {
	return Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(ttl) * time.Second)), //过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),                                       //签发时间
			NotBefore: jwt.NewNumericDate(time.Now()),                                       //生效时间
		}}
}

func (l *CreateTokenLogic) CreateToken(in *Jwt.CreateTokenReq) (*Jwt.CreateTokenResp, error) {
	claims := l.BuildClaims(in.UserID, in.AccessExpire)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	fmt.Println("token:", token)
	tokenStr, err := token.SignedString([]byte(l.svcCtx.Config.JwtConfig.AccessSecret))
	if err != nil {
		return nil, err
	}
	fmt.Println("token After Sign:", token)
	return &Jwt.CreateTokenResp{Token: tokenStr}, nil
}
