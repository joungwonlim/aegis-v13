package dart

import (
	"fmt"
	"testing"
)

func TestIsMajorDisclosure(t *testing.T) {
	tests := []struct {
		name       string
		reportName string
		want       bool
	}{
		{"사업보고서", "사업보고서 (2024.01)", true},
		{"분기보고서", "분기보고서 (2024.3Q)", true},
		{"반기보고서", "반기보고서 (2024)", true},
		{"주요사항보고서", "주요사항보고서(유상증자결정)", true},
		{"유상증자", "유상증자결정", true},
		{"합병", "합병계약체결결정", true},
		{"자기주식", "자기주식취득신탁계약체결", true},
		{"일반공시", "감사보고서제출", false},
		{"기타공시", "임원ㆍ주요주주특정증권등소유상황보고서", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMajorDisclosure(tt.reportName); got != tt.want {
				t.Errorf("IsMajorDisclosure(%q) = %v, want %v", tt.reportName, got, tt.want)
			}
		})
	}
}

func TestGetCategory(t *testing.T) {
	tests := []struct {
		corpCls string
		want    DisclosureCategory
	}{
		{"Y", CategoryKOSPI},
		{"K", CategoryKOSDAQ},
		{"N", CategoryKONEX},
		{"E", CategoryETC},
		{"", CategoryETC},
		{"unknown", CategoryETC},
	}

	for _, tt := range tests {
		t.Run(string(tt.corpCls), func(t *testing.T) {
			if got := GetCategory(tt.corpCls); got != tt.want {
				t.Errorf("GetCategory(%q) = %v, want %v", tt.corpCls, got, tt.want)
			}
		})
	}
}

func TestGetDARTURL(t *testing.T) {
	tests := []struct {
		name    string
		rceptNo string
		want    string
	}{
		{
			name:    "valid receipt number",
			rceptNo: "20240115000123",
			want:    "https://dart.fss.or.kr/dsaf001/main.do?rcpNo=20240115000123",
		},
		{
			name:    "empty receipt number",
			rceptNo: "",
			want:    "https://dart.fss.or.kr/dsaf001/main.do?rcpNo=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDARTURL(tt.rceptNo); got != tt.want {
				t.Errorf("GetDARTURL(%q) = %v, want %v", tt.rceptNo, got, tt.want)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"사업보고서 (2024)", "사업보고서", true},
		{"분기보고서", "분기", true},
		{"주요사항보고서(유상증자)", "유상증자", true},
		{"감사보고서", "사업", false},
		{"", "test", false},
		{"test", "", true}, // Empty substring always matches
	}

	for _, tt := range tests {
		t.Run(tt.s+"/"+tt.substr, func(t *testing.T) {
			if got := containsString(tt.s, tt.substr); got != tt.want {
				t.Errorf("containsString(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"connection reset", fmt.Errorf("connection reset by peer"), true},
		{"EOF uppercase", fmt.Errorf("unexpected EOF"), true},
		{"eof lowercase", fmt.Errorf("read: eof"), true},
		{"timeout", fmt.Errorf("context deadline exceeded: i/o timeout"), true},
		{"connection refused", fmt.Errorf("dial tcp: connection refused"), true},
		{"auth error", fmt.Errorf("API error: 020 - invalid API key"), false},
		{"not found", fmt.Errorf("404 not found"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableError(tt.err); got != tt.want {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
