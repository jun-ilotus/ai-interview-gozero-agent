package types

import "github.com/sashabaranov/go-openai"

type ChatSession struct {
	Messages []openai.ChatCompletionMessage `json:"message"` // 存储对话历史（系统消息+用户消息+AI回复）
}

type VectorMessage struct {
	Role    string `json:"role"`    // 消息角色
	Content string `json:"content"` // 消息内容
}

type KnowledgeChunk struct {
	ID      int64  `json:"id"`      // 知识块ID
	Title   string `json:"title"`   // 知识标题
	Content string `json:"content"` // 知识内容
}

type SessionStore interface {
	GetSession(chatId string) ([]openai.ChatCompletionMessage, error) // 获取消息历史
	SaveSession(chatId string, role, content string) error            // 保存单条消息

	SaveKnowledge(title, content string) error                          // 保存知识库
	RetrieveKnowledge(query string, topK int) ([]KnowledgeChunk, error) // 检索知识库
}

//type SessionStore interface {
//	GetSession(chatId string) (*ChatSession, error)        // 根据 chatId 获取或创建会话
//	SaveSession(chatId string, session *ChatSession) error // 保存更新后的会话
//}
