package svc

import (
	"ai-gozero-agent/api/internal/config"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"github.com/unidoc/unipdf/v3/common/license"
	"log"
)

// ServiceContext 是所有连接共用的，服务启动时初始化一次，整个生命周期内共享
type ServiceContext struct {
	Config       config.Config
	OpenAIClient *openai.Client
	//SessionStore types.SessionStore // 会话存储
	VectorStore *VectorStore
	PdfClient   *PdfClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 创建OpenAI客户端
	//openaiConf := openai.DefaultConfig(c.OpenAI.ApiKey)
	//openaiConf.BaseURL = c.OpenAI.BaseURL
	//openAIClient := openai.NewClientWithConfig(openaiConf)

	// ollama
	openaiConf := openai.DefaultConfig("")
	openaiConf.BaseURL = c.OpenAI.BaseURL
	openAIClient := openai.NewClientWithConfig(openaiConf)

	// 初始化向量存储
	vectorStore, err := NewVectorStore(c.VectorDB, openAIClient)
	if err != nil {
		log.Fatalf("NewVectorStore err: %v", err)
	}

	// 测试数据库连接
	if err := vectorStore.TestConnection(); err != nil {
		log.Fatalf("TestConnection err: %v", err)
	} else {
		log.Println("TestConnection success")
	}

	err = license.SetMeteredKey(c.UniPDFLicense)
	if err != nil {
		fmt.Printf("SetMeteredKey err: %v", err)
	} // 如果没有授权，unipdf会添加水印

	return &ServiceContext{
		Config:       c,
		OpenAIClient: openAIClient,
		//SessionStore: NewMemorySessionStore(), // 内存会话存储
		VectorStore: vectorStore,
		PdfClient:   NewPdfClient(c.MCP.Endpoint),
	}
}
