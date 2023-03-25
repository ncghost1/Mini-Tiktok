package logic

import (
	"Mini-Tiktok/user/app/rpc/userrpc"
	"Mini-Tiktok/video/app/rpc/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ncghost1/snowflake-go"
	"strconv"
	"time"

	"Mini-Tiktok/video/app/rpc/internal/svc"
	"Mini-Tiktok/video/app/rpc/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type CommentActionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCommentActionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentActionLogic {
	return &CommentActionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CommentActionLogic) CommentAction(in *video.CommentReq) (*video.CommentResp, error) {

	// 提取请求参数
	videoId, err := strconv.ParseUint(in.VideoId, 10, 64)
	if err != nil {
		return nil, err
	}

	userid, err := strconv.ParseUint(in.UserId, 10, 64)
	if err != nil {
		return nil, err
	}

	switch in.ActionType {
	case COMMENT_UPDATE: // 评论操作
		createTime := time.Now().Unix()
		sf, err := snowflake.New(l.svcCtx.Config.WorkerId)
		if err != nil {
			return nil, err
		}
		commentId, err := sf.Generate() // 使用雪花算法生成评论id
		if err != nil {
			return nil, err
		}
		comment := model.Comment{
			Id:         commentId,
			UserId:     userid,
			VideoId:    videoId,
			Content:    in.Content,
			CreateTime: createTime,
		}
		commentJson, err := json.Marshal(comment)
		if err != nil {
			return nil, err
		}

		// 先写入数据库
		err = l.svcCtx.Db.Create(&comment).Error
		if err != nil {
			return nil, err
		}

		// 再更新缓存
		// （为什么不是删缓存？因为评论 id 保证唯一，不涉及冲突创建同个评论信息）
		// （这个项目只有创建和删除评论操作，但是如果说有更新评论的操作则需要删缓存）
		conn := l.svcCtx.Redis.NewRedisConn()
		defer conn.Close()
		err = l.svcCtx.Redis.AddComment(conn, videoId, commentId, createTime, commentJson)
		if err != nil {
			return nil, err
		}

		r, err := l.svcCtx.UserRpc.GetUser(l.ctx, &userrpc.GetUserReq{
			UserID:  in.UserId,
			QueryID: in.UserId,
		})
		if err != nil {
			return nil, err
		}

		// 将时间戳转换为 mm-dd
		user := r.User
		utcZone := time.FixedZone("UTC", 8*60*60)
		time.Local = utcZone
		t := time.Now().Unix()
		now := time.Unix(t, 0)
		createDate := fmt.Sprintf("%02d-", now.Month()) + fmt.Sprintf("%02d", now.Day())

		return &video.CommentResp{
			StatusCode: STATUS_SUCCESS,
			StatusMsg:  STATUS_SUCCESS_MSG,
			Comment: &video.Comment{
				Content:    in.Content,
				CreateDate: createDate,
				ID:         commentId,
				User: &video.User{
					FollowCount:   user.FollowCount,
					FollowerCount: user.FollowerCount,
					ID:            user.ID,
					IsFollow:      user.IsFollow,
					Name:          user.Name,
				},
			},
		}, nil
	case COMMENT_DELETE: // 删除评论操作

		// 先修改数据库
		err = l.svcCtx.Db.Where("id = ?", in.CommentId).Delete(model.Comment{}).Error
		if err != nil {
			return nil, err
		}

		commentId, err := strconv.ParseUint(in.CommentId, 10, 64)
		if err != nil {
			return nil, err
		}

		// 删除缓存
		conn := l.svcCtx.Redis.NewRedisConn()
		defer conn.Close()
		err = l.svcCtx.Redis.DelComment(conn, videoId, commentId, l.svcCtx.Config.CacheConfig.COMMENT_CACHE_TTL)
		if err != nil {
			return nil, err
		}

		return &video.CommentResp{
			StatusCode: STATUS_SUCCESS,
			StatusMsg:  STATUS_SUCCESS_MSG,
			Comment:    nil,
		}, nil
	default:
		return nil, errors.New(STATUS_FAIL_PARAM_MSG)
	}
}
