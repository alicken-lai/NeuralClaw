package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

var CfgFile string

var GlobalConfig Config

type Config struct {
	Log       LogConfig       `mapstructure:"log"`
	Retention RetentionPolicy `mapstructure:"retention"`
	Web       WebConfig       `mapstructure:"web"`
	Memory    MemoryConfig    `mapstructure:"memory"`
	Ingest    IngestConfig    `mapstructure:"ingest"`
	Agent     AgentConfig     `mapstructure:"agent"`
}

type AgentConfig struct {
	Provider string `mapstructure:"provider"`
	BaseURL  string `mapstructure:"base_url"`
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
}

type IngestConfig struct {
	OCR OCRConfig `mapstructure:"ocr"`
}

type OCRConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
}

type MemoryConfig struct {
	DBPath    string          `mapstructure:"db_path"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	Retrieval RetrievalConfig `mapstructure:"retrieval"`
}

type EmbeddingConfig struct {
	Provider   string `mapstructure:"provider"` // jina, openai, ollama
	APIKey     string `mapstructure:"api_key"`
	BaseURL    string `mapstructure:"base_url"`
	Model      string `mapstructure:"model"`
	Dimensions int    `mapstructure:"dimensions"`
}

type RetrievalConfig struct {
	VectorWeight          float64 `mapstructure:"vector_weight"`
	BM25Weight            float64 `mapstructure:"bm25_weight"`
	MinScore              float64 `mapstructure:"min_score"`
	CandidatePoolSize     int     `mapstructure:"candidate_pool_size"`
	RerankProvider        string  `mapstructure:"rerank_provider"` // jina, siliconflow, pinecone, none
	RerankAPIKey          string  `mapstructure:"rerank_api_key"`
	RerankEndpoint        string  `mapstructure:"rerank_endpoint"`
	RerankModel           string  `mapstructure:"rerank_model"`
	RecencyHalfLifeDays   float64 `mapstructure:"recency_half_life_days"`
	RecencyWeight         float64 `mapstructure:"recency_weight"`
	LengthNormAnchor      float64 `mapstructure:"length_norm_anchor"`
	TimeDecayHalfLifeDays float64 `mapstructure:"time_decay_half_life_days"`
	HardMinScore          float64 `mapstructure:"hard_min_score"`
}

type WebConfig struct {
	Addr      string `mapstructure:"addr"`
	AuthToken string `mapstructure:"auth_token"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func Load() {
	if CfgFile != "" {
		viper.SetConfigFile(CfgFile)
	} else {
		viper.AddConfigPath("configs")
		viper.SetConfigName("config.example")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("ZCLAW")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Warning: failed to read config file (%s), using defaults\n", err)
	}

	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		fmt.Printf("Error: failed to unmarshal config: %v\n", err)
	}
}
