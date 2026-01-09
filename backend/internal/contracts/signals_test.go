package contracts

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStockSignals_TotalScore(t *testing.T) {
	signals := &StockSignals{
		Code:      "005930",
		Momentum:  0.8,
		Technical: 0.6,
		Value:     0.4,
		Quality:   0.7,
		Flow:      0.5,
		Event:     0.3,
	}

	// Expected: 0.8*0.25 + 0.6*0.20 + 0.4*0.20 + 0.7*0.20 + 0.5*0.10 + 0.3*0.05
	expected := 0.8*0.25 + 0.6*0.20 + 0.4*0.20 + 0.7*0.20 + 0.5*0.10 + 0.3*0.05

	score := signals.TotalScore()
	epsilon := 0.0001
	if diff := score - expected; diff > epsilon || diff < -epsilon {
		t.Errorf("TotalScore() = %v, want %v (diff: %v)", score, expected, diff)
	}
}

func TestStockSignals_IsPositive(t *testing.T) {
	tests := []struct {
		name    string
		signals *StockSignals
		want    bool
	}{
		{
			name: "positive signals",
			signals: &StockSignals{
				Momentum:  0.8,
				Technical: 0.6,
				Value:     0.4,
				Quality:   0.5,
				Flow:      0.3,
				Event:     0.2,
			},
			want: true,
		},
		{
			name: "negative signals",
			signals: &StockSignals{
				Momentum:  -0.8,
				Technical: -0.6,
				Value:     -0.4,
				Quality:   -0.5,
				Flow:      -0.3,
				Event:     -0.2,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.signals.IsPositive(); got != tt.want {
				t.Errorf("IsPositive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignalSet_Get(t *testing.T) {
	signalSet := &SignalSet{
		Date: time.Now(),
		Signals: map[string]*StockSignals{
			"005930": {Code: "005930", Momentum: 0.8},
			"000660": {Code: "000660", Momentum: 0.6},
		},
	}

	// Existing stock
	signals, exists := signalSet.Get("005930")
	if !exists {
		t.Error("Expected to find signals for 005930")
	}
	if signals.Code != "005930" {
		t.Errorf("Got code %s, want 005930", signals.Code)
	}

	// Non-existing stock
	_, exists = signalSet.Get("999999")
	if exists {
		t.Error("Expected not to find signals for 999999")
	}
}

func TestSignalSet_Count(t *testing.T) {
	signalSet := &SignalSet{
		Date: time.Now(),
		Signals: map[string]*StockSignals{
			"005930": {Code: "005930"},
			"000660": {Code: "000660"},
			"035420": {Code: "035420"},
		},
	}

	if count := signalSet.Count(); count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

func TestStockSignals_JSON(t *testing.T) {
	original := &StockSignals{
		Code:      "005930",
		Momentum:  0.8,
		Technical: 0.6,
		Value:     0.4,
		Quality:   0.7,
		Flow:      0.5,
		Event:     0.3,
		Details: SignalDetails{
			Return1M:   0.15,
			Return3M:   0.25,
			VolumeRate: 1.5,
		},
		Events: []EventSignal{
			{
				Type:      "earnings",
				Score:     0.8,
				Source:    "DART",
				Timestamp: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			},
		},
		UpdatedAt: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded StockSignals
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if decoded.Code != original.Code {
		t.Errorf("Code mismatch: got %s, want %s", decoded.Code, original.Code)
	}
	if decoded.Momentum != original.Momentum {
		t.Errorf("Momentum mismatch: got %f, want %f", decoded.Momentum, original.Momentum)
	}
	if len(decoded.Events) != len(original.Events) {
		t.Errorf("Events count mismatch: got %d, want %d", len(decoded.Events), len(original.Events))
	}
}
