package svc

import (
	"ai-gozero-agent/api/internal/config"
	"ai-gozero-agent/api/internal/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sashabaranov/go-openai"
	"time"
)

type VectorStore struct {
	Pool           *pgxpool.Pool  // 数据库连接池
	OpenIClient    *openai.Client // OpenAI客户端
	EmbeddingModel string         // 向量模型名称
}

func NewVectorStore(cfg config.VectorDBConfig, openAIClient *openai.Client) (*VectorStore, error) {
	// 构建连接字符串
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	// 解析配置
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConns = int32(cfg.MaxConn) // 设置最大连接数

	// 创建连接池
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	return &VectorStore{
		Pool:           pool,
		OpenIClient:    openAIClient,
		EmbeddingModel: cfg.EmbeddingModel,
	}, nil
}

// SaveMessage 保存消息到向量数据库
func (vs *VectorStore) SaveMessage(chatId, role, content string) error {
	// 生成文本向量
	embedding, err := vs.generateEmbedding(content)
	if err != nil {
		return fmt.Errorf("generateEmbedding: %w", err)
	}

	// 将向量转换为JSON格式
	embeddingJson, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("marshal embedding: %w", err)
	}

	sql := `INSERT INTO vector_store (chat_id, role, content, embedding) VALUES ($1, $2, $3, $4)`
	_, err = vs.Pool.Exec(context.Background(), sql, chatId, role, content, embeddingJson)

	return err
}

// 获取会话历史消息
func (vs *VectorStore) GetMessages(chatId string, limit int) ([]types.VectorMessage, error) {
	// 查询数据库
	sql := `SELECT role, content FROM vector_store WHERE chat_id = $1 ORDER BY created_at DESC LIMIT $2`
	rows, err := vs.Pool.Query(context.Background(), sql, chatId, limit)
	if err != nil {
		return nil, fmt.Errorf("DB select GetMessage: %w", err)
	}
	defer rows.Close()

	var messages []types.VectorMessage
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, fmt.Errorf("DB select row GetMessage: %w", err)
		}
		messages = append(messages, types.VectorMessage{
			Role:    role,
			Content: content,
		})
	}

	// 反转消息顺序（最新消息在最后）
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

// 生成向量文本
func (vs *VectorStore) generateEmbedding(text string) ([]float32, error) {
	if text == "" {
		return make([]float32, 1536), nil
	}

	// 调用OpenAI Embedding API
	resp, err := vs.OpenIClient.CreateEmbeddings(context.Background(),
		openai.EmbeddingRequest{
			Input: []string{text},
			Model: openai.EmbeddingModel(vs.EmbeddingModel),
		})
	if err != nil {
		return nil, fmt.Errorf("create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, errors.New("未返回嵌入数据")
	}

	return resp.Data[0].Embedding, nil
}

// TestConnection 测试数据库连接
func (vs *VectorStore) TestConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return vs.Pool.Ping(ctx)
}
