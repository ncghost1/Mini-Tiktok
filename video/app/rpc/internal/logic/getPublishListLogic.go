package logic

import (
	"Mini-Tiktok/user/app/rpc/userrpc"
	"Mini-Tiktok/video/app/rpc/model"
	"context"
	"encoding/json"
	"strconv"
	"time"

	"Mini-Tiktok/video/app/rpc/internal/svc"
	"Mini-Tiktok/video/app/rpc/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPublishListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPublishListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPublishListLogic {
	return &GetPublishListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetPublishListLogic) GetPublishList(in *video.PublishListReq) (*video.PublishListResp, error) {
	userid, err := strconv.ParseUint(in.UserID, 10, 64)
	if err != nil {
		return nil, err
	}
	queryid, err := strconv.ParseUint(in.QueryId, 10, 64)
	if err != nil {
		return nil, err
	}

	var modelVideoList []model.Video
	latestTime := time.Now().Unix()
	conn := l.svcCtx.Redis.NewRedisConn()
	defer conn.Close()

	// 获取缓存中的最新发布视频列表（json格式信息）
	list, exists, err := l.svcCtx.Redis.GetExPublishList(conn, queryid, l.svcCtx.Config.CacheConfig.VIDEO_CACHE_TTL)
	if err != nil {
		return nil, err
	}
	if exists {
		for _, v := range list {
			var vid model.Video
			err = json.Unmarshal(v, &vid)
			if err != nil {
				return nil, err
			}
			modelVideoList = append(modelVideoList, vid)
		}
		latestTime = modelVideoList[len(modelVideoList)-1].CreateTime
	}

	// 如果不存在该用户的缓存，或者缓存列表已满，则需要到数据库中查找该用户是否还有发布的视频
	// 设计理论上不会出现缓存列表未满时还有发布视频的情况，除非缓存更新出现失败
	remain := l.svcCtx.Config.CacheConfig.VIDEO_MAX_CACHE_SIZE - len(modelVideoList)
	if !exists || remain == l.svcCtx.Config.CacheConfig.VIDEO_FAVORITE_MAX_CACHE_SIZE {
		var vidList []model.Video
		err = l.svcCtx.Db.Where("user_id = ? and create_time <= ?", queryid, latestTime).Find(&vidList).Error
		if err != nil {
			return nil, err
		}

		for i, v := range vidList {
			if i <= remain {
				marshal, err := json.Marshal(&v)
				if err != nil {
					return nil, err
				}

				// 以下将添加视频缓存的 Redis 命令添加到发送缓冲区
				err = l.svcCtx.Redis.SendAddVideoInfo(conn, v.UserId, v.Id, marshal, time.Now().Unix(), l.svcCtx.Config.CacheConfig.VIDEO_CACHE_TTL)
				if err != nil {
					return nil, err
				}
			}
		}
		modelVideoList = append(modelVideoList, vidList...)
	}

	videoList := make([]*video.Video, len(modelVideoList))
	for i, v := range modelVideoList {

		// 获取 user 信息
		r, err := l.svcCtx.UserRpc.GetUser(l.ctx, &userrpc.GetUserReq{
			UserID:  strconv.FormatUint(userid, 10),
			QueryID: strconv.FormatUint(v.UserId, 10),
		})
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

	// 将缓冲区的命令发送给 Redis
	err = conn.Flush()
	if err != nil {
		return nil, err
	}

	return &video.PublishListResp{
		StatusCode: STATUS_SUCCESS,
		StatusMsg:  STATUS_SUCCESS_MSG,
		VideoList:  videoList,
	}, nil
}
