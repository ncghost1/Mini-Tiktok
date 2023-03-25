package logic

import (
	"Mini-Tiktok/user/app/rpc/internal/logic/utils"
	"Mini-Tiktok/user/app/rpc/internal/svc"
	"Mini-Tiktok/user/app/rpc/model"
	"Mini-Tiktok/user/app/rpc/user"
	"context"
	"gorm.io/gorm"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Login 处理用户登录
func (l *LoginLogic) Login(in *user.LoginReq) (*user.LoginResp, error) {
	var userInfo *model.User
	err := l.svcCtx.Db.Where(&model.User{Username: in.Username}).Take(&userInfo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &user.LoginResp{
				StatusCode: STATUS_FAIL,
				StatusMsg:  STATUS_USER_NOTEXIST_MSG,
				UserID:     0,
			}, nil
		}
		return nil, err
	}

	// 对比明文密码与存储的密码哈希值是否相等
	if ok := utils.BcryptCheck(in.Password, userInfo.Password); !ok {
		return &user.LoginResp{
			StatusCode: STATUS_FAIL,
			StatusMsg:  STATUS_WRONG_PASSWORD_MSG,
			UserID:     0,
		}, nil
	}
	return &user.LoginResp{
		StatusCode: STATUS_SUCCESS,
		StatusMsg:  STATUS_SUCCESS_MSG,
		UserID:     userInfo.Id,
	}, nil
}
