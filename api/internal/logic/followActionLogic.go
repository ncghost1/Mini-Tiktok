package logic

import (
	"Mini-Tiktok/jwt/app/rpc/Jwt"
	"Mini-Tiktok/user/app/rpc/userrpc"
	"context"

	"Mini-Tiktok/api/internal/svc"
	"Mini-Tiktok/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type FollowActionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFollowActionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowActionLogic {
	return &FollowActionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FollowActionLogic) FollowAction(req *types.FollowActionReq) (resp *types.FollowActionResp, err error) {
	token, err := l.svcCtx.JwtRpc.ParseToken(l.ctx, &Jwt.ParseTokenReq{Token: req.Token})
	if err != nil {
		return &types.FollowActionResp{
			Response: types.Response{
				StatusCode: STATUS_FAIL,
				StatusMsg:  STATUS_FAIL_TOKEN_MSG,
			},
		}, nil
	}
	userid := token.UserID

	if userid == req.ToUserId {
		return &types.FollowActionResp{
			Response: types.Response{
				StatusCode: STATUS_FAIL,
				StatusMsg:  STATUS_FAIL_FOLLOW_SELF,
			},
		}, nil
	}

	r, err := l.svcCtx.UserRpc.FollowAction(l.ctx, &userrpc.FollowActionReq{UserId: userid, ToUserId: req.ToUserId, ActionType: req.ActionType})
	if err != nil {
		return nil, err
	}

	return &types.FollowActionResp{Response: types.Response{
		StatusCode: r.StatusCode,
		StatusMsg:  r.StatusMsg,
	}}, nil
}
