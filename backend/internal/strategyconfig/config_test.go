package strategyconfig

import (
	"math"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// 테스트용 YAML 경로
	path := "../../../config/strategy/korea_equity_v13.yaml"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("config file not found")
	}

	cfg, yamlData, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 기본 검증
	if cfg.Meta.StrategyID != "korea_equity_v13" {
		t.Errorf("expected strategy_id=korea_equity_v13, got %s", cfg.Meta.StrategyID)
	}

	// ADTV20 수정 확인
	if cfg.Universe.Filters.ADTV20MinKRW != 2_000_000_000 {
		t.Errorf("expected ADTV20=2_000_000_000, got %d", cfg.Universe.Filters.ADTV20MinKRW)
	}

	// 해시 생성
	hash, err := Hash(cfg)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(hash))
	}

	// 동일 설정 → 동일 해시
	hash2, _ := Hash(cfg)
	if hash != hash2 {
		t.Error("hash not deterministic")
	}

	t.Logf("config hash: %s", hash)
	t.Logf("yaml size: %d bytes", len(yamlData))
}

func TestValidateWeights(t *testing.T) {
	// 가중치 합 검증
	cfg := &Config{}
	cfg.Ranking.WeightsPct = RankingWeights{
		Momentum:  25,
		Flow:      20,
		Technical: 15,
		Event:     15,
		Value:     15,
		Quality:   10,
	}

	if cfg.Ranking.WeightsPct.Sum() != 100 {
		t.Errorf("expected 100, got %d", cfg.Ranking.WeightsPct.Sum())
	}
}

func TestValidateTiers(t *testing.T) {
	w := Weighting{
		Method: "TIERED",
		Tiers: []Tier{
			{Count: 5, WeightEachPct: 0.05},
			{Count: 10, WeightEachPct: 0.045},
			{Count: 5, WeightEachPct: 0.04},
		},
	}

	// Count 합 = 20
	if w.TotalCount() != 20 {
		t.Errorf("expected count=20, got %d", w.TotalCount())
	}

	// Weight 합 = 0.25 + 0.45 + 0.20 = 0.90 (현금 0.10 별도)
	// 0.045 같은 소수는 이진 부동소수 표현에서 오차가 생길 수 있어 1e-6 사용
	expectedWeight := 0.90
	if math.Abs(w.TotalWeightPct()-expectedWeight) > 1e-6 {
		t.Errorf("expected weight=%.2f, got %.4f", expectedWeight, w.TotalWeightPct())
	}
}

func TestWarn(t *testing.T) {
	cfg := &Config{}
	cfg.Universe.Filters.ADTV20MinKRW = 500_000_000 // 5억 (10억 미만)
	cfg.Execution.SlippageModel.Segments = []SlippageSegment{
		{ADTV20MinKRW: 2_000_000_000, SlippagePct: 0.002}, // 0.2% (낙관적)
	}

	warnings := Warn(cfg)
	if len(warnings) < 2 {
		t.Errorf("expected at least 2 warnings, got %d", len(warnings))
	}
}

func TestDecisionSnapshot(t *testing.T) {
	cfg := &Config{
		Meta: Meta{
			StrategyID: "test_strategy",
			Version:    "1.0.0",
		},
	}
	yamlData := []byte("test yaml content")

	snapshot, err := NewDecisionSnapshot(cfg, yamlData, "abc123", "data_20240115")
	if err != nil {
		t.Fatalf("NewDecisionSnapshot failed: %v", err)
	}

	if snapshot.StrategyID != "test_strategy" {
		t.Errorf("expected strategy_id=test_strategy, got %s", snapshot.StrategyID)
	}
	if snapshot.GitCommit != "abc123" {
		t.Errorf("expected git_commit=abc123, got %s", snapshot.GitCommit)
	}
	if len(snapshot.ConfigHash) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(snapshot.ConfigHash))
	}
}

func TestValidateHHMM(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"09:00", true},
		{"15:30", true},
		{"00:00", true},
		{"23:59", true},
		{"9:00", false},
		{"09:0", false},
		{"25:00", false},
		{"09:60", false},
		{"invalid", false},
	}

	for _, tc := range tests {
		err := validateHHMM(tc.input)
		if tc.valid && err != nil {
			t.Errorf("validateHHMM(%s) expected valid, got error: %v", tc.input, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("validateHHMM(%s) expected error, got nil", tc.input)
		}
	}
}

func TestValidateWeightsSum(t *testing.T) {
	tests := []struct {
		weights []float64
		target  float64
		valid   bool
	}{
		{[]float64{0.4, 0.35, 0.25}, 1.0, true},
		{[]float64{0.5, 0.5}, 1.0, true},
		{[]float64{0.3, 0.3, 0.3}, 1.0, false}, // 0.9
		{[]float64{}, 1.0, false},
	}

	for _, tc := range tests {
		err := validateWeightsSum(tc.weights, tc.target, 1e-6)
		if tc.valid && err != nil {
			t.Errorf("validateWeightsSum(%v) expected valid, got error: %v", tc.weights, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("validateWeightsSum(%v) expected error, got nil", tc.weights)
		}
	}
}
