package strategyconfig

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Load reads YAML file and returns Config with raw bytes
// SSOT 핵심: KnownFields(true)로 오타/미사용 필드 즉시 실패
func Load(path string) (*Config, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // 알 수 없는 필드 발견 시 에러 반환
	if err := dec.Decode(&cfg); err != nil {
		return nil, nil, err
	}

	if err := Validate(&cfg); err != nil {
		return nil, data, err
	}

	return &cfg, data, nil
}

// Hash generates SHA256 hash from Config (canonical JSON)
// 주의: map 대신 struct 사용으로 해시 재현성 보장
func Hash(cfg *Config) (string, error) {
	// Struct → JSON (결정적 순서)
	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(sum[:]), nil
}

// NewDecisionSnapshot creates a snapshot for audit
func NewDecisionSnapshot(cfg *Config, yamlData []byte, gitCommit, dataSnapshotID string) (*DecisionSnapshot, error) {
	hash, err := Hash(cfg)
	if err != nil {
		return nil, err
	}

	return &DecisionSnapshot{
		ConfigHash:     hash,
		ConfigYAML:     string(yamlData),
		StrategyID:     cfg.Meta.StrategyID,
		GitCommit:      gitCommit,
		DataSnapshotID: dataSnapshotID,
		CreatedAt:      time.Now(),
	}, nil
}
