package logic

import (
	"Mini-Tiktok/user/app/rpc/internal/svc"
	"Mini-Tiktok/user/app/rpc/model"
	"Mini-Tiktok/user/app/rpc/model/redisCache"
	"Mini-Tiktok/user/app/rpc/user"
	"context"
	"gorm.io/gorm"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserLogic {
	return &GetUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetUser 获取用户信息
func (l *GetUserLogic) GetUser(in *user.GetUserReq) (*user.GetUserResp, error) {
	userid, err := strconv.ParseUint(in.UserID, 10, 64)
	queryid, err := strconv.ParseUint(in.QueryID, 10, 64)
	if err != nil {
		return &user.GetUserResp{
			StatusCode: STATUS_FAIL,
			StatusMsg:  STATUS_FAIL_PARAM_MSG,
			User:       nil,
		}, nil
	}

	conn := l.svcCtx.Redis.NewRedisConn()
	defer conn.Close()

	// 1. 查询用户是否关注查询对象
	isfollow := false
	isfollow, err = l.svcCtx.Redis.IsFollow(conn, userid, queryid)
	if err != nil {
		return nil, err
	}

	// 缓存未查到，则到 DB 中查询
	if !isfollow {
		count := int64(0)
		err := l.svcCtx.Db.Model(&model.Follow{}).Where(&model.Follow{Follower: userid, Following: queryid}).Count(&count).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
		if count > 0 {
			isfollow = true // 已关注
		}
	}

	// 2. 从缓存中获取查询对象用户名，关注数，粉丝数
	username, followCnt, followerCnt, err := l.svcCtx.Redis.GetUserInfo(conn, queryid)
	needUpdateCache := false // 是否需要更新缓存

	if err != nil {
		if err.Error() != redisCache.CACHE_KEY_NOT_EXISTS_MSG {
			return nil, err
		}
		needUpdateCache = true
	}

	// 用户名不存在则到 DB 查询
	if username == "" {
		err = l.svcCtx.Db.Model(model.User{}).Select("username").Where(&model.User{Id: queryid}).Take(&username).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound { // 用户不存在
				return &user.GetUserResp{
					StatusCode: STATUS_FAIL,
					StatusMsg:  STATUS_USER_NOTEXIST_MSG,
					User:       nil,
				}, nil
			}
			return nil, err
		}
	}

	// 关注数不存在则到 DB 查询
	if followCnt == COUNT_NOT_FOUND {
		followCnt = 0
		err = l.svcCtx.Db.Model(&model.Follow{}).Where(&model.Follow{Follower: queryid}).Count(&followCnt).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		}
	}

	// 粉丝数不存在则到 DB 查询
	if followerCnt == COUNT_NOT_FOUND {
		followerCnt = 0
		err = l.svcCtx.Db.Model(&model.Follow{}).Where(&model.Follow{Following: queryid}).Count(&followerCnt).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		}

	}

	// 将从 DB 获取到的数据写入缓存
	if needUpdateCache {
		err = l.svcCtx.Redis.SetUserInfo(conn, queryid, username, followCnt, followerCnt)
		if err != nil {
			return nil, err
		}
	}

	UserInfo := &user.User{
		FollowCount:   followCnt,
		FollowerCount: followerCnt,
		ID:            queryid,
		IsFollow:      isfollow,
		Name:          username,
	}

	return &user.GetUserResp{
		StatusCode: STATUS_SUCCESS,
		StatusMsg:  STATUS_SUCCESS_MSG,
		User:       UserInfo,
	}, nil
}
