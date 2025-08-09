package types

import "github.com/sashabaranov/go-openai"

type ChatSession struct {
	Messages []openai.ChatCompletionMessage `json:"message"` // 存储对话历史（系统消息+用户消息+AI回复）
}

type SessionStore interface {
	GetSession(chatId string) (*ChatSession, error)        // 根据 chatId 获取或创建会话
	SaveSession(chatId string, session *ChatSession) error // 保存更新后的会话
}
