# Documentation Synchronization Automation

**Purpose**: Keep code and documentation in sync when interfaces, modules, or schemas change.

**Trigger**:
- Interface signature changes in `internal/*/module.go`
- Database schema changes in migrations
- API endpoint changes in handlers
- User request: "update docs" or "sync documentation"

---

## Problem Statement

When developers modify code, documentation often gets stale:
1. ❌ Interface changes in `internal/brain/brain.go` but docs not updated
2. ❌ New DB column added but schema docs outdated
3. ❌ API endpoint signature changed but examples in docs still use old format

**Result**: Documentation lies, developers lose trust, onboarding becomes harder.

---

## Workflow

### Step 1: Detect Changes

Monitor these areas for changes:

#### 1.1 Module Interfaces
```bash
git diff HEAD~1 backend/internal/*/module.go | grep "^+.*interface\|^-.*interface"
```

#### 1.2 Database Schema
```bash
git diff --name-only HEAD~1 | grep "db/migrations/"
```

#### 1.3 API Endpoints
```bash
git diff HEAD~1 backend/internal/*/handler.go
```

---

### Step 2: Extract Information

For each detected change, extract:
- Interface name
- Method signatures (name, params, return types)
- Comments/documentation

---

### Step 3: Update Documentation

#### 3.1 Update Module Docs
- Find relevant section in docs
- Replace with extracted interface info
- Add AUTO-GENERATED comment for future reference
- Preserve any manually added sections

#### 3.2 Update Database Schema Docs
- Update table definitions
- Add new columns with descriptions

#### 3.3 Update API Docs
- Update endpoint signatures
- Update request/response examples

---

### Step 4: Verify Documentation

Ensure doc examples actually work:

```bash
# Validate links
grep -r "\[.*\](.*)" docs/ | grep -v "http"
```

---

## Aegis-Specific Rules

### Documentation Structure

```
docs/
├── guide/
│   ├── architecture/   # 시스템 구조
│   ├── backend/        # 백엔드 레이어
│   ├── frontend/       # 프론트엔드
│   └── database/       # DB 스키마
└── README.md
```

### Auto-Generated Sections

Mark these with HTML comments:

```markdown
<!-- AUTO-GENERATED SECTION - DO NOT EDIT MANUALLY -->
<!-- Generated: 2026-01-03 20:30:00 KST -->
<!-- Source: backend/internal/brain/brain.go:15-28 -->

[content]

<!-- END AUTO-GENERATED SECTION -->
```

### Manual Sections (Never Overwrite)

- **Overview**: High-level module description
- **Architecture**: Design decisions, diagrams
- **Best Practices**: Usage guidelines
- **Troubleshooting**: Common issues

---

## Usage Examples

### Example 1: Interface Change Detected

```
User: "I updated the Brain interface, can you sync the docs?"

Claude:
1. Detects change in internal/brain/brain.go
2. Extracts new interface signatures
3. Updates relevant docs
4. Reports:
   ✅ Updated docs/guide/backend/brain.md
   ✅ All examples compile
```

---

## Reporting

### Sync Report Format

```markdown
## Documentation Sync Report

**Date**: 2026-01-03 20:30:00 KST
**Trigger**: Interface change in `internal/brain/brain.go`

### Files Updated
1. ✅ `docs/guide/backend/brain.md`

### Files Requiring Manual Review
2. ⚠️ API docs may need example update

### Verification
- ✅ No broken internal links
```
