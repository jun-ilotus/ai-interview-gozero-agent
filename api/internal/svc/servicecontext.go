package svc

import (
	"ai-gozero-agent/api/internal/config"
	"ai-gozero-agent/api/internal/types"
	"github.com/sashabaranov/go-openai"
)

// ServiceContext 是所有连接共用的，服务启动时初始化一次，整个生命周期内共享
type ServiceContext struct {
	Config       config.Config
	OpenAIClient *openai.Client
	SessionStore types.SessionStore // 会话存储
}

func NewServiceContext(c config.Config) *ServiceContext {
	conf := openai.DefaultConfig(c.OpenAI.ApiKey)
	conf.BaseURL = c.OpenAI.BaseURL

	return &ServiceContext{
		Config:       c,
		OpenAIClient: openai.NewClientWithConfig(conf),
		SessionStore: NewMemorySessionStore(), // 内存会话存储
	}
}
