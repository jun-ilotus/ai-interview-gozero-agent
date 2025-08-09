package svc

import (
	"ai-gozero-agent/api/internal/types"
	"github.com/sashabaranov/go-openai"
	"sync"
	"time"
)

type MemorySessionStore struct {
	sessions     map[string]*types.ChatSession // 存储 chatId 到会话的映射
	lastAccessed map[string]time.Time          // 记录最后访问时间，用于会话清理
	lock         sync.RWMutex                  // 读写锁保证并发安全
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions:     make(map[string]*types.ChatSession),
		lastAccessed: make(map[string]time.Time),
	}
}

func (m *MemorySessionStore) GetSession(chatId string) (*types.ChatSession, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	session, ok := m.sessions[chatId]
	if !ok {
		// 创建新会话
		return &types.ChatSession{
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你是一个专业的Go语言面试言，负责评估候选人的Go语言能力。请提出有深度的问题并评估回答。",
				},
			},
		}, nil
	}

	// 更新访问时间
	m.lastAccessed[chatId] = time.Now()
	return session, nil
}

func (m *MemorySessionStore) SaveSession(chatId string, session *types.ChatSession) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 上下文截断（保留系统消息 和 最近5轮对话）
	if len(session.Messages) > 10 {
		newMessages := []openai.ChatCompletionMessage{session.Messages[0]} // 0是系统消息
		start := len(session.Messages) - 5
		if start < 1 {
			start = 1
		}
		newMessages = append(newMessages, session.Messages[start:]...)
		session.Messages = newMessages
	}

	m.sessions[chatId] = session
	m.lastAccessed[chatId] = time.Now()
	return nil
}

// ClearUpExpiredSession 清理过期会话（可定期调用）
func (m *MemorySessionStore) ClearUpExpiredSession(maxAge time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()

	now := time.Now()
	for chatId, lastAccess := range m.lastAccessed {
		if now.Sub(lastAccess) > maxAge {
			delete(m.sessions, chatId)
			delete(m.lastAccessed, chatId)
		}
	}
}
