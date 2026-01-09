package contracts

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDataQualitySnapshot_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		snapshot DataQualitySnapshot
		want     bool
	}{
		{
			name: "valid snapshot",
			snapshot: DataQualitySnapshot{
				Date:         time.Now(),
				TotalStocks:  100,
				ValidStocks:  90,
				QualityScore: 0.9,
				Coverage:     map[string]float64{"price": 0.95, "volume": 0.90},
			},
			want: true,
		},
		{
			name: "low quality score",
			snapshot: DataQualitySnapshot{
				Date:         time.Now(),
				TotalStocks:  100,
				ValidStocks:  50,
				QualityScore: 0.5,
			},
			want: false,
		},
		{
			name: "no valid stocks",
			snapshot: DataQualitySnapshot{
				Date:         time.Now(),
				TotalStocks:  100,
				ValidStocks:  0,
				QualityScore: 0.8,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.snapshot.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDataQualitySnapshot_CoverageRate(t *testing.T) {
	snapshot := DataQualitySnapshot{
		Date:        time.Now(),
		TotalStocks: 100,
		ValidStocks: 90,
		Coverage: map[string]float64{
			"price":  0.95,
			"volume": 0.90,
			"market": 0.85,
		},
	}

	expected := (0.95 + 0.90 + 0.85) / 3
	if rate := snapshot.CoverageRate(); rate != expected {
		t.Errorf("CoverageRate() = %v, want %v", rate, expected)
	}
}

func TestDataQualitySnapshot_JSON(t *testing.T) {
	original := DataQualitySnapshot{
		Date:         time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		TotalStocks:  100,
		ValidStocks:  90,
		QualityScore: 0.9,
		Coverage: map[string]float64{
			"price": 0.95,
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded DataQualitySnapshot
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if decoded.TotalStocks != original.TotalStocks {
		t.Errorf("TotalStocks mismatch: got %d, want %d", decoded.TotalStocks, original.TotalStocks)
	}
	if decoded.ValidStocks != original.ValidStocks {
		t.Errorf("ValidStocks mismatch: got %d, want %d", decoded.ValidStocks, original.ValidStocks)
	}
	if decoded.QualityScore != original.QualityScore {
		t.Errorf("QualityScore mismatch: got %f, want %f", decoded.QualityScore, original.QualityScore)
	}
}
