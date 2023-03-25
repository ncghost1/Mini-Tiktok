package logic

import (
	"Mini-Tiktok/user/app/rpc/model"
	"context"
	"errors"
	"gorm.io/gorm"
	"strconv"
	"time"

	"Mini-Tiktok/user/app/rpc/internal/svc"
	"Mini-Tiktok/user/app/rpc/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type FollowActionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFollowActionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowActionLogic {
	return &FollowActionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FollowActionLogic) FollowAction(in *user.FollowActionReq) (*user.FollowActionResp, error) {
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

	switch in.ActionType {
	case OP_FOLLOW:
		createTime := time.Now().Unix()

		// 查找用户是否存在
		err = l.svcCtx.Db.Model(model.User{}).Where(&model.User{Id: toUserId}).Take(&model.User{}).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound { // 用户不存在
				return &user.FollowActionResp{
					StatusCode: STATUS_FAIL,
					StatusMsg:  STATUS_USER_NOTEXIST_MSG,
				}, nil
			}
			return nil, err
		}

		err = l.svcCtx.Db.Model(model.User{}).Where(&model.User{Id: toUserId}).Take(&model.User{}).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound { // 用户不存在
				return &user.FollowActionResp{
					StatusCode: STATUS_FAIL,
					StatusMsg:  STATUS_USER_NOTEXIST_MSG,
				}, nil
			}
			return nil, err
		}

		// 先更新 DB
		// 我们不用检查是否用户是否已关注该对象，
		// 因为关注关系双方的 id 做联合主键，不会插入相同的关注记录
		err := l.svcCtx.Db.Create(&model.Follow{
			Follower:   userid,
			Following:  toUserId,
			CreateTime: createTime,
		}).Error
		if err != nil {
			return nil, err
		}

		// 更新双方用户的最新关注列表以及最新粉丝列表
		err = l.svcCtx.Redis.AddFollowUserList(conn, userid, toUserId, createTime)
		if err != nil {
			return nil, err
		}

	case OP_CANCEL_FOLLOW:
		err := l.svcCtx.Db.Delete(&model.Follow{}, &model.Follow{
			Follower:  userid,
			Following: toUserId,
		}).Error
		if err != nil {
			return nil, err
		}

		// 更新双方用户的最新关注列表以及最新粉丝列表
		err = l.svcCtx.Redis.RemFollowUserList(conn, userid, toUserId)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New(STATUS_FAIL_PARAM_MSG)
	}
	return &user.FollowActionResp{
		StatusCode: STATUS_SUCCESS,
		StatusMsg:  STATUS_SUCCESS_MSG,
	}, nil
}
