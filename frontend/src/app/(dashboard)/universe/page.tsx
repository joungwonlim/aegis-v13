'use client'

/**
 * 유니버스 (투자 가능 종목) 페이지
 * SSOT: universe 모듈 + stock 모듈 사용
 *
 * S1-S5 파이프라인 단계별 데이터 표시
 */

import { useState, useMemo } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Input } from '@/shared/components/ui/input'
import { Badge } from '@/shared/components/ui/badge'
import {
  StockDataTable,
  type StockDataItem,
  type StockDataColumn,
} from '@/modules/stock/components'
import { Search, RefreshCw, Loader2 } from 'lucide-react'
import {
  useRanking,
  useUniverse,
  useSignals,
  useScreened,
  usePipelineRanking,
  usePortfolio,
  type RankingCategory,
  type MarketType,
} from '@/modules/universe'
import { PipelineSteps, StepConditions, type PipelineStep } from '@/modules/universe/components'
import { cn } from '@/shared/lib/utils'

// 마켓 필터 옵션
const MARKET_OPTIONS: { value: MarketType; label: string }[] = [
  { value: 'ALL', label: '전체' },
  { value: 'KOSPI', label: '코스피' },
  { value: 'KOSDAQ', label: '코스닥' },
]

// S1 카테고리 옵션 (UI 버튼)
const CATEGORY_OPTIONS: { value: RankingCategory; label: string }[] = [
  { value: 'trading', label: '거래량' },
  { value: 'upper', label: '상승' },
  { value: 'lower', label: '하락' },
  { value: 'capitalization', label: '시가총액' },
  { value: 'high52week', label: '52주 신고가' },
  { value: 'low52week', label: '52주 신저가' },
  { value: 'new', label: '신규상장' },
]

// 카테고리별 제목 매핑
const CATEGORY_TITLES: Record<RankingCategory, string> = {
  trading: '거래량순',
  upper: '상승률순',
  lower: '하락률순',
  capitalization: '시가총액순',
  high52week: '52주 신고가',
  low52week: '52주 신저가',
  top: '시가총액순',
  quantHigh: '퀀트 상위',
  quantLow: '퀀트 하위',
  priceTop: '고가순',
  new: '신규상장',
}

// 단계별 제목
const STEP_TITLES: Record<PipelineStep, string> = {
  S1: '', // S1은 카테고리에 따라 동적 생성
  S2: 'Signals - 팩터 점수',
  S3: 'Screener - Hard Cut 통과',
  S4: 'Ranking - 종합 순위',
  S5: 'Portfolio - 포트폴리오',
}

