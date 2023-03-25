package logic

import (
	"Mini-Tiktok/api/internal/svc"
	"Mini-Tiktok/api/internal/types"
	"Mini-Tiktok/jwt/app/rpc/Jwt"
	"Mini-Tiktok/user/app/rpc/userrpc"
	"context"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserLogic {
	return &GetUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserLogic) GetUser(req *types.GetUserReq) (resp *types.GetUserResp, err error) {

	// 通过鉴权服务解析 token
	token, err := l.svcCtx.JwtRpc.ParseToken(l.ctx, &Jwt.ParseTokenReq{Token: req.Token})
	if err != nil {
		return &types.GetUserResp{
			Response: types.Response{
				StatusCode: STATUS_FAIL,
				StatusMsg:  STATUS_FAIL_TOKEN_MSG,
			}, User: nil,
		}, nil
	}
	userid := token.UserID

	// 获取用户信息
	r, err := l.svcCtx.UserRpc.GetUser(l.ctx, &userrpc.GetUserReq{
		UserID:  userid,
		QueryID: req.UserID,
	})
	if err != nil {
		return nil, err
	}
	var userInfo *types.User
	if r.User == nil {
		userInfo = nil
	} else {
		userInfo = &types.User{
			FollowCount:   r.User.FollowCount,
			FollowerCount: r.User.FollowerCount,
			ID:            r.User.ID,
			IsFollow:      r.User.IsFollow,
			Name:          r.User.Name,
		}
	}

	return &types.GetUserResp{
		Response: types.Response{
			StatusCode: r.StatusCode,
			StatusMsg:  r.StatusMsg,
		},
		User: userInfo,
	}, nil

}
