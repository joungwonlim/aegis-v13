'use client'

/**
 * PipelineSteps - 퀀트 파이프라인 스텝 네비게이션
 * SSOT: S1-S5 단계 시각화 컴포넌트
 */

import { cn } from '@/shared/lib/utils'

export type PipelineStep = 'S1' | 'S2' | 'S3' | 'S4' | 'S5'

interface StepInfo {
  id: PipelineStep
  label: string
  title: string
  count?: number
  description: string
}

const STEPS: StepInfo[] = [
  {
    id: 'S1',
    label: 'S1',
    title: 'Universe',
    description: '투자 가능 종목',
  },
  {
    id: 'S2',
    label: 'S2',
    title: 'Signals',
    description: '팩터 계산',
  },
  {
    id: 'S3',
    label: 'S3',
    title: 'Screener',
    description: 'Hard Cut',
  },
  {
    id: 'S4',
    label: 'S4',
    title: 'Ranking',
    description: '종합 점수',
  },
  {
    id: 'S5',
    label: 'S5',
    title: 'Portfolio',
    description: '포트폴리오',
  },
]

interface PipelineStepsProps {
  activeStep: PipelineStep | null
  onStepClick: (step: PipelineStep) => void
  counts?: Partial<Record<PipelineStep, number>>
}

export function PipelineSteps({ activeStep, onStepClick, counts }: PipelineStepsProps) {
  return (
    <div className="w-full">
      {/* 스텝 인디케이터 */}
      <div className="flex items-center justify-between relative">
        {/* 연결선 */}
        <div className="absolute top-5 left-0 right-0 h-0.5 bg-border" />
        <div
          className="absolute top-5 left-0 h-0.5 bg-primary transition-all duration-300"
          style={{
            width: activeStep
              ? `${(STEPS.findIndex((s) => s.id === activeStep) / (STEPS.length - 1)) * 100}%`
              : '0%',
          }}
        />

        {STEPS.map((step, index) => {
          const isActive = activeStep === step.id
          const isPast =
            activeStep && STEPS.findIndex((s) => s.id === activeStep) > index
          const count = counts?.[step.id]

          return (
            <button
              key={step.id}
              onClick={() => onStepClick(step.id)}
              className={cn(
                'relative z-10 flex flex-col items-center gap-2 px-2 transition-all',
                'hover:scale-105 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary rounded-lg'
              )}
            >
              {/* 스텝 원 */}
              <div
                className={cn(
                  'w-10 h-10 rounded-full flex items-center justify-center text-sm font-bold transition-all',
                  'border-2',
                  isActive
                    ? 'bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/30'
                    : isPast
                      ? 'bg-primary/20 text-primary border-primary/50'
                      : 'bg-background text-muted-foreground border-border hover:border-primary/50'
                )}
              >
                {step.label}
              </div>

              {/* 스텝 정보 */}
              <div className="text-center">
                <div
                  className={cn(
                    'text-sm font-medium',
                    isActive ? 'text-foreground' : 'text-muted-foreground'
                  )}
                >
                  {step.title}
                </div>
                <div className="text-xs text-muted-foreground">{step.description}</div>
                {count !== undefined && (
                  <div
                    className={cn(
                      'text-xs font-mono mt-1',
                      isActive ? 'text-primary font-semibold' : 'text-muted-foreground'
                    )}
                  >
                    {count.toLocaleString()}종목
                  </div>
                )}
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