// S1: 유니버스 컬럼
const s1Columns: StockDataColumn[] = [
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

// S2: 시그널 컬럼 (팩터 점수)
const s2Columns: StockDataColumn[] = [
  {
    key: 'momentum',
    label: 'Momentum',
    width: 'w-20',
    align: 'right',
    sortable: true,
    aggregation: 'avg',
    render: (item) => <ScoreCell value={item.momentum as number} />,
  },
  {
    key: 'technical',
    label: 'Technical',
    width: 'w-20',
    align: 'right',
    sortable: true,
    aggregation: 'avg',
    render: (item) => <ScoreCell value={item.technical as number} />,
  },
  {
    key: 'value',
    label: 'Value',
    width: 'w-20',
    align: 'right',
    sortable: true,
    aggregation: 'avg',
    render: (item) => <ScoreCell value={item.value as number} />,
  },
  {
    key: 'quality',
    label: 'Quality',
    width: 'w-20',
    align: 'right',
    sortable: true,
    aggregation: 'avg',
    render: (item) => <ScoreCell value={item.quality as number} />,
  },
  {
    key: 'flow',
    label: 'Flow',
    width: 'w-20',
    align: 'right',
    sortable: true,
    aggregation: 'avg',
    render: (item) => <ScoreCell value={item.flow as number} />,
  },
  {
    key: 'event',
    label: 'Event',
    width: 'w-20',
    align: 'right',
    sortable: true,
    aggregation: 'avg',
    render: (item) => <ScoreCell value={item.event as number} />,
  },
  {
    key: 'totalScore',
    label: '종합점수',
    width: 'w-24',
    align: 'right',
    sortable: true,
    aggregation: 'avg',
    render: (item) => {
      const score = item.totalScore as number | undefined
      return (
        <span className="font-mono font-semibold text-primary">
          {score?.toFixed(2) ?? '-'}
        </span>
      )
    },
  },
]

// S3: 스크리너 컬럼 (Hard Cut 통과 종목 - 팩터 점수 표시)
const s3Columns: StockDataColumn[] = [
  {
    key: 'momentum',
    label: 'Momentum',
    width: 'w-20',
    align: 'right',
    render: (item) => <ScoreCell value={item.momentum as number} />,
  },
  {
    key: 'technical',
    label: 'Technical',
    width: 'w-20',
    align: 'right',
    render: (item) => <ScoreCell value={item.technical as number} />,
  },
  {
    key: 'flow',
    label: 'Flow',
    width: 'w-20',
    align: 'right',
    render: (item) => <ScoreCell value={item.flow as number} />,
  },
  {
    key: 'totalScore',
    label: '종합점수',
    width: 'w-24',
    align: 'right',
    render: (item) => {
      const score = item.totalScore as number | undefined
      return (
        <span className="font-mono font-semibold text-primary">
          {score?.toFixed(2) ?? '-'}
        </span>
      )
    },
  },
  {
    key: 'passStatus',
    label: '상태',
    width: 'w-20',
    align: 'center',
    render: (item) => {
      const status = item.passStatus as string
      return (
        <Badge variant={status === 'pass' ? 'default' : 'secondary'} className="text-xs">
          {status === 'pass' ? '통과' : '탈락'}
        </Badge>
      )
    },
  },
]

// S4: 랭킹 컬럼
const s4Columns: StockDataColumn[] = [
  {
    key: 'totalScore',
    label: '종합점수',
    width: 'w-24',
    align: 'right',
    render: (item) => {
      const score = item.totalScore as number | undefined
      return (
        <span className="font-mono font-semibold text-primary">
          {score?.toFixed(2) ?? '-'}
        </span>
      )
    },
  },
  {
    key: 'rankChange',
    label: '순위변동',
    width: 'w-20',
    align: 'right',
    render: (item) => {
      const change = (item.rankChange as number) ?? 0
      return (
        <span
          className={cn(
            'font-mono text-sm',
            change > 0 && 'text-red-500',
            change < 0 && 'text-blue-500'
          )}
        >
          {change > 0 ? `▲${change}` : change < 0 ? `▼${Math.abs(change)}` : '-'}
        </span>
      )
    },
  },
  {
    key: 'momentum',
    label: 'Momentum',
    width: 'w-20',
    align: 'right',
    render: (item) => <ScoreCell value={item.momentum as number} />,
  },
  {
    key: 'flow',
    label: 'Flow',
    width: 'w-20',
    align: 'right',
    render: (item) => <ScoreCell value={item.flow as number} />,
  },
]

// S5: 포트폴리오 컬럼
const s5Columns: StockDataColumn[] = [
  {
    key: 'weight',
    label: '비중',
    width: 'w-20',
    align: 'right',
    render: (item) => {
      const weight = item.weight as number | undefined
      return (
        <span className="font-mono font-semibold">{weight ? `${weight.toFixed(1)}%` : '-'}</span>
      )
    },
  },
  {
    key: 'targetQty',
    label: '목표수량',
    width: 'w-20',
    align: 'right',
    render: (item) => {
      const qty = item.targetQty as number | undefined
      return <span className="font-mono">{qty?.toLocaleString() ?? '-'}</span>
    },
  },
  {
    key: 'returnRate',
    label: '수익률',
    width: 'w-20',
    align: 'right',
    render: (item) => {
      const rate = (item.returnRate as number) ?? 0
      return (
        <span
          className={cn(
            'font-mono font-semibold',
            rate > 0 && 'text-red-500',
            rate < 0 && 'text-blue-500'
          )}
        >
          {rate > 0 ? '+' : ''}{rate.toFixed(2)}%
        </span>
      )
    },
  },
  {
    key: 'holdingDays',
    label: '보유일',
    width: 'w-16',
    align: 'right',
    render: (item) => {
      const days = item.holdingDays as number | undefined
      return <span className="font-mono text-muted-foreground">{days ?? '-'}일</span>
    },
  },
]

// 점수 셀 컴포넌트
function ScoreCell({ value }: { value?: number }) {
  if (value === undefined || value === null) return <span className="text-muted-foreground">-</span>
  const color =
    value >= 0.5 ? 'text-red-500' : value >= 0 ? 'text-orange-500' : value >= -0.5 ? 'text-blue-400' : 'text-blue-600'
  return <span className={cn('font-mono text-sm font-medium', color)}>{value.toFixed(2)}</span>
}

// 단계별 컬럼 매핑
const STEP_COLUMNS: Record<PipelineStep, StockDataColumn[]> = {
  S1: s1Columns,
  S2: s2Columns,
  S3: s3Columns,
  S4: s4Columns,
  S5: s5Columns,
}

export default function UniversePage() {
  const [searchTerm, setSearchTerm] = useState('')
  const [market, setMarket] = useState<MarketType>('ALL')
  const [activeStep, setActiveStep] = useState<PipelineStep>('S1')
  const [showConditions, setShowConditions] = useState(false)
  const [category, setCategory] = useState<RankingCategory>('trading')
  const [useNaverRanking, setUseNaverRanking] = useState(false) // Naver 랭킹 모드

  // S1: Brain이 생성한 Universe 데이터 (기본)
  const { data: universeData, isLoading: isLoadingUniverse, refetch: refetchUniverse } = useUniverse(market)

  // S1: Naver 랭킹 데이터 (버튼 클릭 시)
  const { data: rankingData, isLoading: isLoadingRanking, error: errorRanking, refetch: refetchRanking } = useRanking(category, market)

  // S2: 신호 데이터 조회
  const { data: signalsData, isLoading: isLoadingSignals, refetch: refetchSignals } = useSignals(market)

  // S3: 스크리닝 데이터 조회 (Hard Cut 통과 종목)
  const { data: screenedData, isLoading: isLoadingScreened, refetch: refetchScreened } = useScreened(market)

  // S4: 파이프라인 랭킹 조회
  const { data: pipelineRankingData, isLoading: isLoadingPipelineRanking, refetch: refetchPipelineRanking } = usePipelineRanking(market)

  // S5: 포트폴리오 조회
  const { data: portfolioData, isLoading: isLoadingPortfolio, refetch: refetchPortfolio } = usePortfolio()

  // 로딩 상태 (현재 단계에 따라)
  const isLoading = activeStep === 'S1'
    ? (useNaverRanking ? isLoadingRanking : isLoadingUniverse)
    : activeStep === 'S2' ? isLoadingSignals
    : activeStep === 'S3' ? isLoadingScreened
    : activeStep === 'S4' ? isLoadingPipelineRanking
    : activeStep === 'S5' ? isLoadingPortfolio
    : isLoadingUniverse

  const error = errorRanking

  // 새로고침 핸들러
  const refetch = () => {
    if (activeStep === 'S1') {
      if (useNaverRanking) refetchRanking()
      else refetchUniverse()
    }
    else if (activeStep === 'S2') refetchSignals()
    else if (activeStep === 'S3') refetchScreened()
    else if (activeStep === 'S4') refetchPipelineRanking()
    else if (activeStep === 'S5') refetchPortfolio()
    else refetchUniverse()
  }

  // Naver 랭킹 버튼 핸들러
  const handleNaverRankingClick = () => {
    setUseNaverRanking(true)
    refetchRanking()
  }

  // 파이프라인 단계별 종목 수 (실제 데이터 기반)
  const pipelineCounts: Partial<Record<PipelineStep, number>> = {
    S1: useNaverRanking
      ? (rankingData?.data?.count ?? 0)
      : (universeData?.data?.count ?? 0),
    S2: signalsData?.data?.count ?? 0,
    S3: screenedData?.data?.count ?? 0, // S3: Hard Cut 통과 종목
    S4: pipelineRankingData?.data?.count ?? 0,
    S5: portfolioData?.data?.count ?? 0,
  }

  // 스텝 클릭 핸들러
  const handleStepClick = (step: PipelineStep) => {
    if (activeStep === step) {
      setShowConditions(!showConditions)
    } else {
      setActiveStep(step)
      setShowConditions(true)
    }
  }

  // 단계별 데이터 변환
  const tableData: StockDataItem[] = useMemo(() => {
    // S2: 신호 데이터
    if (activeStep === 'S2' && signalsData?.data?.items) {
      return signalsData.data.items.map((item, idx) => ({
        code: item.stockCode,
        name: item.stockName,
        market: item.market,
        price: 0,
        change: 0,
        changeRate: 0,
        volume: 0,
        marketCap: 0,
        rank: idx + 1,
        uniqueKey: `${item.market}_${item.stockCode}`,
        momentum: item.momentum,
        technical: item.technical,
        value: item.value,
        quality: item.quality,
        flow: item.flow,
        event: item.event,
        totalScore: item.totalScore,
      }))
    }

    // S4: 파이프라인 랭킹 데이터
    if (activeStep === 'S4' && pipelineRankingData?.data?.items) {
      return pipelineRankingData.data.items.map((item) => ({
        code: item.stockCode,
        name: item.stockName,
        market: item.market,
        price: item.currentPrice,
        change: 0,
        changeRate: item.changeRate,
        volume: 0,
        marketCap: 0,
        rank: item.rank,
        uniqueKey: `${item.market}_${item.stockCode}`,
        momentum: item.momentum,
        technical: item.technical,
        value: item.value,
        quality: item.quality,
        flow: item.flow,
        event: item.event,
        totalScore: item.totalScore,
        rankChange: 0,
      }))
    }

    // S5: 포트폴리오 데이터
    if (activeStep === 'S5' && portfolioData?.data?.positions) {
      return portfolioData.data.positions.map((item, idx) => ({
        code: item.stockCode,
        name: item.stockName,
        market: item.market,
        price: item.currentPrice,
        change: 0,
        changeRate: item.changeRate,
        volume: 0,
        marketCap: 0,
        rank: idx + 1,
        uniqueKey: `${item.market}_${item.stockCode}`,
        weight: item.weight * 100, // 비중을 퍼센트로
        targetQty: item.targetQty,
        returnRate: item.changeRate,
        holdingDays: 0,
        action: item.action,
        reason: item.reason,
      }))
    }

    // S1: Universe 데이터 (Brain이 생성) 또는 Naver 랭킹 데이터
    if (activeStep === 'S1') {
      if (useNaverRanking && rankingData?.data?.items) {
        // Naver 랭킹 데이터
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
      }

      // Universe 데이터 (기본)
      if (universeData?.data?.items) {
        return universeData.data.items.map((item, idx) => ({
          code: item.stockCode,
          name: item.stockName,
          market: item.market,
          price: item.currentPrice,
          change: 0,
          changeRate: item.changeRate,
          volume: item.volume,
          marketCap: item.marketCap,
          rank: idx + 1,
          uniqueKey: `${item.market}_${item.stockCode}`,
        }))
      }

      return []
    }

    // S3: Screener - Hard Cut 통과 종목 (API로 필터링된 데이터)
    if (activeStep === 'S3' && screenedData?.data?.items) {
      return screenedData.data.items.map((item, idx) => ({
        code: item.stockCode,
        name: item.stockName,
        market: item.market,
        price: 0,
        change: 0,
        changeRate: 0,
        volume: 0,
        marketCap: 0,
        rank: idx + 1,
        uniqueKey: `${item.market}_${item.stockCode}`,
        momentum: item.momentum,
        technical: item.technical,
        value: item.value,
        quality: item.quality,
        flow: item.flow,
        event: item.event,
        totalScore: item.totalScore,
        passStatus: item.passedAll ? 'pass' : 'fail',
      }))
    }

    return []
  }, [rankingData, universeData, signalsData, screenedData, pipelineRankingData, portfolioData, activeStep, useNaverRanking])

  // 검색 필터링
  const filteredData = useMemo(() => {
    if (!searchTerm) return tableData
    return tableData.filter(
      (item) =>
        item.code.includes(searchTerm) ||
        item.name?.toLowerCase().includes(searchTerm.toLowerCase())
    )
  }, [tableData, searchTerm])

  // 현재 단계의 컬럼
  const currentColumns = STEP_COLUMNS[activeStep]

  if (error) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-destructive">데이터를 불러오는데 실패했습니다.</p>
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
          {showConditions && (
            <StepConditions step={activeStep} onClose={() => setShowConditions(false)} />
          )}
        </CardContent>
      </Card>

      {/* 필터 바 */}
      <div className="flex items-center gap-1 flex-wrap bg-muted/30 p-2 rounded-lg">
        {/* 현재 단계 표시 */}
        <Badge variant="outline" className="font-mono mr-2">
          {activeStep}
        </Badge>

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

        {/* S1일 때 카테고리 필터 + 네이버 실시간 버튼 */}
        {activeStep === 'S1' && (
          <>
            <div className="w-px h-6 bg-border mx-2" />
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
            {/* 네이버 실시간 가져오기 버튼 */}
            <Button
              variant={useNaverRanking ? 'default' : 'outline'}
              size="sm"
              className={cn(
                'h-8 px-3 text-sm ml-2',
                useNaverRanking && 'bg-green-600 hover:bg-green-700 text-white'
              )}
              onClick={handleNaverRankingClick}
            >
              {useNaverRanking ? '✓ 실시간' : '네이버 실시간 가져오기'}
            </Button>
          </>
        )}

        {/* 구분선 */}
        <div className="w-px h-6 bg-border mx-2" />

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
            {activeStep === 'S1'
              ? (useNaverRanking ? `Naver 실시간 - ${CATEGORY_TITLES[category]}` : `Universe - ${CATEGORY_TITLES[category]}`)
              : STEP_TITLES[activeStep]}
            <span className="text-muted-foreground font-normal">
              ({searchTerm ? `검색 ${filteredData.length}개` : `${filteredData.length}개`})
            </span>
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
              extraColumns={currentColumns}
              showSummary={activeStep === 'S2'}
              emptyMessage="조건에 맞는 종목이 없습니다."
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
