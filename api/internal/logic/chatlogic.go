package logic

import (
	"context"
	"errors"
	"io"

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

		messages := []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "你是一个专业的Go语言面试言，负责评估候选人的Go语言能力。请提出有深度的问题并评估回答。",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.Message,
			},
		}

		// 创建OpenAI请求
		request := openai.ChatCompletionRequest{
			Model:       l.svcCtx.Config.OpenAI.Model,
			Messages:    messages,
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

		for {
			select {
			case <-l.ctx.Done():
				return
			default:
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
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
