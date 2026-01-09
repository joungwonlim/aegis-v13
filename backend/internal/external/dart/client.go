package dart

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// Client handles communication with DART (Data Analysis, Retrieval and Transfer System) API
// ⭐ SSOT: DART API 호출은 이 클라이언트에서만
type Client struct {
	httpClient *http.Client
	logger     *logger.Logger
	apiKey     string
	baseURL    string
}

// NewClient creates a new DART API client
// DART API requires legacy TLS configuration (RSA key exchange)
func NewClient(apiKey string, log *logger.Logger) *Client {
	return &Client{
		httpClient: newLegacyCompatibleClient(30 * time.Second),
		logger:     log,
		apiKey:     apiKey,
		baseURL:    "https://opendart.fss.or.kr",
	}
}

// newLegacyCompatibleClient creates an HTTP client compatible with legacy TLS servers
// DART server requires RSA key exchange cipher suites which Go 1.22+ no longer offers by default
func newLegacyCompatibleClient(timeout time.Duration) *http.Client {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,

		// Include RSA KEX cipher suites for legacy server compatibility
		// DART server doesn't support ECDHE, so we need RSA key exchange
		CipherSuites: []uint16{
			// ECDHE (modern) - will be used if server supports
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,

			// RSA KEX (legacy) - required for DART API
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	tr := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: false, // Disable HTTP/2 for legacy server compatibility

		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		TLSHandshakeTimeout:   10 * time.Second,
		TLSClientConfig:       tlsCfg,
		MaxIdleConns:          20,
		MaxConnsPerHost:       5, // Reduced to avoid overwhelming DART API
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}
}

// DisclosureResponse represents DART API response for disclosure list
type DisclosureResponse struct {
	Status      string       `json:"status"`
	Message     string       `json:"message"`
	PageNo      int          `json:"page_no"`
	PageCount   int          `json:"page_count"`
	TotalCount  int          `json:"total_count"`
	TotalPage   int          `json:"total_page"`
	Disclosures []Disclosure `json:"list"`
}

// Disclosure represents a single disclosure item
type Disclosure struct {
	CorpCode  string `json:"corp_code"`
	CorpName  string `json:"corp_name"`
	StockCode string `json:"stock_code"`
	CorpCls   string `json:"corp_cls"`  // Y: 유가, K: 코스닥, N: 코넥스, E: 기타
	ReportNm  string `json:"report_nm"` // 공시 제목
	RceptNo   string `json:"rcept_no"`  // 접수번호
	FlrNm     string `json:"flr_nm"`    // 공시 제출인
	RceptDt   string `json:"rcept_dt"`  // 접수일자 (YYYYMMDD)
	Rm        string `json:"rm"`        // 비고
}

// DisclosureCategory represents disclosure category
type DisclosureCategory string

const (
	CategoryKOSPI  DisclosureCategory = "KOSPI"
	CategoryKOSDAQ DisclosureCategory = "KOSDAQ"
	CategoryKONEX  DisclosureCategory = "KONEX"
	CategoryETC    DisclosureCategory = "ETC"
)

// IsMajorDisclosure checks if the disclosure is a major one
func IsMajorDisclosure(reportName string) bool {
	majorKeywords := []string{
		"사업보고서",
		"분기보고서",
		"반기보고서",
		"주요사항보고서",
		"유상증자",
		"무상증자",
		"합병",
		"분할",
		"영업양수도",
		"자기주식",
		"전환사채",
		"신주인수권부사채",
	}

	for _, keyword := range majorKeywords {
		if containsString(reportName, keyword) {
			return true
		}
	}
	return false
}

// GetCategory returns disclosure category based on corp_cls
func GetCategory(corpCls string) DisclosureCategory {
	switch corpCls {
	case "Y":
		return CategoryKOSPI
	case "K":
		return CategoryKOSDAQ
	case "N":
		return CategoryKONEX
	default:
		return CategoryETC
	}
}

// GetDARTURL builds the DART disclosure URL
func GetDARTURL(rceptNo string) string {
	return "https://dart.fss.or.kr/dsaf001/main.do?rcpNo=" + rceptNo
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
