'use client'

/**
 * 랭킹 (종합 점수 기반) 페이지
 */

import { useState } from 'react'
import { Card, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Badge } from '@/shared/components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/shared/components/ui/tabs'
import {
  StockDataTable,
  type StockDataItem,
  type StockDataColumn,
} from '@/modules/stock/components'
import { RefreshCw, TrendingUp, TrendingDown, Trophy } from 'lucide-react'

// 데모 데이터 - 랭킹 종목 (점수 순 정렬)
const DEMO_RANKING: StockDataItem[] = [
  { code: '005930', name: '삼성전자', price: 72300, change: 1700, changeRate: 2.41, score: 92.5, rank: 1 },
  { code: '000660', name: 'SK하이닉스', price: 178500, change: -2500, changeRate: -1.38, score: 89.3, rank: 2 },
  { code: '035420', name: 'NAVER', price: 215000, change: 3000, changeRate: 1.41, score: 87.8, rank: 3 },
  { code: '051910', name: 'LG화학', price: 428000, change: 8000, changeRate: 1.90, score: 85.6, rank: 4 },
  { code: '006400', name: '삼성SDI', price: 385000, change: 0, changeRate: 0, score: 84.1, rank: 5 },
  { code: '068270', name: '셀트리온', price: 178000, change: 2000, changeRate: 1.14, score: 82.5, rank: 6 },
  { code: '035720', name: '카카오', price: 52800, change: -800, changeRate: -1.49, score: 80.2, rank: 7 },
  { code: '105560', name: 'KB금융', price: 78900, change: 1200, changeRate: 1.54, score: 78.9, rank: 8 },
  { code: '055550', name: '신한지주', price: 52100, change: 500, changeRate: 0.97, score: 77.8, rank: 9 },
  { code: '012330', name: '현대모비스', price: 235000, change: 4000, changeRate: 1.73, score: 76.1, rank: 10 },
]

// 랭킹용 추가 컬럼 정의 (기본 컬럼: 순번, 종목명, 현재가, 전일대비)
const rankingExtraColumns: StockDataColumn[] = [
  {
    key: 'score',
    label: '종합점수',
    width: 'w-24',
    align: 'right',
    render: (item) => {
      const score = item.score ?? 0
      return (
        <Badge
          variant="secondary"
          className={`font-mono ${
            score >= 85
              ? 'bg-positive/10 text-positive'
              : score >= 70
              ? 'bg-yellow-500/10 text-yellow-600'
              : 'bg-muted'
          }`}
        >
          {score.toFixed(1)}
        </Badge>
      )
    },
  },
]

export default function RankingPage() {
  const [data] = useState<StockDataItem[]>(DEMO_RANKING)

  // 상승/하락 종목 필터
  const upStocks = data.filter((d) => (d.change ?? 0) > 0)
  const downStocks = data.filter((d) => (d.change ?? 0) < 0)

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">종합 랭킹</h1>
          <Badge>TOP {data.length}</Badge>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="icon">
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* 요약 통계 */}
      <div className="grid grid-cols-3 gap-4">
        <Card>
          <CardContent className="pt-4">
            <div className="flex items-center gap-2">
              <Trophy className="h-5 w-5 text-yellow-500" />
              <div>
                <p className="text-sm text-muted-foreground">1위</p>
                <p className="text-lg font-bold">{data[0]?.name}</p>
                <p className="text-sm font-mono text-muted-foreground">
                  {data[0]?.score?.toFixed(1)}점
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <div className="flex items-center gap-2">
              <TrendingUp className="h-5 w-5 text-positive" />
              <div>
                <p className="text-sm text-muted-foreground">상승 종목</p>
                <p className="text-2xl font-bold font-mono text-positive">
                  {upStocks.length}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <div className="flex items-center gap-2">
              <TrendingDown className="h-5 w-5 text-negative" />
              <div>
                <p className="text-sm text-muted-foreground">하락 종목</p>
                <p className="text-2xl font-bold font-mono text-negative">
                  {downStocks.length}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 랭킹 테이블 */}
      <Card>
        <CardContent className="p-0">
          <Tabs defaultValue="all" className="w-full">
            <div className="border-b px-4">
              <TabsList className="bg-transparent h-12">
                <TabsTrigger value="all">전체 ({data.length})</TabsTrigger>
                <TabsTrigger value="up">
                  <TrendingUp className="h-3 w-3 mr-1 text-positive" />
                  상승 ({upStocks.length})
                </TabsTrigger>
                <TabsTrigger value="down">
                  <TrendingDown className="h-3 w-3 mr-1 text-negative" />
                  하락 ({downStocks.length})
                </TabsTrigger>
              </TabsList>
            </div>
            <TabsContent value="all" className="mt-0">
              <StockDataTable
                data={data}
                extraColumns={rankingExtraColumns}
                emptyMessage="랭킹 데이터가 없습니다."
              />
            </TabsContent>
            <TabsContent value="up" className="mt-0">
              <StockDataTable
                data={upStocks}
                extraColumns={rankingExtraColumns}
                emptyMessage="상승 종목이 없습니다."
              />
            </TabsContent>
            <TabsContent value="down" className="mt-0">
              <StockDataTable
                data={downStocks}
                extraColumns={rankingExtraColumns}
                emptyMessage="하락 종목이 없습니다."
              />
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  )
}
