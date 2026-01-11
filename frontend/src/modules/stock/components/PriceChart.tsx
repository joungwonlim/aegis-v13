'use client'

/**
 * PriceChart - Daily Chart (일봉 캔들스틱 차트)
 * SSOT: 종목 일봉 차트 표시
 */

import { useMemo, useState } from 'react'
import {
  ComposedChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
  ReferenceLine,
} from 'recharts'
import { Card, CardHeader, CardTitle, CardContent } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import type { DailyPrice } from '../types'

interface PriceChartProps {
  data: DailyPrice[]
  isLoading?: boolean
  avgBuyPrice?: number
}

type PeriodType = '1M' | '3M' | '6M' | '1Y'

const PERIOD_OPTIONS: { label: string; value: PeriodType; days: number }[] = [
  { label: '1개월', value: '1M', days: 30 },
  { label: '3개월', value: '3M', days: 90 },
  { label: '6개월', value: '6M', days: 180 },
  { label: '1년', value: '1Y', days: 365 },
]

// 색상 정의 (한국 주식시장: 빨강=상승, 파랑=하락)
const COLORS = {
  up: '#ef4444', // red-500
  down: '#3b82f6', // blue-500
  avgPrice: '#fbbf24', // amber-400
  crosshair: '#6b7280', // gray-500
}

// Custom Tooltip
function CustomTooltip({
  active,
  payload,
}: {
  active?: boolean
  payload?: Array<{ payload: DailyPrice & { displayDate: string } }>
}) {
  if (!active || !payload || !payload.length) return null

  const data = payload[0].payload
  const isUp = data.close >= data.open

  return (
    <div className="bg-popover border border-border rounded-lg p-3 shadow-lg">
      <p className="text-xs font-medium text-foreground mb-2">{data.date}</p>
      <div className="space-y-1">
        <div className="flex items-center justify-between gap-4 text-xs">
          <span className="text-muted-foreground">시가:</span>
          <span className="font-mono">{data.open.toLocaleString()}원</span>
        </div>
        <div className="flex items-center justify-between gap-4 text-xs">
          <span className="text-muted-foreground">고가:</span>
          <span className="font-mono" style={{ color: COLORS.up }}>
            {data.high.toLocaleString()}원
          </span>
        </div>
        <div className="flex items-center justify-between gap-4 text-xs">
          <span className="text-muted-foreground">저가:</span>
          <span className="font-mono" style={{ color: COLORS.down }}>
            {data.low.toLocaleString()}원
          </span>
        </div>
        <div className="flex items-center justify-between gap-4 text-xs">
          <span className="text-muted-foreground">종가:</span>
          <span
            className="font-mono"
            style={{ color: isUp ? COLORS.up : COLORS.down }}
          >
            {data.close.toLocaleString()}원
          </span>
        </div>
        <div className="flex items-center justify-between gap-4 text-xs">
          <span className="text-muted-foreground">거래량:</span>
          <span className="font-mono">{data.volume.toLocaleString()}</span>
        </div>
      </div>
    </div>
  )
}

