package logic

import (
	"Mini-Tiktok/api/internal/svc"
	"Mini-Tiktok/api/internal/types"
	"Mini-Tiktok/jwt/app/rpc/Jwt"
	"Mini-Tiktok/video/app/rpc/videorpc"
	"context"
	"github.com/zeromicro/go-zero/core/logx"
)

type CommentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCommentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentListLogic {
	return &CommentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CommentListLogic) CommentList(req *types.CommentListReq) (resp *types.CommentListResp, err error) {
	token, err := l.svcCtx.JwtRpc.ParseToken(l.ctx, &Jwt.ParseTokenReq{Token: req.Token})
	if err != nil {
		return &types.CommentListResp{
			Response: types.Response{
				StatusCode: STATUS_FAIL,
				StatusMsg:  STATUS_FAIL_TOKEN_MSG,
			},
			CommentList: nil,
		}, nil
	}
	userid := token.UserID

	r, err := l.svcCtx.VideoRpc.GetCommentList(l.ctx, &videorpc.CommentListReq{VideoId: req.VideoId, UserId: userid})
	if err != nil {
		return nil, err
	}

	commentList := make([]types.Comment, len(r.CommentList))
	for i, v := range r.CommentList {
		commentList[i] = types.Comment{
			Content:    v.Content,
			CreateDate: v.CreateDate,
			ID:         v.ID,
			User: types.User{
				FollowCount:   v.User.FollowCount,
				FollowerCount: v.User.FollowerCount,
				ID:            v.User.ID,
				IsFollow:      v.User.IsFollow,
				Name:          v.User.Name,
			},
		}
	}

	return &types.CommentListResp{
		Response: types.Response{
			StatusCode: r.StatusCode,
			StatusMsg:  r.StatusMsg,
		},
		CommentList: commentList,
	}, nil
}
