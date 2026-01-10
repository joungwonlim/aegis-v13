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
import { useRanking, type RankingCategory, type MarketType } from '@/modules/universe'
import { PipelineSteps, StepConditions, type PipelineStep } from '@/modules/universe/components'
import { cn } from '@/shared/lib/utils'

// 마켓 필터 옵션
const MARKET_OPTIONS: { value: MarketType; label: string }[] = [
  { value: 'ALL', label: '전체' },
  { value: 'KOSPI', label: '코스피' },
  { value: 'KOSDAQ', label: '코스닥' },
]

// 단계별 제목
const STEP_TITLES: Record<PipelineStep, string> = {
  S1: 'Universe - 투자 가능 종목',
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
    key: 'value',
    label: 'Value',
    width: 'w-20',
    align: 'right',
    render: (item) => <ScoreCell value={item.value as number} />,
  },
  {
    key: 'quality',
    label: 'Quality',
    width: 'w-20',
    align: 'right',
    render: (item) => <ScoreCell value={item.quality as number} />,
  },
  {
    key: 'flow',
    label: 'Flow',
    width: 'w-20',
    align: 'right',
    render: (item) => <ScoreCell value={item.flow as number} />,
  },
  {
    key: 'event',
    label: 'Event',
    width: 'w-20',
    align: 'right',
    render: (item) => <ScoreCell value={item.event as number} />,
  },
]

// S3: 스크리너 컬럼
const s3Columns: StockDataColumn[] = [
  {
    key: 'per',
    label: 'PER',
    width: 'w-16',
    align: 'right',
    render: (item) => {
      const per = item.per as number | undefined
      return <span className="font-mono text-sm">{per?.toFixed(1) ?? '-'}</span>
    },
  },
  {
    key: 'pbr',
    label: 'PBR',
    width: 'w-16',
    align: 'right',
    render: (item) => {
      const pbr = item.pbr as number | undefined
      return <span className="font-mono text-sm">{pbr?.toFixed(2) ?? '-'}</span>
    },
  },
  {
    key: 'roe',
    label: 'ROE',
    width: 'w-16',
    align: 'right',
    render: (item) => {
      const roe = item.roe as number | undefined
      return <span className="font-mono text-sm">{roe ? `${roe.toFixed(1)}%` : '-'}</span>
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

  // S1은 trading 카테고리, S4는 top 카테고리 사용 (예시)
  const category: RankingCategory = activeStep === 'S4' ? 'top' : 'trading'

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
    if (activeStep === step) {
      setShowConditions(!showConditions)
    } else {
      setActiveStep(step)
      setShowConditions(true)
    }
  }

  // 단계별 데이터 변환
  const tableData: StockDataItem[] = useMemo(() => {
    if (!rankingData?.data?.items) return []

    const baseData = rankingData.data.items.map((item, idx) => ({
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
      // 시뮬레이션 데이터 (실제로는 API에서)
      momentum: Math.random() * 2 - 1,
      technical: Math.random() * 2 - 1,
      value: Math.random() * 2 - 1,
      quality: Math.random() * 2 - 1,
      flow: Math.random() * 2 - 1,
      event: Math.random() * 2 - 1,
      per: Math.random() * 30 + 5,
      pbr: Math.random() * 3 + 0.3,
      roe: Math.random() * 20,
      passStatus: Math.random() > 0.3 ? 'pass' : 'fail',
      totalScore: Math.random() * 100,
      rankChange: Math.floor(Math.random() * 10) - 5,
      weight: Math.random() * 10,
      targetQty: Math.floor(Math.random() * 100),
      returnRate: Math.random() * 40 - 20,
      holdingDays: Math.floor(Math.random() * 30),
    }))

    // 단계별 필터링
    switch (activeStep) {
      case 'S3':
        return baseData.filter((item) => item.passStatus === 'pass').slice(0, 100)
      case 'S4':
        return baseData.slice(0, 50)
      case 'S5':
        return baseData.slice(0, 20)
      default:
        return baseData
    }
  }, [rankingData, activeStep])

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
            {STEP_TITLES[activeStep]}
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
              emptyMessage="조건에 맞는 종목이 없습니다."
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
