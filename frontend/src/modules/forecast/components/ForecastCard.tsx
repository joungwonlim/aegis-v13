/**
 * ForecastCard - Forecast 예측 정보 카드
 * SSOT: modules/forecast/components/ForecastCard.tsx
 */

import { TrendingUp, TrendingDown, Activity } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/card'
import { Badge } from '@/shared/components/ui/badge'
import { cn } from '@/shared/lib/utils'
import type { ForecastPrediction } from '../types'

interface ForecastCardProps {
  prediction: ForecastPrediction | null
  isLoading?: boolean
}

export function ForecastCard({ prediction, isLoading }: ForecastCardProps) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Activity className="w-4 h-4" />
            이벤트 예측
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="animate-pulse space-y-2">
            <div className="h-4 bg-muted rounded w-3/4" />
            <div className="h-4 bg-muted rounded w-1/2" />
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!prediction) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Activity className="w-4 h-4" />
            이벤트 예측
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            최근 이벤트가 감지되지 않았습니다.
          </p>
        </CardContent>
      </Card>
    )
  }

  const {
    event_type,
    expected_return_1d,
    expected_return_5d,
    win_rate_1d,
    win_rate_5d,
    p10_mdd,
    confidence,
    sample_count,
    level,
  } = prediction

  const isPositive1D = expected_return_1d >= 0
  const isPositive5D = expected_return_5d >= 0

  // 이벤트 타입 한글 변환
  const eventTypeLabel = event_type === 'E1_SURGE' ? '급등' : '갭+급등'

  // 레벨 한글 변환
  const levelLabels: Record<string, string> = {
    SYMBOL: '종목',
    SECTOR: '섹터',
    BUCKET: '시총구간',
    MARKET: '시장',
  }
  const levelLabel = levelLabels[level] || level

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          <Activity className="w-4 h-4" />
          이벤트 예측
          <Badge variant="outline" className="ml-auto text-xs">
            {eventTypeLabel}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* 기대 수익률 */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <div className="text-xs text-muted-foreground mb-1">1일 후 예상</div>
            <div
              className={cn(
                'text-lg font-bold font-mono flex items-center gap-1',
                isPositive1D ? 'text-red-500' : 'text-blue-500'
              )}
            >
              {isPositive1D ? (
                <TrendingUp className="w-4 h-4" />
              ) : (
                <TrendingDown className="w-4 h-4" />
              )}
              {isPositive1D ? '+' : ''}
              {(expected_return_1d * 100).toFixed(2)}%
            </div>
            <div className="text-xs text-muted-foreground mt-1">
              승률: {(win_rate_1d * 100).toFixed(1)}%
            </div>
          </div>

          <div>
            <div className="text-xs text-muted-foreground mb-1">5일 후 예상</div>
            <div
              className={cn(
                'text-lg font-bold font-mono flex items-center gap-1',
                isPositive5D ? 'text-red-500' : 'text-blue-500'
              )}
            >
              {isPositive5D ? (
                <TrendingUp className="w-4 h-4" />
              ) : (
                <TrendingDown className="w-4 h-4" />
              )}
              {isPositive5D ? '+' : ''}
              {(expected_return_5d * 100).toFixed(2)}%
            </div>
            <div className="text-xs text-muted-foreground mt-1">
              승률: {(win_rate_5d * 100).toFixed(1)}%
            </div>
          </div>
        </div>

        {/* 리스크 지표 */}
        <div className="border-t pt-3 space-y-2">
          <div className="flex justify-between text-sm">
            <span className="text-muted-foreground">최악 MDD (P10)</span>
            <span className="font-mono text-destructive">
              {(p10_mdd * 100).toFixed(2)}%
            </span>
          </div>

          <div className="flex justify-between text-sm">
            <span className="text-muted-foreground">신뢰도</span>
            <span className="font-mono">
              {(confidence * 100).toFixed(0)}%
            </span>
          </div>

          <div className="flex justify-between text-sm">
            <span className="text-muted-foreground">샘플 수</span>
            <span className="font-mono">{sample_count}건</span>
          </div>

          <div className="flex justify-between text-sm">
            <span className="text-muted-foreground">기준 레벨</span>
            <Badge variant="secondary" className="text-xs">
              {levelLabel}
            </Badge>
          </div>
        </div>

        {/* 경고 메시지 */}
        {confidence < 0.5 && (
          <div className="text-xs text-amber-600 bg-amber-50 dark:bg-amber-950 p-2 rounded">
            ⚠️ 샘플 수가 적어 신뢰도가 낮습니다. 참고용으로만 활용하세요.
          </div>
        )}
      </CardContent>
    </Card>
  )
}
