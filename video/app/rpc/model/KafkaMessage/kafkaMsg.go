package KafkaMessage

type MsgInfo struct {
	Op      string `json:"op"`     // 只实现支持 insert 和 delete
	Model   string `json:"model"`  // 标明要使用的 Model 名称（小写），暂时只实现支持 favorite（只设计点赞使用异步写入）
	Columns string `json:"column"` // 列值，输入按照对应 Model 中的成员顺序，以逗号分割（如 favorite 列值: 1,2,1679147184)
}
