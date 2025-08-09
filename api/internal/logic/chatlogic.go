package logic

import (
	"context"
	"errors"
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

		// 获取或创建会话
		session, err := l.svcCtx.SessionStore.GetSession(req.ChatId)
		if err != nil {
			l.Logger.Errorf("get session failed: %v", err)
			return
		}

		// 添加用户消息到会话历史
		userMessage := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Message,
		}
		session.Messages = append(session.Messages, userMessage)

		// 创建OpenAI请求 (使用完整上下文)
		request := openai.ChatCompletionRequest{
			Model:       l.svcCtx.Config.OpenAI.Model,
			Messages:    session.Messages, // 使用会话历史
			Stream:      true,
			MaxTokens:   l.svcCtx.Config.OpenAI.MaxTokens,
			Temperature: l.svcCtx.Config.OpenAI.Temperature,
		}

		// 创建流式响应
		stream, err := l.svcCtx.OpenAIClient.CreateChatCompletionStream(l.ctx, request)
		if err != nil {
			l.Logger.Error(err)
			return
		}
		defer stream.Close()

		// 收集完整响应内容
		var fullResponse strings.Builder

		for {
			select {
			case <-l.ctx.Done():
				return
			default:
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					// 流结束后保存会话
					assistantMessage := openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleAssistant,
						Content: fullResponse.String(),
					}
					session.Messages = append(session.Messages, assistantMessage)

					if err := l.svcCtx.SessionStore.SaveSession(req.ChatId, session); err != nil {
						l.Logger.Errorf("save session failed: %v", err)
					}

					// 发送结束标记
					ch <- &types.ChatResponse{IsLast: true}
					return
				}
				if err != nil {
					l.Logger.Error(err)
					return
				}

				if len(response.Choices) > 0 {
					content := response.Choices[0].Delta.Content
					if content != "" {
						fullResponse.WriteString(content) // 收集完整响应
						ch <- &types.ChatResponse{
							Content: content,
							IsLast:  false,
						}
					}
				}
			}
		}
	}()

	return ch, nil
}
