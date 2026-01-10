'use client'

/**
 * InvestorTradingChart - 투자자별 매매동향 차트
 * SSOT: 외국인/기관/개인 순매수 차트
 */

import { useMemo, useState } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import type { InvestorTrading } from '../types'

interface InvestorTradingChartProps {
  data: InvestorTrading[]
  isLoading?: boolean
}

type PeriodType = 'ALL' | '1M' | '3M' | '6M' | '1Y'

const PERIODS: { label: string; value: PeriodType; days: number }[] = [
  { label: '전체', value: 'ALL', days: 9999 },
  { label: '1개월', value: '1M', days: 30 },
  { label: '3개월', value: '3M', days: 90 },
  { label: '6개월', value: '6M', days: 180 },
  { label: '1년', value: '1Y', days: 365 },
]

// 색상 정의
const COLORS = {
  foreign: '#F04452', // 외국인 - 빨강
  inst: '#7B61FF', // 기관 - 보라
  indiv: '#F2A93B', // 개인 - 주황
}

// 숫자 포맷 함수
function formatAxisValue(value: number): string {
  const absValue = Math.abs(value)
  if (absValue >= 100000000) {
    return (value / 100000000).toFixed(1) + '억'
  } else if (absValue >= 10000) {
    return (value / 10000).toFixed(0) + '만'
  } else if (absValue >= 1000) {
    return (value / 1000).toFixed(1) + '천'
  }
  return value.toLocaleString()
}

function formatVolume(volume: number): string {
  const absVolume = Math.abs(volume)
  const sign = volume < 0 ? '-' : ''

  if (absVolume >= 100000000) {
    return sign + (absVolume / 100000000).toFixed(2) + '억'
  } else if (absVolume >= 10000) {
    return sign + Math.round(absVolume / 10000).toLocaleString() + '만'
  } else if (absVolume >= 1000) {
    return sign + (absVolume / 1000).toFixed(1) + '천'
  }
  return volume.toLocaleString()
}

