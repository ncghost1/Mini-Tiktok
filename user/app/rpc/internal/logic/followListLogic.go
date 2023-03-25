package logic

import (
	"Mini-Tiktok/user/app/rpc/model"
	"Mini-Tiktok/user/app/rpc/model/redisCache"
	"context"
	"gorm.io/gorm"
	"strconv"

	"Mini-Tiktok/user/app/rpc/internal/svc"
	"Mini-Tiktok/user/app/rpc/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type FollowListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFollowListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowListLogic {
	return &FollowListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FollowListLogic) FollowList(in *user.FollowListReq) (*user.FollowListResp, error) {
	userid, err := strconv.ParseUint(in.UserId, 10, 64)
	if err != nil {
		return nil, err
	}

	toUserId, err := strconv.ParseUint(in.ToUserId, 10, 64)
	if err != nil {
		return nil, err
	}

	conn := l.svcCtx.Redis.NewRedisConn()
	defer conn.Close()

	// 1. 先查缓存
	cacheList, err := l.svcCtx.Redis.GetFollowUserList(conn, toUserId)
	if err != nil {
		if err.Error() != redisCache.CACHE_KEY_NOT_EXISTS_MSG {
			return nil, err
		}
	}

	remain := l.svcCtx.Config.CacheConfig.FOLLOWLIST_MAX_CACHE_SIZE - len(cacheList)
	var DbFollowList []model.Follow

	// 2. 若缓存为空或缓存已满则需要到 DB 查询数据
	if remain == 0 || remain == l.svcCtx.Config.CacheConfig.FOLLOWLIST_MAX_CACHE_SIZE {
		err = l.svcCtx.Db.Find(&DbFollowList, &model.Follow{Follower: toUserId}).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return nil, err
			}
		}
	}

	var followList []*user.User

	// 3. 遍历从缓存得到的列表并将用户信息加入到 followerList
	for _, v := range cacheList {
		id, err := strconv.ParseUint(v[0], 10, 64)
		if err != nil {
			return nil, err
		}

		username := v[1]

		followCount, err := strconv.ParseInt(v[2], 10, 64)
		if err != nil {
			return nil, err
		}

		followerCount, err := strconv.ParseInt(v[3], 10, 64)
		if err != nil {
			return nil, err
		}

		isfollow := false
		count := int64(0)
		db := l.svcCtx.Db.Model(&model.Follow{}).Where(model.Follow{Follower: userid, Following: id}).Count(&count)
		if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
			return nil, db.Error
		}
		if count > 0 {
			isfollow = true // 已关注
		}

		userInfo := &user.User{
			FollowCount:   followCount,
			FollowerCount: followerCount,
			ID:            id,
			IsFollow:      isfollow,
			Name:          username,
		}

		followList = append(followList, userInfo)
	}

	// 4. 遍历从数据库得到的列表并将用户信息加入到 followerList
	for _, v := range DbFollowList {
		id := v.Following

		getUserLogic := NewGetUserLogic(l.ctx, l.svcCtx)
		u, err := getUserLogic.GetUser(&user.GetUserReq{
			UserID:  strconv.FormatUint(userid, 10),
			QueryID: strconv.FormatUint(id, 10),
		})
		if err != nil {
			return nil, err
		}

		userInfo := &user.User{
			FollowCount:   u.User.FollowCount,
			FollowerCount: u.User.FollowerCount,
			ID:            u.User.ID,
			IsFollow:      u.User.IsFollow,
			Name:          u.User.Name,
		}

		followList = append(followList, userInfo)
	}

	return &user.FollowListResp{
		StatusCode: STATUS_SUCCESS,
		StatusMsg:  STATUS_SUCCESS_MSG,
		UserList:   followList,
	}, nil
}
