package logic

import (
	"ai-gozero-agent/api/internal/utils"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"
	openai "github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
)

type ChatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// SSE流式接口
func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChatLogic) Chat(req *types.InterViewAPPChatReq) (<-chan *types.ChatResponse, error) {
	ch := make(chan *types.ChatResponse)

	go func() {
		defer close(ch)

		// 1.保存用户消息到向量数据库
		if err := l.svcCtx.VectorStore.SaveMessage(req.ChatId, openai.ChatMessageRoleUser, req.Message); err != nil {
			l.Logger.Errorf("save message failed: %v", err)
			// 不返回，继续处理会话
		}

		// 知识检索（RAG核心）
		knowledge, err := l.svcCtx.VectorStore.RetrieveKnowledge(req.Message, 3)
		if err != nil {
			l.Logger.Errorf("retrieve knowledge failed: %v", err)
			knowledge = []types.KnowledgeChunk{}
		}

		// 2.获取会话历史
		message, err := l.getSessionHistory(req.ChatId, knowledge)
		if err != nil {
			l.Logger.Errorf("get session history failed: %v", err)
			ch <- &types.ChatResponse{
				Content: "get session history failed",
				IsLast:  true,
			}
		}

		// 3.创建OpenAI请求
		request := openai.ChatCompletionRequest{
			Model:            l.svcCtx.Config.OpenAI.Model,
			Messages:         message, // 使用会话历史
			Stream:           true,
			MaxTokens:        l.svcCtx.Config.OpenAI.MaxTokens,
			Temperature:      l.svcCtx.Config.OpenAI.Temperature,
			TopP:             l.svcCtx.Config.OpenAI.TopP,
			FrequencyPenalty: l.svcCtx.Config.OpenAI.FrequencyPenalty,
			PresencePenalty:  l.svcCtx.Config.OpenAI.PresencePenalty,
			Seed:             l.svcCtx.Config.OpenAI.Seed,
		}

		// 4.创建流式响应
		stream, err := l.svcCtx.OpenAIClient.CreateChatCompletionStream(l.ctx, request)
		if err != nil {
			l.Logger.Error(err)
			return
		}
		defer stream.Close()

		// 5.处理流式响应
		var fullResponse strings.Builder
		for {
			select {
			case <-l.ctx.Done():
				return
			default:
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) { // 流结束
					// 流结束后保存会话
					if fullResponse.String() != "" {
						if saveErr := l.svcCtx.VectorStore.SaveMessage(
							req.ChatId, openai.ChatMessageRoleAssistant, fullResponse.String()); saveErr != nil {
							l.Logger.Errorf("save message failed: %v", saveErr)
						}
					}
					// 发送结束标记
					ch <- &types.ChatResponse{IsLast: true}
					return
				}
				if err != nil {
					l.Logger.Error(err)
					return
				}

				if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
					content := response.Choices[0].Delta.Content
					fullResponse.WriteString(content) // 收集完整响应
					ch <- &types.ChatResponse{
						Content: content,
						IsLast:  false,
					}
				}
			}
		}
	}()

	return ch, nil
}

// 获取会话历史
func (l *ChatLogic) getSessionHistory(chatId string, knowledge []types.KnowledgeChunk) ([]openai.ChatCompletionMessage, error) {
	// 获得最近的10条消息（约5轮对话）
	vectorMessage, err := l.svcCtx.VectorStore.GetMessages(chatId, 10)
	if err != nil {
		return nil, err
	}

	// 构建系统消息 - 注入知识
	systemMessage := "你是一个专业的goGo语言面试言，负责评估候选人的Go语言能力。请提出有深度的问题并评估回答。"
	if len(knowledge) > 0 {
		systemMessage += "\n\n相关背景知识："
		for i, k := range knowledge {
			// 限制知识片段长度
			truncateContent := utils.TruncateText(k.Content, 500)
			systemMessage += fmt.Sprintf("\n[知识片段%d] %s：%s", i+1, k.Title, truncateContent)
		}
	}
	fmt.Println("检索的数据", systemMessage)

	// 转换为OpenAI消息格式
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemMessage,
		},
	}

	// 添加历史消息
	for _, msg := range vectorMessage {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages, nil
}
