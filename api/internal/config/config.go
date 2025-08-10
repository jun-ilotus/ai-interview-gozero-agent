package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	OpenAI struct {
		ApiKey         string
		BaseURL        string
		Model          string
		EmbeddingModel string

		MaxTokens   int
		Temperature float32
		TopP        float32

		FrequencyPenalty float32
		PresencePenalty  float32
		Seed             *int
	}
	VectorDB      VectorDBConfig
	UniPDFLicense string
}

// VectorDBConfig 向量数据库配置
type VectorDBConfig struct {
	Host           string
	Port           int
	DBName         string
	User           string
	Password       string
	Table          string
	MaxConn        int
	EmbeddingModel string
}
