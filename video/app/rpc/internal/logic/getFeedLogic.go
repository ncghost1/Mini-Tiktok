package logic

import (
	"Mini-Tiktok/user/app/rpc/userrpc"
	"Mini-Tiktok/video/app/rpc/internal/svc"
	"Mini-Tiktok/video/app/rpc/model"
	"Mini-Tiktok/video/app/rpc/video"
	"context"
	"encoding/json"
	"gorm.io/gorm"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFeedLogic {
	return &GetFeedLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetFeed 返回按投稿时间倒序的视频列表，默认每次推送 30 条视频
func (l *GetFeedLogic) GetFeed(in *video.FeedReq) (*video.FeedResp, error) {
	userid, err := strconv.ParseUint(in.UserId, 10, 64)
	if err != nil {
		return nil, err
	}
	latestTime := in.LatestTime
	limit := in.Limit
	nextTime := latestTime

	if latestTime == EMPTY_NEXT_TIME {
		return &video.FeedResp{
			NextTime:   nextTime,
			StatusCode: STATUS_SUCCESS,
			StatusMsg:  STATUS_SUCCESS_MSG,
			VideoList:  nil,
		}, nil
	}

	conn := l.svcCtx.Redis.NewRedisConn()
	defer conn.Close()

	// 1. 先从 Redis 的 Feed 缓存获取时间 <= latestTime 的 limit 条视频信息
	feed, err := l.svcCtx.Redis.GetFeed(conn, latestTime, limit)
	if err != nil {
		return nil, err
	}
	var modelVideoList []model.Video
	for _, b := range feed {
		var v model.Video
		err = json.Unmarshal(b, &v)
		if err != nil {
			return nil, err
		}
		modelVideoList = append(modelVideoList, v)
	}

	// 2. 从 Redis 中获取 feed 后，还剩余多少条视频信息要返回，就要从数据库中拿
	remain := int(limit) - len(modelVideoList)
	if remain > 0 {
		var remainList []model.Video
		maxT := latestTime
		if len(modelVideoList) > 0 {
			maxT = modelVideoList[len(modelVideoList)-1].CreateTime
		}

		err = l.svcCtx.Db.Find(&remainList, "create_time <= ?", maxT).Limit(remain).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				nextTime = EMPTY_NEXT_TIME // 剩余视频不足以完成下一次推送请求，提前将返回的 nextTime 设为 0（节省下一次需要查 DB 的消耗）
			} else {
				return nil, err
			}
		}

		modelVideoList = append(modelVideoList, remainList...)
	}

	// 视频列表为空，则直接返回
	if len(modelVideoList) == 0 {
		return &video.FeedResp{
			NextTime:   nextTime,
			StatusCode: STATUS_SUCCESS,
			StatusMsg:  STATUS_SUCCESS_MSG,
			VideoList:  nil,
		}, nil
	}

	if nextTime != EMPTY_NEXT_TIME {
		nextTime = modelVideoList[len(modelVideoList)-1].CreateTime
	}

	// 3. 从 modelVideoList 中的 userid 获取 user 信息,以及通过 videoId 获取评论数，点赞数，用户是否点赞信息
	videoList := make([]*video.Video, len(modelVideoList))
	for i, v := range modelVideoList {

		// 获取 user 信息
		r, err := l.svcCtx.UserRpc.GetUser(l.ctx, &userrpc.GetUserReq{
			UserID:  strconv.FormatUint(userid, 10),
			QueryID: strconv.FormatUint(v.UserId, 10),
		})
		if err != nil {
			return nil, err
		}
		userInfo := r.User

		// 一次性获取点赞数，评论数，用户是否点赞过视频的缓存数据
		favCount, comCount, isFavor, err := l.svcCtx.Redis.GetExFavComCountIsFavor(conn, v.Id, v.UserId,
			l.svcCtx.Config.CacheConfig.FAVORITE_CACHE_TTL,
			l.svcCtx.Config.CacheConfig.COMMENT_CACHE_TTL,
		)
		if err != nil {
			return nil, err
		}
		// 如果缓存未找到，还需要查库，以下相同
		if favCount == COUNT_NOT_FOUND {
			err := l.svcCtx.Db.Model(&model.Favorite{}).Where(&model.Favorite{VideoId: v.Id}).Count(&favCount).Error
			if err != nil {
				return nil, err
			}

			// 将更新 Redis 命令写入缓冲区，后续更新命令一起调用 Flush() 提交，节省 RTT
			err = l.svcCtx.Redis.SendSetExFavorCount(conn, v.Id, favCount, l.svcCtx.Config.CacheConfig.FAVORITE_CACHE_TTL)
			if err != nil {
				return nil, err
			}
		}

		if comCount == COUNT_NOT_FOUND {
			err := l.svcCtx.Db.Model(&model.Comment{}).Where(&model.Comment{VideoId: v.Id}).Count(&comCount).Error
			if err != nil {
				return nil, err
			}

			// 将更新 Redis 命令写入缓冲区，后续更新命令一起调用 Flush() 提交，节省 RTT
			err = l.svcCtx.Redis.SendSetExCommentCount(conn, v.Id, comCount, l.svcCtx.Config.CacheConfig.COMMENT_CACHE_TTL)
			if err != nil {
				return nil, err
			}
		}

		if !isFavor {
			var cnt int64
			err = l.svcCtx.Db.Model(&model.Favorite{}).Where(&model.Favorite{UserId: userid, VideoId: v.Id}).Count(&cnt).Error
			if err != nil {
				return nil, err
			}
			if cnt > 0 {
				isFavor = true
			}
		}

		vid := &video.Video{
			Author: &video.User{
				FollowCount:   userInfo.FollowCount,
				FollowerCount: userInfo.FollowerCount,
				ID:            userInfo.ID,
				IsFollow:      userInfo.IsFollow,
				Name:          userInfo.Name,
			},
			CommentCount:  comCount,
			CoverURL:      v.CoverUrl,
			FavoriteCount: favCount,
			ID:            v.Id,
			IsFavorite:    isFavor,
			PlayURL:       v.PlayUrl,
			Title:         v.Title,
		}
		videoList[i] = vid
	}

	err = conn.Flush()
	if err != nil {
		return nil, err
	}

	return &video.FeedResp{
		NextTime:   nextTime,
		StatusCode: STATUS_SUCCESS,
		StatusMsg:  STATUS_SUCCESS_MSG,
		VideoList:  videoList,
	}, nil
}
