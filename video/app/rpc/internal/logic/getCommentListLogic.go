package logic

import (
	"Mini-Tiktok/user/app/rpc/userrpc"
	"Mini-Tiktok/video/app/rpc/model"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"Mini-Tiktok/video/app/rpc/internal/svc"
	"Mini-Tiktok/video/app/rpc/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCommentListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCommentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCommentListLogic {
	return &GetCommentListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetCommentListLogic) GetCommentList(in *video.CommentListReq) (*video.CommentListResp, error) {
	videoId, err := strconv.ParseUint(in.VideoId, 10, 64)
	var commentList []*video.Comment
	var modelComList []model.Comment
	if err != nil {
		return &video.CommentListResp{
			StatusCode:  STATUS_FAIL,
			StatusMsg:   STATUS_FAIL_PARAM_MSG,
			CommentList: nil,
		}, nil
	}

	conn := l.svcCtx.Redis.NewRedisConn()
	defer conn.Close()

	// 先从缓存获取视频最新评论
	cacheJson, exists, err := l.svcCtx.Redis.GetExCommentList(conn, videoId, l.svcCtx.Config.CacheConfig.COMMENT_CACHE_TTL)
	if err != nil {
		return nil, err
	}
	if exists {
		for _, v := range cacheJson {
			var comment model.Comment
			err = json.Unmarshal(v, &comment)
			if err != nil {
				return nil, err
			}
			modelComList = append(modelComList, comment)
		}
	}

	latestTime := time.Now().Unix()
	if len(modelComList) > 0 {
		latestTime = modelComList[len(modelComList)-1].CreateTime
	}

	// 如果不存在该视频的缓存，或者缓存列表已满，则需要到数据库中查找该视频是否还有更早的评论
	// 设计理论上不会出现缓存列表未满时还有评论信息的情况，除非缓存更新出现失败
	remain := l.svcCtx.Config.CacheConfig.VIDEO_COMMENT_MAX_CACHE_SIZE - len(modelComList)
	if !exists || remain == l.svcCtx.Config.CacheConfig.VIDEO_COMMENT_MAX_CACHE_SIZE {
		// 从数据库查找比缓存评论要早的评论
		var comList []model.Comment
		err = l.svcCtx.Db.Where("video_id = ? and create_time <= ?", videoId, latestTime).Find(&comList).Error
		if err != nil {
			return nil, err
		}

		for i, v := range comList {
			marshal, err := json.Marshal(&v)
			if err != nil {
				return nil, err
			}

			if i <= remain {
				// 以下将添加评论缓存的 Redis 命令添加到发送缓冲区
				err := l.svcCtx.Redis.SendAddCommentList(conn, videoId, v.Id, v.CreateTime, l.svcCtx.Config.CacheConfig.COMMENT_CACHE_TTL)
				if err != nil {
					return nil, err
				}

				err = l.svcCtx.Redis.SendSetExCommentJson(conn, v.Id, marshal, l.svcCtx.Config.CacheConfig.COMMENT_CACHE_TTL)
				if err != nil {
					return nil, err
				}
			}
		}
		modelComList = append(modelComList, comList...)
	}

	for _, v := range modelComList {
		// 通过评论的 userid 查 user 信息
		queryUserId := strconv.FormatUint(v.UserId, 10)
		r, err := l.svcCtx.UserRpc.GetUser(l.ctx, &userrpc.GetUserReq{UserID: in.UserId, QueryID: queryUserId})
		if err != nil {
			return nil, err
		}

		// 将时间戳转换为 mm-dd
		utcZone := time.FixedZone("UTC", 8*60*60)
		time.Local = utcZone
		t := time.Unix(v.CreateTime, 0)
		createDate := fmt.Sprintf("%02d-", t.Month()) + fmt.Sprintf("%02d", t.Day())

		var u *video.User
		if r.StatusCode == STATUS_SUCCESS {
			u = &video.User{
				FollowCount:   r.User.FollowCount,
				FollowerCount: r.User.FollowerCount,
				ID:            r.User.ID,
				IsFollow:      r.User.IsFollow,
				Name:          r.User.Name,
			}
		} else {
			u = nil
		}

		comment := &video.Comment{
			Content:    v.Content,
			CreateDate: createDate,
			ID:         v.Id,
			User:       u,
		}
		commentList = append(commentList, comment)
	}

	// 将缓冲区命令发送给 Redis（如果有）
	err = conn.Flush()
	if err != nil {
		return nil, err
	}

	return &video.CommentListResp{
		StatusCode:  STATUS_SUCCESS,
		StatusMsg:   STATUS_SUCCESS_MSG,
		CommentList: commentList,
	}, nil
}
