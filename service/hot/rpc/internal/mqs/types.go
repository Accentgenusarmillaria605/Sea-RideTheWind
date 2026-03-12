package mqs

// ArticleHotEvent 热点事件消息（Kafka message value）
// 业务侧只需发送 article_id + type，权重由热点系统根据配置自行决定
type ArticleHotEvent struct {
	ArticleID string `json:"article_id"`
	Type      string `json:"type"` // "like" | "comment" | "coin" | "share"
}