export function PriceChart({ data, isLoading, avgBuyPrice }: PriceChartProps) {
  const [period, setPeriod] = useState<PeriodType>('3M')
  const [mouseY, setMouseY] = useState<number | null>(null)

  // 기간별 데이터 필터링
  const filteredData = useMemo(() => {
    const periodDays = PERIOD_OPTIONS.find((p) => p.value === period)?.days ?? 90
    return data.slice(-periodDays)
  }, [data, period])

  // 차트 데이터 포맷팅
  const chartData = useMemo(() => {
    if (filteredData.length === 0) return []
    return filteredData.map((d) => ({
      ...d,
      displayDate: d.date.substring(5), // MM-DD 형식
    }))
  }, [filteredData])

  // Y축 범위 계산
  const [yMin, yMax] = useMemo(() => {
    if (chartData.length === 0) return [0, 100]
    const allPrices = chartData.flatMap((d) => [d.high, d.low])
    // 평단가도 범위에 포함
    if (avgBuyPrice) {
      allPrices.push(avgBuyPrice)
    }
    const min = Math.min(...allPrices)
    const max = Math.max(...allPrices)
    const padding = (max - min) * 0.1
    return [min - padding, max + padding]
  }, [chartData, avgBuyPrice])

  // 차트 높이 설정
  const chartHeight = 300
  const marginTop = 10
  const marginBottom = 10
  const effectiveHeight = chartHeight - marginTop - marginBottom

  // 마우스 Y 좌표를 가격으로 변환
  const mousePrice = useMemo(() => {
    if (mouseY === null) return null
    const priceAtY = yMax - ((mouseY - marginTop) / effectiveHeight) * (yMax - yMin)
    return priceAtY
  }, [mouseY, yMin, yMax, effectiveHeight])

  if (isLoading) {
    return (
      <Card>
        <CardContent className="py-8">
          <div className="h-[300px] flex items-center justify-center">
            <div className="animate-pulse text-muted-foreground">차트 로딩중...</div>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (chartData.length === 0) {
    return (
      <Card>
        <CardContent className="py-8">
          <div className="h-[300px] flex items-center justify-center">
            <div className="text-muted-foreground">차트 데이터가 없습니다</div>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">Daily Chart</CardTitle>
          <div className="flex gap-1">
            {PERIOD_OPTIONS.map((opt) => (
              <Button
                key={opt.value}
                variant={period === opt.value ? 'default' : 'outline'}
                size="sm"
                className="h-7 px-2 text-xs"
                onClick={() => setPeriod(opt.value)}
              >
                {opt.label}
              </Button>
            ))}
          </div>
        </div>
      </CardHeader>
      <CardContent className="pb-4">
        {/* 캔들스틱 차트 */}
        <div className="h-[300px]">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart
              data={chartData}
              margin={{ top: marginTop, right: 60, left: 10, bottom: marginBottom }}
              onMouseMove={(e: { activeCoordinate?: { y: number } }) => {
                if (e && e.activeCoordinate) {
                  setMouseY(e.activeCoordinate.y)
                }
              }}
              onMouseLeave={() => setMouseY(null)}
            >
              <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
              <XAxis
                dataKey="displayDate"
                tick={{ fontSize: 11, className: 'fill-muted-foreground' }}
                tickLine={{ className: 'stroke-border' }}
                axisLine={{ className: 'stroke-border' }}
                interval="preserveStartEnd"
              />
              <YAxis
                domain={[yMin, yMax]}
                tick={{ fontSize: 11, className: 'fill-muted-foreground' }}
                tickLine={{ className: 'stroke-border' }}
                axisLine={{ className: 'stroke-border' }}
                tickFormatter={(v) => v.toLocaleString()}
                width={60}
              />
              <Tooltip
                content={<CustomTooltip />}
                cursor={{
                  stroke: COLORS.crosshair,
                  strokeWidth: 1,
                  strokeDasharray: '4 4',
                }}
              />

              {/* 마우스 가로선 (크로스헤어) */}
              {mousePrice !== null && mousePrice >= yMin && mousePrice <= yMax && (
                <ReferenceLine
                  y={mousePrice}
                  stroke={COLORS.crosshair}
                  strokeWidth={1}
                  strokeDasharray="4 4"
                  ifOverflow="extendDomain"
                  label={{
                    value: Math.round(mousePrice).toLocaleString(),
                    position: 'right',
                    fill: COLORS.crosshair,
                    fontSize: 11,
                    fontWeight: 'bold',
                  }}
                />
              )}

              {/* 평균매수가 표시 (보유종목일 경우) */}
              {avgBuyPrice && avgBuyPrice >= yMin && avgBuyPrice <= yMax && (
                <ReferenceLine
                  y={avgBuyPrice}
                  stroke={COLORS.avgPrice}
                  strokeWidth={2}
                  strokeDasharray="5 5"
                  label={{
                    value: `평단 ${avgBuyPrice.toLocaleString()}`,
                    position: 'right',
                    fill: COLORS.avgPrice,
                    fontSize: 11,
                    fontWeight: 'bold',
                  }}
                />
              )}

              {/* 캔들스틱 */}
              <Bar
                dataKey="close"
                shape={(rawProps: unknown) => {
                  const props = rawProps as {
                    x: number
                    width: number
                    payload: DailyPrice
                    index: number
                  }
                  const { x, width, payload } = props
                  const yScale = effectiveHeight / (yMax - yMin)

                  const getY = (price: number) => marginTop + (yMax - price) * yScale

                  const isUp = payload.close >= payload.open
                  const color = isUp ? COLORS.up : COLORS.down

                  const wickX = x + width / 2
                  const highY = getY(payload.high)
                  const lowY = getY(payload.low)
                  const openY = getY(payload.open)
                  const closeY = getY(payload.close)

                  const bodyTop = Math.min(openY, closeY)
                  const bodyHeight = Math.abs(closeY - openY) || 1

                  return (
                    <g key={`candle-${props.index}`}>
                      {/* 심지 */}
                      <line
                        x1={wickX}
                        x2={wickX}
                        y1={highY}
                        y2={lowY}
                        stroke={color}
                        strokeWidth={1}
                      />
                      {/* 몸통 */}
                      <rect
                        x={x + 1}
                        y={bodyTop}
                        width={Math.max(width - 2, 2)}
                        height={bodyHeight}
                        fill={isUp ? 'transparent' : color}
                        stroke={color}
                        strokeWidth={1}
                      />
                    </g>
                  )
                }}
              />
            </ComposedChart>
          </ResponsiveContainer>
        </div>

        {/* 거래량 차트 */}
        <div className="h-[80px] mt-2">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart
              data={chartData}
              margin={{ top: 0, right: 60, left: 10, bottom: 0 }}
            >
              <XAxis dataKey="displayDate" hide />
              <YAxis
                tick={{ fontSize: 10, className: 'fill-muted-foreground' }}
                tickFormatter={(v) => (v / 1000000).toFixed(0) + 'M'}
                width={60}
              />
              <Bar dataKey="volume">
                {chartData.map((entry, index) => (
                  <Cell
                    key={`cell-${index}`}
                    fill={entry.close >= entry.open ? COLORS.up : COLORS.down}
                    fillOpacity={0.5}
                  />
                ))}
              </Bar>
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  )
}
