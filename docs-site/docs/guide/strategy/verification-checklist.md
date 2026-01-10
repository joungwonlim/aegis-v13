# strategyconfig 구현 검증 체크리스트

> Sonnet 구현 후 Opus가 검증하는 용도

---

## 검증 방법

각 항목별로 아래 명령어 또는 코드로 검증:

```bash
# 파일 존재 확인
ls -la backend/config/strategy/korea_equity_v13.yaml
ls -la backend/internal/strategyconfig/

# 테스트 실행
cd backend && go test ./internal/strategyconfig/...

# 빌드 확인
cd backend && go build ./...
```

---

## 1. YAML 파일 검증

### 1.1 파일 위치
- [ ] `backend/config/strategy/korea_equity_v13.yaml` 존재

### 1.2 오류 수정 확인

| # | 항목 | 기대값 | 검증 명령 |
|---|------|--------|----------|
| 1 | ADTV20 | `2_000_000_000` (20억) | `grep adtv20_min_krw` |
| 2 | exclude_krx_flags | 배열 형태 | `grep -A5 exclude_krx_flags` |
| 3 | spread.formula | 존재 | `grep formula` |
| 4 | avoid_open_minutes | 삭제됨 | `grep avoid_open` (없어야 함) |
| 5 | tier 비중 | 합 90% | 계산 확인 |
| 6 | version | `1.0.1` | `grep version` |

### 1.3 검증 스크립트

```bash
# 1. ADTV20 확인 (20억 = 2,000,000,000)
grep "adtv20_min_krw" backend/config/strategy/korea_equity_v13.yaml
# 기대: adtv20_min_krw: 2_000_000_000

# 2. exclude_krx_flags 확인
grep -A6 "exclude_krx_flags" backend/config/strategy/korea_equity_v13.yaml
# 기대: 배열 형태로 TRADING_HALT, ADMIN_ISSUE 등

# 3. spread.formula 확인
grep "formula" backend/config/strategy/korea_equity_v13.yaml
# 기대: formula: "((ask1-bid1)/((ask1+bid1)/2))"

# 4. avoid_open_minutes 삭제 확인
grep "avoid_open" backend/config/strategy/korea_equity_v13.yaml
# 기대: 결과 없음

# 5. tier 비중 확인
grep -A10 "tiers:" backend/config/strategy/korea_equity_v13.yaml
# 기대: 5×0.08 + 10×0.05 + 5×0.02 = 0.90
```

---

## 2. Go 패키지 검증

### 2.1 파일 구조

- [ ] `backend/internal/strategyconfig/config.go` 존재
- [ ] `backend/internal/strategyconfig/loader.go` 존재
- [ ] `backend/internal/strategyconfig/validate.go` 존재
- [ ] `backend/internal/strategyconfig/config_test.go` 존재

### 2.2 타입 정의 (config.go)

| 타입 | 필수 필드 | 검증 |
|------|----------|------|
| `Config` | Meta, Universe, Signals, Screening, Ranking, Portfolio, Execution, Exit, RiskOverlay, BacktestCost | grep 확인 |
| `Universe` | ExcludeKRXFlags []string, Filters | grep 확인 |
| `Spread` | MaxPct, Formula | grep 확인 |
| `RankingWeights` | Sum() 메서드 | grep 확인 |
| `Weighting` | TotalCount(), TotalWeightPct() 메서드 | grep 확인 |
| `DecisionSnapshot` | ConfigHash, ConfigYAML, StrategyID, GitCommit, DataSnapshotID, CreatedAt | grep 확인 |

```bash
# 타입 확인
grep "type Config struct" backend/internal/strategyconfig/config.go
grep "type DecisionSnapshot struct" backend/internal/strategyconfig/config.go
grep "func (w RankingWeights) Sum()" backend/internal/strategyconfig/config.go
grep "func (w Weighting) TotalCount()" backend/internal/strategyconfig/config.go
```

### 2.3 로더 (loader.go)

| 함수 | 시그니처 | 검증 |
|------|----------|------|
| `Load` | `func Load(path string) (*Config, []byte, error)` | grep 확인 |
| `Hash` | `func Hash(cfg *Config) (string, error)` | grep 확인 |
| `NewDecisionSnapshot` | `func NewDecisionSnapshot(...) (*DecisionSnapshot, error)` | grep 확인 |

```bash
grep "func Load" backend/internal/strategyconfig/loader.go
grep "func Hash" backend/internal/strategyconfig/loader.go
grep "func NewDecisionSnapshot" backend/internal/strategyconfig/loader.go
```

### 2.4 Validate (validate.go)

| 함수 | 검증 규칙 |
|------|----------|
| `Validate` | 아래 모든 규칙 포함 |
| `Warn` | 경고 규칙 포함 |

**Validate 필수 규칙:**

```bash
# 검증 규칙 확인
grep "strategy_id" backend/internal/strategyconfig/validate.go
grep "adtv20_min_krw" backend/internal/strategyconfig/validate.go
grep "spread.formula\|Formula" backend/internal/strategyconfig/validate.go
grep "Sum() != 100\|Sum().*100" backend/internal/strategyconfig/validate.go
grep "TotalCount()" backend/internal/strategyconfig/validate.go
grep "TotalWeightPct()" backend/internal/strategyconfig/validate.go
grep "FIXED.*ATR\|ATR.*FIXED" backend/internal/strategyconfig/validate.go
```

