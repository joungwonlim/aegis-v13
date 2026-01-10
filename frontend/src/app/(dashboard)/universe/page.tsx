'use client'

/**
 * 유니버스 (투자 가능 종목) 페이지
 * SSOT: universe 모듈 + stock 모듈 사용
 *
 * 포트폴리오 연동 (보유종목 녹색점, 청산 모니터링 빨간점)은
 * StockDataTable 모듈 내부에서 자동 처리됨
 */

import { useState, useMemo } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Input } from '@/shared/components/ui/input'
import {
  StockDataTable,
  type StockDataItem,
  type StockDataColumn,
} from '@/modules/stock/components'
import { Search, RefreshCw, Loader2 } from 'lucide-react'
import { useRanking, type RankingCategory, type MarketType } from '@/modules/universe'
import { PipelineSteps, StepConditions, type PipelineStep } from '@/modules/universe/components'
import { cn } from '@/shared/lib/utils'

// 마켓 필터 옵션
const MARKET_OPTIONS: { value: MarketType; label: string }[] = [
  { value: 'ALL', label: '전체' },
  { value: 'KOSPI', label: '코스피' },
  { value: 'KOSDAQ', label: '코스닥' },
]

// 카테고리 필터 옵션
const CATEGORY_OPTIONS: { value: RankingCategory; label: string }[] = [
  { value: 'trading', label: '거래량 상위' },
  { value: 'quantHigh', label: '거래량 급증' },
  { value: 'quantLow', label: '거래량 급락' },
  { value: 'top', label: '거래대금 상위' },
  { value: 'upper', label: '상승' },
  { value: 'lower', label: '하락' },
  { value: 'new', label: '신규상장' },
  { value: 'high52week', label: '52주 최고' },
  { value: 'low52week', label: '52주 최저' },
]

// 유니버스용 추가 컬럼 정의 (기본 컬럼: 순번, 종목명, 현재가, 전일대비)
const universeExtraColumns: StockDataColumn[] = [
  {
    key: 'volume',
    label: '거래량',
    width: 'w-24',
    align: 'right',
    render: (item) => (
      <span className="font-mono text-muted-foreground">
        {item.volume != null ? item.volume.toLocaleString('ko-KR') : '-'}
      </span>
    ),
  },
  {
    key: 'marketCap',
    label: '시가총액(억)',
    width: 'w-28',
    align: 'right',
    render: (item) => (
      <span className="font-mono">
        {item.marketCap ? item.marketCap.toLocaleString('ko-KR') : '-'}
      </span>
    ),
  },
]

export default function UniversePage() {
  const [searchTerm, setSearchTerm] = useState('')
  const [category, setCategory] = useState<RankingCategory>('trading')
  const [market, setMarket] = useState<MarketType>('ALL')
  const [activeStep, setActiveStep] = useState<PipelineStep | null>(null)

  // 랭킹 데이터 조회
  const { data: rankingData, isLoading, error, refetch } = useRanking(category, market)

  // 파이프라인 단계별 종목 수 (예시 - 실제로는 API에서 가져와야 함)
  const pipelineCounts: Partial<Record<PipelineStep, number>> = {
    S1: 2847,
    S2: 2847,
    S3: 523,
    S4: 50,
    S5: 20,
  }

  // 스텝 클릭 핸들러
  const handleStepClick = (step: PipelineStep) => {
    setActiveStep(activeStep === step ? null : step)
  }

  // RankingItem → StockDataItem 변환
  const tableData: StockDataItem[] = useMemo(() => {
    if (!rankingData?.data?.items) return []
    return rankingData.data.items.map((item, idx) => ({
      code: item.StockCode,
      name: item.StockName,
      market: item.Market,
      price: item.CurrentPrice,
      change: item.PriceChange,
      changeRate: item.ChangeRate,
      volume: item.Volume,
      marketCap: item.MarketCap,
      rank: idx + 1,
      uniqueKey: `${item.Market}_${item.StockCode}`,
    }))
  }, [rankingData])

  // 검색 필터링
  const filteredData = useMemo(() => {
    if (!searchTerm) return tableData
    return tableData.filter(
      (item) =>
        item.code.includes(searchTerm) ||
        item.name?.toLowerCase().includes(searchTerm.toLowerCase())
    )
  }, [tableData, searchTerm])

  if (error) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-destructive">유니버스를 불러오는데 실패했습니다.</p>
            <p className="text-sm text-muted-foreground mt-2">
              백엔드 서버 연결을 확인해주세요.
            </p>
            <Button variant="outline" className="mt-4" onClick={() => refetch()}>
              다시 시도
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-4">
      {/* 파이프라인 스텝 */}
      <Card>
        <CardContent className="pt-6">
          <PipelineSteps
            activeStep={activeStep}
            onStepClick={handleStepClick}
            counts={pipelineCounts}
          />
          {activeStep && (
            <StepConditions step={activeStep} onClose={() => setActiveStep(null)} />
          )}
        </CardContent>
      </Card>

      {/* 필터 바 */}
      <div className="flex items-center gap-1 flex-wrap bg-muted/30 p-2 rounded-lg">
        {/* 마켓 필터 */}
        {MARKET_OPTIONS.map((opt) => (
          <Button
            key={opt.value}
            variant={market === opt.value ? 'default' : 'ghost'}
            size="sm"
            className={cn(
              'h-8 px-3 text-sm',
              market === opt.value && 'bg-primary text-primary-foreground'
            )}
            onClick={() => setMarket(opt.value)}
          >
            {opt.label}
          </Button>
        ))}

        {/* 구분선 */}
        <div className="w-px h-6 bg-border mx-1" />

        {/* 카테고리 필터 */}
        {CATEGORY_OPTIONS.map((opt) => (
          <Button
            key={opt.value}
            variant={category === opt.value ? 'secondary' : 'ghost'}
            size="sm"
            className={cn(
              'h-8 px-3 text-sm',
              category === opt.value && 'bg-secondary font-medium'
            )}
            onClick={() => setCategory(opt.value)}
          >
            {opt.label}
          </Button>
        ))}

        {/* 구분선 */}
        <div className="w-px h-6 bg-border mx-1" />

        {/* 검색 */}
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="종목 검색..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="h-8 pl-8 w-36 text-sm"
          />
        </div>

        {/* 새로고침 */}
        <Button
          variant="ghost"
          size="sm"
          className="h-8 w-8 p-0"
          onClick={() => refetch()}
          disabled={isLoading}
        >
          {isLoading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
        </Button>
      </div>

      {/* 종목 테이블 */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-lg flex items-center gap-2">
            {searchTerm
              ? `검색 결과: ${filteredData.length}개`
              : `전체 ${tableData.length}개 종목`}
            {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="py-12 flex items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <StockDataTable
              data={filteredData}
              extraColumns={universeExtraColumns}
              emptyMessage="조건에 맞는 종목이 없습니다."
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
