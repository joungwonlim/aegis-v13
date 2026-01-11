/**
 * ForecastTab - Forecast 예측 분석 탭
 * SSOT: modules/forecast/components/ForecastTab.tsx
 * 펼치기/접기 방식으로 필요할 때만 데이터 조회
 */

'use client'

import { useState, useEffect } from 'react'
import {
  TrendingUp,
  TrendingDown,
  Activity,
  AlertCircle,
  History,
  Calendar,
  Sparkles,
  Target,
  Loader2,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { LineChart, Line, XAxis, YAxis, ResponsiveContainer, ReferenceLine, Tooltip } from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/card'
import { Button } from '@/shared/components/ui/button'
import { Badge } from '@/shared/components/ui/badge'
import { useForecastAnalysis } from '../hooks/useForecastAnalysis'
import { useForecastEvents } from '../hooks/useForecastEvents'

interface ForecastTabProps {
  symbol: string
}

export function ForecastTab({ symbol }: ForecastTabProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [hasLoaded, setHasLoaded] = useState(false)

  const { data, isLoading, error, analyze } = useForecastAnalysis(symbol, { autoFetch: false })
  const { data: events, isLoading: eventsLoading, fetchEvents } = useForecastEvents(symbol, {
    autoFetch: false,
  })

  // 펼쳐질 때 한번만 데이터 조회
  useEffect(() => {
    if (isOpen && !hasLoaded) {
      analyze()
      fetchEvents()
      setHasLoaded(true)
    }
  }, [isOpen, hasLoaded, analyze, fetchEvents])

  const formatPercent = (n: number | null | undefined) => {
    if (n == null) return '-'
    const sign = n >= 0 ? '+' : ''
    return `${sign}${(n * 100).toFixed(2)}%`
  }

  const getFallbackLevelLabel = (level: string) => {
    switch (level) {
      case 'SYMBOL':
        return '종목 단독'
      case 'SECTOR':
        return '섹터 평균'
      case 'BUCKET':
        return '변동성 버킷'
      case 'MARKET':
        return '시장 전체'
      default:
        return level
    }
  }

  const getQualityBadge = (quality: string) => {
    switch (quality) {
      case 'HIGH':
        return { label: '높음', variant: 'default' as const }
      case 'MEDIUM':
        return { label: '중간', variant: 'secondary' as const }
      case 'LOW':
        return { label: '낮음', variant: 'outline' as const }
      case 'UNKNOWN':
        return { label: '불명', variant: 'outline' as const }
      default:
        return { label: quality, variant: 'outline' as const }
    }
  }

  return (
    <div className="space-y-4">
      {/* 펼치기/접기 헤더 */}
      <Card className="bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 border-blue-200 dark:border-blue-800">
        <CardHeader>
          <Button
            variant="ghost"
            className="w-full justify-between p-0 h-auto hover:bg-transparent"
            onClick={() => setIsOpen(!isOpen)}
          >
            <div className="flex items-center gap-2 text-blue-900 dark:text-blue-100">
              <Sparkles className="w-5 h-5" />
              <span className="text-base font-semibold">Forecast 예측 분석</span>
              {isLoading && <Loader2 className="w-4 h-4 animate-spin text-blue-600" />}
            </div>
            {isOpen ? (
              <ChevronUp className="w-5 h-5 text-blue-900 dark:text-blue-100" />
            ) : (
              <ChevronDown className="w-5 h-5 text-blue-900 dark:text-blue-100" />
            )}
          </Button>
        </CardHeader>
      </Card>

      {/* 펼쳐진 내용 */}
      {isOpen && (
        <div className="space-y-4">
          {/* 에러 표시 */}
          {error && (
            <Card className="bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800">
              <CardContent className="pt-6">
                <div className="flex items-start gap-2">
                  <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                  <div className="text-sm text-red-900 dark:text-red-100">
                    <p className="font-medium">분석 실패</p>
                    <p className="mt-1 text-red-700 dark:text-red-300">{error.message}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* 예측 결과 */}
          {data?.prediction && (
            <Card className="bg-gradient-to-br from-emerald-50 to-teal-50 dark:from-emerald-900/20 dark:to-teal-900/20 border-emerald-200 dark:border-emerald-800">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base flex items-center gap-2 text-emerald-900 dark:text-emerald-100">
                    <Target className="w-5 h-5" />
                    다음 주 예측 (5거래일)
                  </CardTitle>
                  <div className="flex items-center gap-2">
                    {data.event_detected && (
                      <Badge variant="default" className="bg-green-600">
                        {data.event_type === 'E1' ? '급등일 감지' : '갭+급등일 감지'}
                      </Badge>
                    )}
                    <Badge variant="secondary">
                      {data.prediction_type === 'EVENT_BASED' ? '이벤트 기반' : data.prediction_type === 'GENERAL' ? '일반 분석' : '예측 없음'}
                    </Badge>
                    <Badge variant={getQualityBadge(data.quality).variant}>
                      신뢰도: {getQualityBadge(data.quality).label}
                    </Badge>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-3">
                  {/* 기대 수익률 */}
                  <Card>
                    <CardContent className="pt-3">
                      <div className="flex items-center gap-1.5 mb-1">
                        <TrendingUp className="w-4 h-4 text-emerald-500" />
                        <span className="text-xs font-medium text-muted-foreground">기대 수익률</span>
                      </div>
                      <div
                        className={`text-lg font-bold font-mono ${
                          data.prediction.expected_ret_5d >= 0
                            ? 'text-red-600 dark:text-red-400'
                            : 'text-blue-600 dark:text-blue-400'
                        }`}
                      >
                        {formatPercent(data.prediction.expected_ret_5d)}
                      </div>
                    </CardContent>
                  </Card>

                  {/* 승률 */}
                  <Card>
                    <CardContent className="pt-3">
                      <div className="flex items-center gap-1.5 mb-1">
                        <Target className="w-4 h-4 text-emerald-500" />
                        <span className="text-xs font-medium text-muted-foreground">승률</span>
                      </div>
                      <div className="text-lg font-bold font-mono text-emerald-600 dark:text-emerald-400">
                        {(data.prediction.win_rate_5d * 100).toFixed(1)}%
                      </div>
                    </CardContent>
                  </Card>

                  {/* 최대 낙폭 */}
                  <Card>
                    <CardContent className="pt-3">
                      <div className="flex items-center gap-1.5 mb-1">
                        <TrendingDown className="w-4 h-4 text-blue-500" />
                        <span className="text-xs font-medium text-muted-foreground">최대 낙폭 (P10)</span>
                      </div>
                      <div className="text-lg font-bold font-mono text-blue-600 dark:text-blue-400">
                        {formatPercent(data.prediction.p10_mdd_5d)}
                      </div>
                    </CardContent>
                  </Card>

                  {/* 최대 상승 */}
                  <Card>
                    <CardContent className="pt-3">
                      <div className="flex items-center gap-1.5 mb-1">
                        <TrendingUp className="w-4 h-4 text-red-500" />
                        <span className="text-xs font-medium text-muted-foreground">최대 상승 (P90)</span>
                      </div>
                      <div className="text-lg font-bold font-mono text-red-600 dark:text-red-400">
                        {formatPercent(data.prediction.p90_runup_5d)}
                      </div>
                    </CardContent>
                  </Card>
                </div>

                {/* 분석 근거 */}
                <div className="mt-3 pt-3 border-t">
                  <p className="text-xs text-emerald-700 dark:text-emerald-300 mb-2 font-medium">
                    분석 근거
                  </p>
                  <div className="grid grid-cols-2 gap-2 text-xs">
                    <div className="text-muted-foreground">
                      <span className="font-medium">표본 크기: </span>
                      <span className="text-foreground font-mono">{data.sample_size}건</span>
                    </div>
                    <div className="text-muted-foreground">
                      <span className="font-medium">적용 범위: </span>
                      <span className="text-foreground">{getFallbackLevelLabel(data.fallback_level)}</span>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* 구분선 */}
          {data && <div className="border-t my-6" />}

          {/* 과거 이벤트 히스토리 */}
          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
              <History className="w-5 h-5" />
              과거 이벤트 히스토리
              {eventsLoading && <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />}
            </h3>

            {events && events.length > 0 && (
              <div className="space-y-3">
                {events.map((event) => (
                  <Card key={event.id} className="hover:border-foreground/50 transition-colors">
                    <CardContent className="pt-4">
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex items-center gap-2">
                          <Badge variant={event.event_type === 'E1_SURGE' ? 'default' : 'secondary'}>
                            {event.event_type === 'E1_SURGE' ? '급등일' : '갭+급등일'}
                          </Badge>
                          <span className="text-xs text-muted-foreground flex items-center gap-1">
                            <Calendar className="w-3 h-3" />
                            {new Date(event.trade_date).toLocaleDateString('ko-KR')}
                          </span>
                        </div>
                        <div className="text-xs font-semibold text-foreground font-mono">
                          당일: {formatPercent(event.ret)}
                        </div>
                      </div>

                      {/* 전방 성과 차트 */}
                      {event.fwd_ret_5d != null && (() => {
                        const chartData = [
                          { day: '1일', value: (event.fwd_ret_1d || 0) * 100 },
                          { day: '2일', value: (event.fwd_ret_2d || 0) * 100 },
                          { day: '3일', value: (event.fwd_ret_3d || 0) * 100 },
                          { day: '5일', value: (event.fwd_ret_5d || 0) * 100 },
                        ]

                        const allValues = chartData.map(d => d.value)
                        const minValue = Math.min(...allValues)
                        const maxValue = Math.max(...allValues)
                        const padding = Math.max(Math.abs(maxValue - minValue) * 0.2, 2)
                        const yMin = minValue - padding
                        const yMax = maxValue + padding

                        return (
                          <div className="space-y-3 mt-3">
                            {/* 수치 그리드 */}
                            <div className="grid grid-cols-4 gap-2 text-xs">
                              <div className="text-center">
                                <div className="text-muted-foreground mb-1">1일후</div>
                                <div className={`font-mono font-semibold ${
                                  chartData[0].value >= 0 ? 'text-red-600 dark:text-red-400' : 'text-blue-600 dark:text-blue-400'
                                }`}>
                                  {chartData[0].value.toFixed(2)}%
                                </div>
                              </div>
                              <div className="text-center">
                                <div className="text-muted-foreground mb-1">2일후</div>
                                <div className={`font-mono font-semibold ${
                                  chartData[1].value >= 0 ? 'text-red-600 dark:text-red-400' : 'text-blue-600 dark:text-blue-400'
                                }`}>
                                  {chartData[1].value.toFixed(2)}%
                                </div>
                              </div>
                              <div className="text-center">
                                <div className="text-muted-foreground mb-1">3일후</div>
                                <div className={`font-mono font-semibold ${
                                  chartData[2].value >= 0 ? 'text-red-600 dark:text-red-400' : 'text-blue-600 dark:text-blue-400'
                                }`}>
                                  {chartData[2].value.toFixed(2)}%
                                </div>
                              </div>
                              <div className="text-center">
                                <div className="text-muted-foreground mb-1">5일후</div>
                                <div className={`font-mono font-semibold ${
                                  chartData[3].value >= 0 ? 'text-red-600 dark:text-red-400' : 'text-blue-600 dark:text-blue-400'
                                }`}>
                                  {chartData[3].value.toFixed(2)}%
                                </div>
                              </div>
                            </div>

                            {/* 라인 차트 */}
                            <div>
                              <ResponsiveContainer width="100%" height={120}>
                                <LineChart data={chartData}>
                                  <XAxis
                                    dataKey="day"
                                    tick={{ fontSize: 11 }}
                                    stroke="hsl(var(--muted-foreground))"
                                  />
                                  <YAxis
                                    domain={[yMin, yMax]}
                                    tick={{ fontSize: 11 }}
                                    stroke="hsl(var(--muted-foreground))"
                                    tickFormatter={(val) => `${val.toFixed(1)}%`}
                                  />
                                  <Tooltip
                                    formatter={(value: number) => `${value.toFixed(2)}%`}
                                    contentStyle={{
                                      backgroundColor: 'hsl(var(--background))',
                                      border: '1px solid hsl(var(--border))',
                                      borderRadius: '6px',
                                    }}
                                  />
                                  <ReferenceLine
                                    y={0}
                                    stroke="hsl(var(--border))"
                                    strokeDasharray="3 3"
                                  />
                                  <Line
                                    type="monotone"
                                    dataKey="value"
                                    stroke="#3b82f6"
                                    strokeWidth={2.5}
                                    dot={{ fill: '#3b82f6', r: 4 }}
                                  />
                                </LineChart>
                              </ResponsiveContainer>
                            </div>

                            {/* 추가 지표 */}
                            <div className="grid grid-cols-3 gap-2 text-xs pt-2 border-t">
                              {event.max_drawdown_5d != null && (
                                <div className="text-center">
                                  <div className="text-muted-foreground mb-1">최대 낙폭</div>
                                  <div className="font-mono font-semibold text-blue-600 dark:text-blue-400">
                                    {(event.max_drawdown_5d * 100).toFixed(2)}%
                                  </div>
                                </div>
                              )}
                              {event.max_runup_5d != null && (
                                <div className="text-center">
                                  <div className="text-muted-foreground mb-1">최대 상승</div>
                                  <div className="font-mono font-semibold text-red-600 dark:text-red-400">
                                    {(event.max_runup_5d * 100).toFixed(2)}%
                                  </div>
                                </div>
                              )}
                              {event.gap_hold_3d != null && (
                                <div className="text-center">
                                  <div className="text-muted-foreground mb-1">갭 유지 (3일)</div>
                                  <Badge variant={event.gap_hold_3d ? 'default' : 'secondary'}>
                                    {event.gap_hold_3d ? '유지' : '붕괴'}
                                  </Badge>
                                </div>
                              )}
                            </div>
                          </div>
                        )
                      })()}
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}

            {events && events.length === 0 && (
              <Card className="bg-muted/50">
                <CardContent className="pt-6">
                  <div className="text-center py-8 text-muted-foreground text-sm">
                    <History className="w-10 h-10 mx-auto mb-2 opacity-30" />
                    <p>과거 이벤트가 없습니다</p>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
