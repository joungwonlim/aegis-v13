---
sidebar_position: 1
title: Introduction
description: Aegis v13 프로젝트 소개
---

# Introduction

> Aegis v13 - 기관급 퀀트 트레이딩 시스템

---

## What is Aegis?

Aegis는 **기관 투자자 수준의 퀀트 트레이딩 시스템**입니다.

### 핵심 특징

| 특징 | 설명 |
|------|------|
| **레이어 기반 아키텍처** | 기능별이 아닌 처리 단계별 구조 |
| **Contract 중심 설계** | 명확한 인터페이스로 모듈 교체 용이 |
| **Brain as Orchestrator** | Brain은 로직 없이 파이프라인만 조율 |
| **결정론적 스코어링** | AI 의존 최소화, 재현 가능한 결과 |

---

## Architecture Philosophy

### 기존 방식의 문제점

```
❌ 기능별 폴더 (brain/, fetcher/, execution/)
   → 책임 경계 불명확
   → 순환 참조 발생
   → 테스트 어려움
```

### v13 접근 방식

```
✅ 레이어별 폴더 (data/, signals/, selection/, portfolio/, execution/, audit/)
   → 데이터가 한 방향으로만 흐름
   → 각 레이어는 이전 레이어의 출력만 의존
   → 레이어 단위 테스트/교체 가능
```

---

## 7-Stage Pipeline

```
S0: Data Quality    → 원천 데이터 수집/검증
S1: Universe        → 투자 가능 종목 필터링
S2: Signals         → 팩터/이벤트 시그널 생성
S3: Screener        → 1차 필터링 (Hard Cut)
S4: Ranking         → 종합 점수 산출
S5: Portfolio       → 포트폴리오 구성/리밸런싱
S6: Execution       → 주문 생성/체결
S7: Audit           → 성과 분석/피드백
```

---

## On This Page

- [What is Aegis?](#what-is-aegis)
- [Architecture Philosophy](#architecture-philosophy)
- [7-Stage Pipeline](#7-stage-pipeline)

---

**Next**: [Tech Stack](./tech-stack.md)
