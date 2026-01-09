# 문서 자동 배포 규칙 (Docs Auto-Deploy)

**CRITICAL**: 문서 변경 시 GitHub Pages 자동 배포를 보장하는 필수 규칙입니다.

---

## 핵심 원칙

**"문서를 커밋하면, 반드시 배포도 트리거한다"**

---

## 자동 배포 체크리스트 (BLOCKER)

문서 변경 커밋 후 **반드시** 다음을 확인:

### ✅ Step 1: Workflow 존재 확인
```bash
ls .github/workflows/deploy-docs.yml
```

**없으면**: workflow 먼저 생성 (아래 템플릿 참고)

### ✅ Step 2: 배포 트리거 확인

문서 커밋 후:

```bash
# 최근 커밋 확인
git log -1 --name-only

# docs-site/** 변경이 있는지 확인
git diff HEAD~1 --name-only | grep "docs-site/"
```

**변경이 있으면**: ✅ Workflow 자동 트리거됨
**변경이 없으면**: ❌ 더미 커밋 필요

### ✅ Step 3: 배포 상태 확인 (30초 대기)

```bash
# GitHub Actions 상태 확인 (브라우저)
https://github.com/{org}/{repo}/actions
```

**Green ✓**: 배포 성공
**Yellow ⏳**: 진행 중 (2-3분 대기)
**Red ✗**: 실패 (로그 확인)

### ✅ Step 4: 배포 완료 확인 (3분 후)

```bash
# 실제 사이트 확인
https://{org}.github.io/{repo}/
```

**최신 내용 보임**: ✅ 배포 완료
**구 내용 보임**: ❌ 캐시 문제 (Ctrl+F5)

---

## 자동 배포가 안되는 경우

### Case 1: Workflow 없음

**증상**: 문서 커밋했는데 Actions 탭에 아무것도 안보임

**해결**:
```bash
# Workflow 파일 생성 (아래 템플릿)
cat > .github/workflows/deploy-docs.yml
```

### Case 2: Workflow 있지만 트리거 안됨

**증상**: Workflow는 있는데 최신 커밋에서 실행 안됨

**원인**: workflow 생성/수정 커밋에 `docs-site/**` 변경이 없음

**해결**: 더미 커밋으로 트리거
```bash
echo "<!-- Deploy trigger -->" >> docs-site/docs/guide/overview/development-schedule.md
git add docs-site/docs/guide/overview/development-schedule.md
git commit -m "docs(trigger): trigger deployment workflow"
git push origin main
```

### Case 3: GitHub Pages Source 설정 오류

**증상**: Workflow는 성공했는데 사이트 안보임

**해결**:
1. Repository Settings → Pages
2. **Source**: `GitHub Actions` 선택 (NOT "Deploy from a branch")
3. Save

---

## Workflow 템플릿

`.github/workflows/deploy-docs.yml`:

```yaml
name: Deploy Docs to GitHub Pages

on:
  push:
    branches:
      - main
    paths:
      - 'docs-site/**'
      - '.github/workflows/deploy-docs.yml'
  workflow_dispatch:  # ⭐ 수동 트리거 허용

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: docs-site

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: docs-site/package-lock.json

      - name: Install dependencies
        run: npm ci

      - name: Build website
        run: npm run build

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: docs-site/build

  deploy:
    needs: build
    runs-on: ubuntu-latest

    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}

    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

---

## 수동 트리거 방법

배포가 안되면 수동으로 트리거:

### 방법 1: GitHub UI (추천)
1. Repository → Actions 탭
2. "Deploy Docs to GitHub Pages" workflow 선택
3. "Run workflow" 버튼 클릭
4. Branch: `main` 선택
5. "Run workflow" 실행

### 방법 2: 더미 커밋
```bash
echo "<!-- Deploy trigger $(date +%s) -->" >> docs-site/docs/guide/overview/development-schedule.md
git add docs-site/docs/guide/overview/development-schedule.md
git commit -m "docs(trigger): manual deployment trigger"
git push origin main
```

---

## Claude Code 작업 흐름 (MANDATORY)

문서 작업 시 **반드시** 이 순서를 따름:

### 1. 문서 수정
```bash
# 문서 파일 수정
vim docs-site/docs/guide/**/*.md
```

### 2. 커밋 전 Workflow 확인
```bash
ls .github/workflows/deploy-docs.yml  # ✅ 있어야 함
```

**없으면**: Workflow 먼저 생성

### 3. 커밋 & 푸시
```bash
git add docs-site/
git commit -m "docs(...): ..."
git push origin main
```

### 4. 배포 트리거 확인 (30초 이내)
```bash
# Browser: https://github.com/{org}/{repo}/actions
# 최신 workflow run이 실행 중이어야 함
```

**실행 안됨**: 더미 커밋으로 트리거

### 5. 배포 완료 확인 (3분 이내)
```bash
# Browser: https://{org}.github.io/{repo}/
# 변경사항이 반영되어야 함
```

**반영 안됨**: 캐시 문제 → Ctrl+F5

### 6. 사용자에게 안내
```
✅ 문서가 업데이트되었습니다.
https://{org}.github.io/{repo}/

배포 완료까지 2-3분 소요됩니다.
```

---

## 금지 패턴

### ❌ 문서만 커밋하고 배포 확인 안함
```bash
git commit -m "docs: update"
git push
# (배포 확인 생략) ← 금지!
```

### ❌ Workflow 없는데 문서 커밋
```bash
# .github/workflows/deploy-docs.yml 없음
git commit -m "docs: add new page"
git push
# (배포 안됨) ← 금지!
```

### ❌ 배포 실패를 사용자에게 숨김
```
"문서 업데이트했습니다" (실제로는 배포 실패)
```

---

## 체크리스트 (문서 작업 완료 시)

- [ ] 문서 파일 수정됨
- [ ] `.github/workflows/deploy-docs.yml` 존재 확인
- [ ] 커밋 & 푸시 완료
- [ ] Actions 탭에서 workflow 실행 확인 (30초 이내)
- [ ] 배포 완료 확인 (3분 이내)
- [ ] 사용자에게 URL 안내

**하나라도 실패하면 사용자에게 알림**

---

## 참고 문서

- [GitHub Actions 문서](https://docs.github.com/en/actions)
- [GitHub Pages 문서](https://docs.github.com/en/pages)
- [Docusaurus 배포](https://docusaurus.io/docs/deployment)

---

**문서 버전**: v1.0
**최종 업데이트**: 2026-01-10
