# Aegis v13 Documentation

> 기관급 퀀트 트레이딩 시스템

---

## Guide

### Overview
- [Introduction](./guide/overview/introduction.md) - 프로젝트 소개
- [Tech Stack](./guide/overview/tech-stack.md) - 기술 스택
- [Getting Started](./guide/overview/getting-started.md) - 시작하기

### Architecture
- [System Overview](./guide/architecture/system-overview.md) - 시스템 전체 구조
- [Data Flow](./guide/architecture/data-flow.md) - 데이터 흐름 (7단계 파이프라인)
- [Contracts](./guide/architecture/contracts.md) - 핵심 계약/인터페이스

### Backend (Go)
- [Folder Structure](./guide/backend/folder-structure.md) - 폴더 구조
- [Data Layer](./guide/backend/data-layer.md) - 데이터 수집/정제
- [Signals Layer](./guide/backend/signals-layer.md) - 시그널 생성
- [Selection Layer](./guide/backend/selection-layer.md) - 스크리닝/랭킹
- [Portfolio Layer](./guide/backend/portfolio-layer.md) - 포트폴리오 구성
- [Execution Layer](./guide/backend/execution-layer.md) - 주문 실행
- [Audit Layer](./guide/backend/audit-layer.md) - 성과 분석

### Frontend (Next.js)
- [Folder Structure](./guide/frontend/folder-structure.md) - 폴더 구조
- [Components](./guide/frontend/components.md) - 컴포넌트 구조

### Database (PostgreSQL)
- [Schema Design](./guide/database/schema-design.md) - 스키마 설계
- [Migrations](./guide/database/migrations.md) - 마이그레이션 가이드

---

## Quick Links

| 영역 | 설명 |
|------|------|
| `/backend` | Go BFF 서버 |
| `/frontend` | Next.js 클라이언트 |
| `/docs` | 이 문서 |
