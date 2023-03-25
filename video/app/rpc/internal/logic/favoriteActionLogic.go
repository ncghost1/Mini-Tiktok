package logic

import (
	"Mini-Tiktok/video/app/rpc/internal/svc"
	"Mini-Tiktok/video/app/rpc/model"
	"Mini-Tiktok/video/app/rpc/model/KafkaMessage"
	"Mini-Tiktok/video/app/rpc/video"
	"context"
	"encoding/json"
	"errors"
	"github.com/segmentio/kafka-go"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type FavoriteActionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFavoriteActionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FavoriteActionLogic {
	return &FavoriteActionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// FavoriteAction 处理点赞/取消点赞请求
func (l *FavoriteActionLogic) FavoriteAction(in *video.FavoriteReq) (*video.FavoriteResp, error) {

	// 提取请求参数
	videoId, err := strconv.ParseUint(in.VideoId, 10, 64)
	if err != nil {
		return nil, err
	}

	userid, err := strconv.ParseUint(in.UserId, 10, 64)
	if err != nil {
		return nil, err
	}

	actionType := in.ActionType

	switch actionType {
	case FAVORITE_UPDATE: // 点赞
		createTime := time.Now().Unix()
		favorite := model.Favorite{
			UserId:     userid,
			VideoId:    videoId,
			CreateTime: createTime,
		}

		// 先更新 Redis
		conn := l.svcCtx.Redis.NewRedisConn()
		defer conn.Close()
		err = l.svcCtx.Redis.AddFavorite(conn, videoId, userid, createTime)
		if err != nil {
			return nil, err
		}

		// 制造请求消息，之后将请求写入消息队列，让 DB 异步写入点赞数据
		cols, err := json.Marshal(favorite)
		if err != nil {
			return nil, err
		}
		msg := KafkaMessage.MsgInfo{
			Op:      OP_INSERT,
			Model:   MODEL_FAVORITE,
			Columns: string(cols),
		}
		marshal, err := json.Marshal(msg)
		kafkaMsg := kafka.Message{
			Value: marshal,
		}

		err = l.svcCtx.KafkaWriter.WriteMessages(l.ctx, kafkaMsg)
		if err != nil {
			return nil, err
		}

	case FAVORITE_DELETE: // 取消点赞
		favorite := model.Favorite{
			UserId:  userid,
			VideoId: videoId,
		}

		// 先更新 Redis
		conn := l.svcCtx.Redis.NewRedisConn()
		defer conn.Close()
		err = l.svcCtx.Redis.DelFavorite(conn, videoId, userid,
			l.svcCtx.Config.CacheConfig.FAVORITE_CACHE_TTL,
			l.svcCtx.Config.CacheConfig.FAVORITE_DEL_CACHE_TTL)
		if err != nil {
			return nil, err
		}

		// 制造请求消息，之后将请求写入消息队列，让 DB 异步删除点赞数据
		cols, err := json.Marshal(favorite)
		if err != nil {
			return nil, err
		}
		msg := KafkaMessage.MsgInfo{
			Op:      OP_DELETE,
			Model:   MODEL_FAVORITE,
			Columns: string(cols),
		}
		marshal, err := json.Marshal(msg)
		kafkaMsg := kafka.Message{
			Value: marshal,
		}

		err = l.svcCtx.KafkaWriter.WriteMessages(l.ctx, kafkaMsg)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New(STATUS_FAIL_PARAM_MSG)
	}

	return &video.FavoriteResp{
		StatusCode: STATUS_SUCCESS,
		StatusMsg:  STATUS_SUCCESS_MSG,
	}, nil
}