export function InvestorTradingChart({ data, isLoading }: InvestorTradingChartProps) {
  const [period, setPeriod] = useState<PeriodType>('ALL')

  // 사용 가능한 데이터 일수
  const availableDays = data.length

  // 기간별 데이터 필터링
  const filteredData = useMemo(() => {
    if (period === 'ALL') return data
    const periodConfig = PERIODS.find((p) => p.value === period)
    const days = periodConfig?.days ?? 30
    return data.slice(-days)
  }, [data, period])

  // 기간 버튼 활성화 여부 (해당 기간의 80% 이상 데이터가 있어야 활성화)
  const isPeriodAvailable = (p: PeriodType): boolean => {
    if (p === 'ALL') return true
    const periodConfig = PERIODS.find((pc) => pc.value === p)
    if (!periodConfig) return false
    return availableDays >= periodConfig.days * 0.8
  }

  // 차트 데이터 포맷팅
  const chartData = useMemo(() => {
    return filteredData.map((d) => ({
      ...d,
      displayDate: d.date.substring(5).replace('-', '. ') + '.',
      foreign: d.foreign_net,
      inst: d.inst_net,
      indiv: d.indiv_net,
    }))
  }, [filteredData])

  // 날짜 범위
  const dateRange = useMemo(() => {
    if (chartData.length === 0) return ''
    const first = chartData[0]?.date ?? ''
    const last = chartData[chartData.length - 1]?.date ?? ''
    const formatDate = (d: string) => d.replace(/-/g, '. ') + '.'
    return `${formatDate(first)} - ${formatDate(last)} 기준`
  }, [chartData])

  if (isLoading) {
    return (
      <Card>
        <CardContent className="py-8">
          <div className="h-[250px] flex items-center justify-center">
            <div className="animate-pulse text-muted-foreground">로딩중...</div>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (chartData.length === 0) {
    return (
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">투자자별 매매동향</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-[250px] flex items-center justify-center">
            <div className="text-muted-foreground">데이터가 없습니다</div>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">투자자별 매매동향</CardTitle>
          <div className="flex gap-1">
            {PERIODS.map((p) => {
              const available = isPeriodAvailable(p.value)
              return (
                <Button
                  key={p.value}
                  variant={period === p.value ? 'default' : 'outline'}
                  size="sm"
                  className="h-7 px-2 text-xs"
                  onClick={() => setPeriod(p.value)}
                  disabled={!available}
                  title={!available ? '데이터 부족' : undefined}
                >
                  {p.label}
                </Button>
              )
            })}
          </div>
        </div>
        <p className="text-xs text-muted-foreground">
          {dateRange} ({availableDays}일)
        </p>
      </CardHeader>
      <CardContent className="pb-4">
        {/* 범례 */}
        <div className="flex items-center justify-end gap-4 mb-2">
          <div className="flex items-center gap-1.5">
            <div className="w-3 h-0.5 rounded" style={{ backgroundColor: COLORS.foreign }} />
            <span className="text-xs text-muted-foreground">외국인</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-3 h-0.5 rounded" style={{ backgroundColor: COLORS.inst }} />
            <span className="text-xs text-muted-foreground">기관</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-3 h-0.5 rounded" style={{ backgroundColor: COLORS.indiv }} />
            <span className="text-xs text-muted-foreground">개인</span>
          </div>
          <span className="text-xs text-muted-foreground">(주)</span>
        </div>

        {/* 차트 */}
        <div className="h-[200px]">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData} margin={{ top: 5, right: 50, left: 10, bottom: 5 }}>
              <CartesianGrid strokeDasharray="1 1" className="stroke-border" />
              <XAxis
                dataKey="displayDate"
                tick={{ fontSize: 11, className: 'fill-muted-foreground' }}
                tickLine={false}
                axisLine={{ className: 'stroke-border' }}
                interval="preserveStartEnd"
                minTickGap={50}
              />
              <YAxis
                tick={{ fontSize: 11, className: 'fill-muted-foreground' }}
                tickLine={false}
                axisLine={false}
                tickFormatter={(v) => formatAxisValue(v)}
                orientation="right"
                width={55}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'hsl(var(--popover))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '8px',
                  fontSize: '12px',
                }}
                labelStyle={{ color: 'hsl(var(--foreground))', marginBottom: '4px' }}
                formatter={(value, name) => {
                  const labels: Record<string, string> = {
                    foreign: '외국인',
                    inst: '기관',
                    indiv: '개인',
                  }
                  const numValue = Number(value) || 0
                  const strName = String(name || '')
                  const color = numValue >= 0 ? 'hsl(var(--chart-1))' : 'hsl(var(--chart-2))'
                  return [
                    <span key={strName} style={{ color }}>
                      {formatVolume(numValue)}주
                    </span>,
                    labels[strName] || strName,
                  ]
                }}
              />
              <ReferenceLine y={0} className="stroke-border" strokeWidth={1} />
              <Line
                type="monotone"
                dataKey="foreign"
                stroke={COLORS.foreign}
                strokeWidth={1.5}
                dot={false}
                activeDot={{ r: 4, fill: COLORS.foreign }}
              />
              <Line
                type="monotone"
                dataKey="inst"
                stroke={COLORS.inst}
                strokeWidth={1.5}
                dot={false}
                activeDot={{ r: 4, fill: COLORS.inst }}
              />
              <Line
                type="monotone"
                dataKey="indiv"
                stroke={COLORS.indiv}
                strokeWidth={1.5}
                dot={false}
                activeDot={{ r: 4, fill: COLORS.indiv }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* 최근 데이터 테이블 */}
        <div className="mt-4 border-t overflow-x-auto">
          <table className="w-full text-xs min-w-[500px]">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="py-2 px-2 text-left font-medium text-muted-foreground">날짜</th>
                <th className="py-2 px-2 text-right font-medium text-muted-foreground">종가</th>
                <th className="py-2 px-2 text-right font-medium text-muted-foreground">등락률</th>
                <th className="py-2 px-2 text-right font-medium text-muted-foreground">외국인</th>
                <th className="py-2 px-2 text-right font-medium text-muted-foreground">기관</th>
                <th className="py-2 px-2 text-right font-medium text-muted-foreground">개인</th>
              </tr>
            </thead>
            <tbody>
              {chartData
                .slice()
                .reverse()
                .slice(0, 5)
                .map((row, idx) => (
                  <tr key={idx} className="border-b last:border-b-0 hover:bg-muted/30">
                    <td className="py-2 px-2">{row.date.replace(/-/g, '. ')}</td>
                    <td className="py-2 px-2 text-right font-mono">
                      {row.close_price > 0 ? row.close_price.toLocaleString() : '-'}
                    </td>
                    <td
                      className={`py-2 px-2 text-right font-mono ${
                        row.change_rate >= 0 ? 'text-red-500' : 'text-blue-500'
                      }`}
                    >
                      {row.change_rate !== 0
                        ? `${row.change_rate > 0 ? '+' : ''}${row.change_rate.toFixed(2)}%`
                        : '-'}
                    </td>
                    <td
                      className={`py-2 px-2 text-right font-mono ${
                        row.foreign >= 0 ? 'text-red-500' : 'text-blue-500'
                      }`}
                    >
                      {row.foreign >= 0 ? '+' : ''}
                      {formatVolume(row.foreign)}
                    </td>
                    <td
                      className={`py-2 px-2 text-right font-mono ${
                        row.inst >= 0 ? 'text-red-500' : 'text-blue-500'
                      }`}
                    >
                      {row.inst >= 0 ? '+' : ''}
                      {formatVolume(row.inst)}
                    </td>
                    <td
                      className={`py-2 px-2 text-right font-mono ${
                        row.indiv >= 0 ? 'text-red-500' : 'text-blue-500'
                      }`}
                    >
                      {row.indiv >= 0 ? '+' : ''}
                      {formatVolume(row.indiv)}
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}