**Warn 필수 규칙:**

```bash
grep "LOW_ADTV" backend/internal/strategyconfig/validate.go
grep "OPTIMISTIC_SLIPPAGE" backend/internal/strategyconfig/validate.go
```

---

## 3. 테스트 검증

### 3.1 테스트 실행

```bash
cd backend && go test -v ./internal/strategyconfig/...
```

### 3.2 필수 테스트 케이스

- [ ] `TestLoad` - YAML 로드 성공
- [ ] `TestValidateWeights` - 가중치 합 검증
- [ ] `TestValidateTiers` - Tier 검증
- [ ] `TestWarn` - 경고 발생 확인
- [ ] Hash 결정성 검증 (동일 설정 → 동일 해시)

```bash
grep "func TestLoad" backend/internal/strategyconfig/config_test.go
grep "func TestValidateWeights" backend/internal/strategyconfig/config_test.go
grep "func TestValidateTiers" backend/internal/strategyconfig/config_test.go
grep "func TestWarn" backend/internal/strategyconfig/config_test.go
```

---

## 4. DB 마이그레이션 검증

### 4.1 파일 확인

- [ ] `backend/migrations/013_create_decision_snapshots.sql` 존재

### 4.2 테이블 구조

```bash
grep "CREATE TABLE audit.decision_snapshots" backend/migrations/013_create_decision_snapshots.sql
grep "config_hash" backend/migrations/013_create_decision_snapshots.sql
grep "config_yaml" backend/migrations/013_create_decision_snapshots.sql
```

**필수 컬럼:**
- [ ] `config_hash VARCHAR(64)`
- [ ] `config_yaml TEXT`
- [ ] `strategy_id VARCHAR(50)`
- [ ] `git_commit VARCHAR(40)`
- [ ] `data_snapshot_id VARCHAR(50)`
- [ ] `decision_date DATE`
- [ ] `created_at TIMESTAMPTZ`

---

## 5. 빌드 검증

```bash
cd backend && go build ./...
# 에러 없이 완료되어야 함
```

---

## 6. 통합 검증

### 6.1 전체 흐름 테스트

```go
// 이 코드가 에러 없이 실행되어야 함
cfg, yamlData, err := strategyconfig.Load("config/strategy/korea_equity_v13.yaml")
if err != nil {
    // FAIL
}

warnings := strategyconfig.Warn(cfg)
// warnings 출력

hash, err := strategyconfig.Hash(cfg)
// hash 길이 64

snapshot, err := strategyconfig.NewDecisionSnapshot(cfg, yamlData, "test", "test")
// snapshot.ConfigHash == hash
```

---

## 7. 검증 결과 기록

| 항목 | 상태 | 비고 |
|------|------|------|
| YAML 파일 수정 | ✅ | ADTV20=2e9, spread.formula 포함 |
| config.go 타입 | ✅ | 310줄, 모든 타입/메서드 구현 |
| loader.go 함수 | ✅ | Load, Hash, NewDecisionSnapshot |
| validate.go 규칙 | ✅ | 25+ 검증 규칙, Warn 함수 |
| config_test.go | ✅ | 7개 테스트 케이스 |
| 테스트 통과 | ✅ | 6 passed, 1 skipped (경로 문제) |
| DB 마이그레이션 | ✅ | 013_create_decision_snapshots.sql |
| 빌드 성공 | ✅ | strategyconfig 패키지 빌드 OK |

> 검증일: 2026-01-10

---

## 8. 검증 실패 시 조치

1. **YAML 오류**: `implementation-spec.md`의 Section 2.2 참조하여 수정
2. **타입 누락**: `implementation-spec.md`의 Section 3 참조
3. **Validate 규칙 누락**: `implementation-spec.md`의 Section 5 참조
4. **테스트 실패**: 에러 메시지 확인 후 해당 코드 수정

---

## 9. 자동 검증 스크립트

```bash
#!/bin/bash
# verify_strategyconfig.sh

echo "=== strategyconfig 구현 검증 ==="

# 1. 파일 존재 확인
echo -n "1. YAML 파일: "
[ -f "backend/config/strategy/korea_equity_v13.yaml" ] && echo "OK" || echo "FAIL"

echo -n "2. config.go: "
[ -f "backend/internal/strategyconfig/config.go" ] && echo "OK" || echo "FAIL"

echo -n "3. loader.go: "
[ -f "backend/internal/strategyconfig/loader.go" ] && echo "OK" || echo "FAIL"

echo -n "4. validate.go: "
[ -f "backend/internal/strategyconfig/validate.go" ] && echo "OK" || echo "FAIL"

echo -n "5. config_test.go: "
[ -f "backend/internal/strategyconfig/config_test.go" ] && echo "OK" || echo "FAIL"

echo -n "6. 마이그레이션: "
[ -f "backend/migrations/013_create_decision_snapshots.sql" ] && echo "OK" || echo "FAIL"

# 2. ADTV20 값 확인
echo -n "7. ADTV20=2e9: "
grep -q "adtv20_min_krw: 2_000_000_000" backend/config/strategy/korea_equity_v13.yaml && echo "OK" || echo "FAIL"

# 3. 테스트 실행
echo "8. 테스트 실행..."
cd backend && go test ./internal/strategyconfig/... -v

# 4. 빌드
echo "9. 빌드..."
cd backend && go build ./...

echo "=== 검증 완료 ==="
```
