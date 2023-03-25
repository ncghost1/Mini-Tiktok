package logic

import (
	"Mini-Tiktok/user/app/rpc/internal/logic/utils"
	"Mini-Tiktok/user/app/rpc/internal/svc"
	"Mini-Tiktok/user/app/rpc/model"
	"Mini-Tiktok/user/app/rpc/user"
	"context"
	"github.com/ncghost1/snowflake-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Register 处理用户注册
func (l *RegisterLogic) Register(in *user.RegisterReq) (*user.RegisterResp, error) {
	username := in.Username
	password := in.Password
	exist := int64(0)
	err := l.svcCtx.Db.Model(&model.User{}).Where(&model.User{Username: username}).Count(&exist).Error
	if err != nil {
		return nil, err
	}

	if exist > 0 {
		return &user.RegisterResp{
			StatusCode: STATUS_FAIL,
			StatusMsg:  STATUS_USER_EXISTS_MSG,
			UserID:     0,
		}, nil
	}
	sf, err := snowflake.New(l.svcCtx.Config.WorkerId)
	if err != nil {
		return nil, err
	}
	uuid, err := sf.Generate() // 雪花算法生成分布式id
	if err != nil {
		return nil, err
	}

	userInfo := &model.User{
		Id:       uuid,
		Username: username,
		Password: utils.BcryptHash(password),
	}

	// 插入注册用户记录
	err = l.svcCtx.Db.Create(&userInfo).Error
	if err != nil {
		return nil, err
	}

	return &user.RegisterResp{
		StatusCode: STATUS_SUCCESS,
		StatusMsg:  STATUS_SUCCESS_MSG,
		UserID:     userInfo.Id,
	}, nil
}
