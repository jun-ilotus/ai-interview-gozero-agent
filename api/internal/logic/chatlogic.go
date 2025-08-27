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
		stateManager := NewStateManager(l.svcCtx)
		// 获取当前状态
		currentState, err := stateManager.GetOrInitState(req.ChatId)
		if err != nil {
			l.Logger.Errorf("get current state failed: %v", err)
			currentState = types.StateStart
		}

		// 知识检索（RAG核心）
		knowledge, err := l.svcCtx.VectorStore.RetrieveKnowledge(req.Message, 3)
		if err != nil {
			l.Logger.Errorf("retrieve knowledge failed: %v", err)
			knowledge = []types.KnowledgeChunk{}
		}

		// 2.获取会话历史，构建带状态系统消息
		message, err := l.buildMessageWithState(req.ChatId, currentState, knowledge)
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
			ch <- &types.ChatResponse{Content: "系统错误：无法连接AI服务", IsLast: true}
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
				if errors.Is(err, io.EOF) { // 流结束后处理状态更新
					finalResponse := fullResponse.String()
					// 流结束后保存会话
					if finalResponse != "" {
						// 保存AI回复
						if saveErr := l.svcCtx.VectorStore.SaveMessage(
							req.ChatId, openai.ChatMessageRoleAssistant, fullResponse.String()); saveErr != nil {
							l.Logger.Errorf("save message failed: %v", saveErr)
						}

						// 更新状态
						newState, err := stateManager.EvaluateAndUpdateState(req.ChatId, finalResponse)
						if err != nil {
							l.Logger.Errorf("evaluate and update state failed: %v", err)
						} else {
							l.Logger.Infof("evaluate and update state: %v", newState)
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

// buildMessageWithState 构建带状态的消息
func (l *ChatLogic) buildMessageWithState(chatId, currentState string, knowledge []types.KnowledgeChunk) ([]openai.ChatCompletionMessage, error) {
	// 构建状态待定的系统消息
	systemMessage := "你是一个专业的Go语言面试官，负责评估候选人的Go语言能力"
	systemMessage += "\n\n当前状态：" + currentState

	switch currentState {
	case types.StateStart:
		systemMessage += "\n目标：欢迎候选人并开始面试流程"
	case types.StateQuestion:
		systemMessage += "\n目标：提出有深度的问题考察Go语言核心概念"
	case types.StateFollowUp:
		systemMessage += "\n目标：基于候选人的回答进行追问，深入考察理解深度"
	case types.StateEvaluate:
		systemMessage += "\n目标：全面评估候选人的技术能力"
	case types.StateEnd:
		systemMessage += "\n目标：结束面试并提供反馈"
	}

	// 注入知识
	if len(knowledge) > 0 {
		systemMessage += "\n\n相关背景知识："
		for i, k := range knowledge {
			// 限制知识片段长度
			truncateContent := utils.TruncateText(k.Content, 500)
			systemMessage += fmt.Sprintf("\n[知识片段%d] %s：%s", i+1, k.Title, truncateContent)
		}
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemMessage,
		},
	}

	history, err := l.svcCtx.VectorStore.GetMessages(chatId, 10)
	if err != nil {
		return nil, err
	}
	for _, msg := range history {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return messages, nil
}
