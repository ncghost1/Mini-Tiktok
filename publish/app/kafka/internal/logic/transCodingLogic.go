package logic

import (
	"Mini-Tiktok/publish/app/kafka/internal/svc"
	"Mini-Tiktok/publish/app/kafka/model"
	"context"
	"encoding/json"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/ncghost1/snowflake-go"
	"os"
	"os/exec"
	"strings"
	"time"
)

type TranscodingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTranscodingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TranscodingLogic {
	return &TranscodingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type MsgInfo struct {
	Title        string `json:"title"`
	OssObjectKey string `json:"ossObjectKey"`
}

// TransCoding 视频转码服务
func (l *TranscodingLogic) TransCoding(userid string, msg []byte) error {
	var msgInfo *MsgInfo
	err := json.Unmarshal(msg, &msgInfo)
	if err != nil {
		return err
	}
	// 1. 从 Oss 下载待处理的视频文件
	filePath, err := l.OssDownloadFile(msgInfo.OssObjectKey)
	if err != nil {
		return err
	}
	splits := strings.Split(filePath, "/")
	if len(splits) == 0 {
		return err
	}

	fileName := splits[len(splits)-1]
	coverName := strings.TrimSuffix(fileName, MP4_SUFFIX) + "_cover" + JPG_SUFFIX
	outputPath := OUTPUT_FILEPATH + fileName
	coverPath := OUTPUT_FILEPATH + coverName
	if !pathExist(OUTPUT_FILEPATH) {
		err = os.MkdirAll(OUTPUT_FILEPATH, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// 2. 调用 ffmpeg 视频转码与截取封面（请确保本地安装了 ffmpeg，并设置了环境变量）
	cmd := exec.Command("ffmpeg", "-i", filePath, "-preset", "fast", outputPath)
	cmd.Run()
	cmd = exec.Command("ffmpeg", "-i", outputPath, "-ss", "00:00:00", "-frames:v", "1", coverPath)
	cmd.Run()

	// 3. 将转码后视频与封面上传至 OSS
	outputVideo, err := os.Open(outputPath)
	outputCover, err := os.Open(coverPath)
	playUrl, err := l.OssUploadFile(outputVideo, l.svcCtx.Config.AliyunOss.VideoPath, fileName)
	if err != nil {
		return err
	}

	coverUrl, err := l.OssUploadFile(outputCover, l.svcCtx.Config.AliyunOss.CoverPath, coverName)
	if err != nil {
		return err
	}

	// 4.将视频信息写入 db 和 cache
	sf, err := snowflake.New(l.svcCtx.Config.WorkerId)
	videoId, err := sf.Generate()
	if err != nil {
		return err
	}

	createTime := time.Now().Unix()
	videoInfo := &model.Video{
		Id:         videoId,
		UserId:     userid,
		Title:      msgInfo.Title,
		PlayUrl:    playUrl,
		CoverUrl:   coverUrl,
		CreateTime: createTime,
	}
	err = l.svcCtx.Db.Model(&videoInfo).Create(&videoInfo).Error
	if err != nil {
		return err
	}

	videoJson, err := json.Marshal(&videoInfo)
	if err != nil {
		return err
	}

	conn := l.svcCtx.Redis.NewRedisConn()
	defer conn.Close()
	err = l.svcCtx.Redis.AddVideoInfoAndFeed(conn, videoId, videoJson, createTime, l.svcCtx.Config.CacheConfig.VIDEO_CACHE_TTL, l.svcCtx.Config.CacheConfig.FEED_MAX_CACHE_SIZE)
	if err != nil {
		return err
	}

	// 5. 最后将原视频（本地与OSS）删除
	outputVideo.Close()
	outputCover.Close()
	err = os.Remove(outputPath)
	if err != nil {
		return err
	}

	err = os.Remove(coverPath)
	if err != nil {
		return err
	}

	err = os.Remove(filePath)
	if err != nil {
		return err
	}
	err = l.OssDeleteFile(msgInfo.OssObjectKey)
	if err != nil {
		return err
	}

	return nil
}

// OssDownloadFile 从 OSS 下载文件至本地
func (l *TranscodingLogic) OssDownloadFile(objectKey string) (string, error) {
	bucket, err := l.svcCtx.Oss.Bucket(l.svcCtx.Config.AliyunOss.Bucket)
	if err != nil {
		return "", err
	}

	splits := strings.Split(objectKey, "/")
	if len(splits) == 0 {
		return "", err
	}

	fileName := splits[len(splits)-1]
	filePath := TEMP_FILEPATH + fileName + MP4_SUFFIX
	if !pathExist(TEMP_FILEPATH) {
		err = os.MkdirAll(TEMP_FILEPATH, os.ModePerm)
		if err != nil {
			return "", err
		}
	}
	err = bucket.GetObjectToFile(objectKey, filePath)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func pathExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// OssUploadFile 将文件上传到OSS
// 返回值 string 为上传后的文件 url
func (l *TranscodingLogic) OssUploadFile(file *os.File, path, fileName string) (string, error) {
	bucket, err := l.svcCtx.Oss.Bucket(l.svcCtx.Config.AliyunOss.Bucket)
	if err != nil {
		return "", err
	}
	ossObjKey := path + fileName
	err = bucket.PutObject(ossObjKey, file, oss.Checkpoint(true, ""))

	if err != nil {
		return "", err
	}
	url := l.svcCtx.Config.AliyunOss.UrlPrefix + ossObjKey
	return url, nil
}

func (l *TranscodingLogic) OssDeleteFile(objectKey string) error {
	bucket, err := l.svcCtx.Oss.Bucket(l.svcCtx.Config.AliyunOss.Bucket)
	if err != nil {
		return err
	}
	err = bucket.DeleteObject(objectKey)
	if err != nil {
		return err
	}
	return nil
}
