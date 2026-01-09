package naver

import (
	"testing"
	"time"
)

func TestParsePriceJSON(t *testing.T) {
	tests := []struct {
		name    string
		rawData [][]interface{}
		want    int // Expected number of prices
		wantErr bool
	}{
		{
			name: "valid data with header",
			rawData: [][]interface{}{
				{"날짜", "시가", "고가", "저가", "종가", "거래량"}, // Header
				{"20240115", 72300.0, 73000.0, 72000.0, 72500.0, 1000000.0},
				{"20240116", 72500.0, 73500.0, 72300.0, 73000.0, 1200000.0},
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "valid data with string numbers",
			rawData: [][]interface{}{
				{"날짜", "시가", "고가", "저가", "종가", "거래량"},
				{"20240115", "72300", "73000", "72000", "72500", "1000000"},
			},
			want:    1,
			wantErr: false,
		},
		{
			name:    "empty data",
			rawData: [][]interface{}{},
			want:    0,
			wantErr: false,
		},
		{
			name: "data with insufficient columns",
			rawData: [][]interface{}{
				{"날짜", "시가"},
				{"20240115", 72300.0, 73000.0},
			},
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{}
			got, err := c.parsePriceJSON(tt.rawData)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePriceJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("parsePriceJSON() got %d prices, want %d", len(got), tt.want)
			}

			// Verify data structure
			for _, price := range got {
				if price.TradeDate.IsZero() {
					t.Error("parsePriceJSON() TradeDate is zero")
				}
				if price.ClosePrice <= 0 {
					t.Error("parsePriceJSON() ClosePrice is not positive")
				}
				if price.TradingValue != price.ClosePrice*price.Volume {
					t.Errorf("parsePriceJSON() TradingValue mismatch: got %d, want %d",
						price.TradingValue, price.ClosePrice*price.Volume)
				}
			}
		})
	}
}

func TestParsePriceRegex(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    int // Expected number of prices
		wantErr bool
	}{
		{
			name: "valid regex format",
			body: `[["20240115", 72300, 73000, 72000, 72500, 1000000], ["20240116", 72500, 73500, 72300, 73000, 1200000]]`,
			want: 2,
			wantErr: false,
		},
		{
			name:    "invalid format",
			body:    `{"invalid": "json"}`,
			want:    0,
			wantErr: false,
		},
		{
			name:    "empty string",
			body:    "",
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{}
			got, err := c.parsePriceRegex(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePriceRegex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("parsePriceRegex() got %d prices, want %d", len(got), tt.want)
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int64
	}{
		{"float64", 123.45, 123},
		{"int64", int64(123), 123},
		{"int", int(123), 123},
		{"string", "123", 123},
		{"invalid string", "abc", 0},
		{"nil", nil, 0},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toInt64(tt.input); got != tt.want {
				t.Errorf("toInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriceDataStructure(t *testing.T) {
	now := time.Now()
	price := PriceData{
		StockCode:    "005930",
		TradeDate:    now,
		OpenPrice:    72000,
		HighPrice:    73000,
		LowPrice:     71500,
		ClosePrice:   72500,
		Volume:       1000000,
		TradingValue: 72500000000,
	}

	// Verify all fields are accessible
	if price.StockCode != "005930" {
		t.Errorf("StockCode = %s, want 005930", price.StockCode)
	}
	if price.OpenPrice != 72000 {
		t.Errorf("OpenPrice = %d, want 72000", price.OpenPrice)
	}
	if price.Volume != 1000000 {
		t.Errorf("Volume = %d, want 1000000", price.Volume)
	}
}
