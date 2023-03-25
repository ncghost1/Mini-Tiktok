package logic

import (
	"Mini-Tiktok/video/app/kafka/internal/svc"
	"Mini-Tiktok/video/app/rpc/model"
	"context"
	"encoding/json"
	"errors"
)

type WriteDbLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewWriteDbLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteDbLogic {
	return &WriteDbLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type MsgInfo struct {
	Op      string `json:"op"`     // 只实现支持 insert 和 delete
	Model   string `json:"model"`  // 标明要使用的 Model 名称（小写），暂时只实现支持 favorite（只设计点赞使用异步写入）
	Columns string `json:"column"` // 列值，输入按照对应 Model 中的成员顺序，以逗号分割（如 favorite 列值: 1,2,1679147184)
}

var models map[string]interface{}

func InitModels() {
	models = make(map[string]interface{})
	models["favorite"] = model.Favorite{}
}

// WriteDb 解析消息并执行写入数据库操作
func (l *WriteDbLogic) WriteDb(msg []byte) error {
	var msgInfo *MsgInfo
	err := json.Unmarshal(msg, &msgInfo)
	if err != nil {
		return err
	}

	conn := l.svcCtx.Redis.NewRedisConn()
	defer conn.Close()

	if m, ok := models[msgInfo.Model]; ok {
		switch msgInfo.Op {
		case OP_INSERT:
			switch msgInfo.Model {
			case MODEL_FAVORITE:
				if _, ok := m.(model.Favorite); ok {
					favorite, err := ParseFavoriteModel([]byte(msgInfo.Columns))
					if err != nil {
						return err
					}

					err = l.svcCtx.Db.Create(&favorite).Error
					if err != nil {

						// 如果发生错误，还需要查找点赞数据是否存在，决定是否将视频 id 从用户最近取消点赞视频集合中删除
						// （Gorm 没预设插入相同主键错误信息来让我们判断，只能再查一次 DB 了）
						cnt := int64(0)
						err2 := l.svcCtx.Db.Model(&model.Favorite{}).Count(&cnt).Error
						if err2 != nil {
							return err2
						}

						// 记录不存在
						if cnt == 0 {
							err2 = l.svcCtx.Redis.DelFavorite(conn, favorite.VideoId, favorite.UserId) // 回滚缓存
							if err2 != nil {
								return err2
							}
						}

						return err
					}

				} else {
					return errors.New(MODEL_UNKNOWN_ERROR)
				}
			}
		case OP_DELETE:
			switch msgInfo.Model {
			case MODEL_FAVORITE:
				if _, ok := m.(model.Favorite); ok {
					favorite, err := ParseFavoriteModel([]byte(msgInfo.Columns))
					if err != nil {
						return err
					}

					err = l.svcCtx.Db.Delete(&model.Favorite{}, &model.Favorite{
						UserId:  favorite.UserId,
						VideoId: favorite.VideoId,
					}).Error
					if err != nil {
						err = l.svcCtx.Db.Delete(&model.Favorite{}, &model.Favorite{
							UserId:  favorite.UserId,
							VideoId: favorite.VideoId,
						}).Error

						// 消费失败也需要尝试将视频 id 从用户最近取消点赞视频集合中删除，删除后客户端重新获取点赞时查询数据库，即可做到回滚效果
						err2 := l.svcCtx.Redis.RemDelCacheMember(conn, favorite.VideoId, favorite.UserId)
						if err2 != nil {
							return err2
						}
						return err
					}

					err = l.svcCtx.Redis.RemDelCacheMember(conn, favorite.VideoId, favorite.UserId)
					if err != nil {
						return err
					}

				} else {
					return errors.New(MODEL_UNKNOWN_ERROR)
				}
			}
		default:
			return errors.New(OP_UNKNOWN_ERROR)
		}
	} else {
		return errors.New(MODEL_UNKNOWN_ERROR)
	}

	return nil
}

func ParseFavoriteModel(columns []byte) (*model.Favorite, error) {
	var favor *model.Favorite
	err := json.Unmarshal(columns, &favor)
	if err != nil {
		return nil, err
	}

	return favor, nil
}
