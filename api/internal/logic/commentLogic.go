package logic

import (
	"Mini-Tiktok/api/internal/svc"
	"Mini-Tiktok/api/internal/types"
	"Mini-Tiktok/jwt/app/rpc/Jwt"
	"Mini-Tiktok/video/app/rpc/videorpc"
	"context"
	"github.com/zeromicro/go-zero/core/logx"
)

type CommentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentLogic {
	return &CommentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CommentLogic) Comment(req *types.CommentReq) (resp *types.CommentResp, err error) {
	token, err := l.svcCtx.JwtRpc.ParseToken(l.ctx, &Jwt.ParseTokenReq{Token: req.Token})
	if err != nil {
		return &types.CommentResp{
			Response: types.Response{
				StatusCode: STATUS_FAIL,
				StatusMsg:  STATUS_FAIL_TOKEN_MSG,
			},
			Comment: nil,
		}, nil
	}
	userid := token.UserID
	content := ""
	commentId := ""
	if req.CommentText != nil {
		content = *req.CommentText
	}
	if req.CommentId != nil {
		commentId = *req.CommentId
	}

	r, err := l.svcCtx.VideoRpc.CommentAction(l.ctx, &videorpc.CommentReq{
		VideoId:    req.VideoId,
		UserId:     userid,
		ActionType: req.ActionType,
		Content:    content,
		CommentId:  commentId,
	})
	if err != nil {
		return nil, err
	}

	var comment *types.Comment
	comment = nil
	if r.Comment != nil {
		comment = &types.Comment{
			Content:    r.Comment.Content,
			CreateDate: r.Comment.CreateDate,
			ID:         r.Comment.ID,
			User: types.User{
				FollowCount:   r.Comment.User.FollowCount,
				FollowerCount: r.Comment.User.FollowerCount,
				ID:            r.Comment.User.ID,
				IsFollow:      r.Comment.User.IsFollow,
				Name:          r.Comment.User.Name,
			},
		}
	}

	return &types.CommentResp{
		Response: types.Response{
			StatusCode: STATUS_SUCCESS,
			StatusMsg:  STATUS_SUCCESS_MSG,
		},
		Comment: comment,
	}, nil
}
