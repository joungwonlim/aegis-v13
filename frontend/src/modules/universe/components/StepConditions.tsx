'use client'

/**
 * StepConditions - 단계별 조건 패널
 * SSOT: S1-S5 조건 표시 컴포넌트
 */

import { Card, CardContent, CardHeader, CardTitle } from '@/shared/components/ui/card'
import { Badge } from '@/shared/components/ui/badge'
import { ChevronUp, Check, X } from 'lucide-react'
import type { PipelineStep } from './PipelineSteps'

interface StepConditionsProps {
  step: PipelineStep
  onClose: () => void
}

export function StepConditions({ step, onClose }: StepConditionsProps) {
  const config = STEP_CONFIGS[step]

  return (
    <Card className="mt-4 animate-in slide-in-from-top-2 duration-200">
      <CardHeader className="pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Badge variant="outline" className="font-mono">
              {step}
            </Badge>
            <CardTitle className="text-lg">{config.title}</CardTitle>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-md hover:bg-muted transition-colors"
            aria-label="접기"
          >
            <ChevronUp className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>
        <p className="text-sm text-muted-foreground">{config.description}</p>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {config.conditions.map((condition, index) => (
            <ConditionItem key={index} {...condition} />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

interface ConditionItemProps {
  label: string
  value: string
  type: 'include' | 'exclude' | 'info'
}

function ConditionItem({ label, value, type }: ConditionItemProps) {
  return (
    <div className="flex items-center gap-3 py-2 px-3 rounded-lg bg-muted/50">
      {type === 'include' && (
        <div className="w-5 h-5 rounded-full bg-green-500/20 flex items-center justify-center">
          <Check className="w-3 h-3 text-green-600" />
        </div>
      )}
      {type === 'exclude' && (
        <div className="w-5 h-5 rounded-full bg-red-500/20 flex items-center justify-center">
          <X className="w-3 h-3 text-red-600" />
        </div>
      )}
      {type === 'info' && (
        <div className="w-5 h-5 rounded-full bg-blue-500/20 flex items-center justify-center">
          <span className="text-xs text-blue-600 font-bold">i</span>
        </div>
      )}
      <span className="text-sm flex-1">{label}</span>
      <span className="text-sm font-mono text-muted-foreground">{value}</span>
    </div>
  )
}

// 단계별 설정
interface StepConfig {
  title: string
  description: string
  conditions: ConditionItemProps[]
}

const STEP_CONFIGS: Record<PipelineStep, StepConfig> = {
  S1: {
    title: 'Universe 조건',
    description: '투자 가능한 종목 풀을 정의하는 기본 필터입니다.',
    conditions: [
      { label: '시가총액', value: '≥ 1,000억', type: 'include' },
      { label: '일평균 거래대금', value: '≥ 10억', type: 'include' },
      { label: '상장일', value: '≥ 1년', type: 'include' },
      { label: '관리종목', value: '제외', type: 'exclude' },
      { label: '거래정지', value: '제외', type: 'exclude' },
      { label: '스팩/리츠', value: '제외', type: 'exclude' },
    ],
  },
  S2: {
    title: 'Signals (팩터)',
    description: '종목별로 6가지 팩터 점수를 계산합니다.',
    conditions: [
      { label: 'Momentum', value: '수익률 모멘텀', type: 'info' },
      { label: 'Technical', value: 'RSI, MACD 등', type: 'info' },
      { label: 'Value', value: 'PER, PBR, PSR', type: 'info' },
      { label: 'Quality', value: 'ROE, 부채비율', type: 'info' },
      { label: 'Flow', value: '외국인/기관 수급', type: 'info' },
      { label: 'Event', value: '공시/이벤트', type: 'info' },
    ],
  },
  S3: {
    title: 'Screener (Hard Cut)',
    description: '명확히 부적합한 종목을 제거하는 필터입니다.',
    conditions: [
      { label: '모멘텀 점수', value: '≥ 0 (중립 이상)', type: 'include' },
      { label: 'PER', value: '≤ 50', type: 'include' },
      { label: 'PBR', value: '≥ 0.2', type: 'include' },
      { label: 'ROE', value: '≥ 5%', type: 'include' },
      { label: '부채비율', value: '≤ 200%', type: 'include' },
      { label: '적자 기업', value: '제외', type: 'exclude' },
    ],
  },
  S4: {
    title: 'Ranking (가중치)',
    description: '팩터별 가중치를 적용하여 종합 점수를 계산합니다.',
    conditions: [
      { label: 'Momentum', value: '25%', type: 'info' },
      { label: 'Technical', value: '15%', type: 'info' },
      { label: 'Value', value: '20%', type: 'info' },
      { label: 'Quality', value: '15%', type: 'info' },
      { label: 'Flow', value: '15%', type: 'info' },
      { label: 'Event', value: '10%', type: 'info' },
    ],
  },
  S5: {
    title: 'Portfolio 규칙',
    description: '포트폴리오 구성 및 비중 조절 규칙입니다.',
    conditions: [
      { label: '최대 종목 수', value: '20종목', type: 'info' },
      { label: '종목당 최대 비중', value: '10%', type: 'include' },
      { label: '섹터당 최대 비중', value: '30%', type: 'include' },
      { label: '최소 투자 금액', value: '100만원', type: 'include' },
      { label: '리밸런싱 주기', value: '월 1회', type: 'info' },
      { label: '손절 기준', value: '-15%', type: 'exclude' },
    ],
  },
}
