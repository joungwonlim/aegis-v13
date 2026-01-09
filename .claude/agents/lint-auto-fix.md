# Lint Auto-Fix Automation

**Purpose**: Automatically detect and fix lint errors in Go and TypeScript code.

**Trigger**:
- Before commit (pre-commit hook)
- On user request: "fix lint errors"
- When opening PR

---

## Workflow

### Step 1: Scan for Lint Errors

#### Backend (Go)
```bash
cd backend
make lint
```

#### Frontend (TypeScript)
```bash
cd frontend
pnpm lint
```

---

### Step 2: Categorize Errors

#### ✅ Auto-Fixable (Safe)
**Go**:
- Unused imports → `goimports -w .`
- Formatting → `gofmt -w .`
- Simple fixes → `golangci-lint run --fix`

**TypeScript**:
- Unused imports → `eslint --fix`
- Formatting → `prettier --write`
- Missing semicolons → `eslint --fix`

#### ⚠️ Manual Review Required
**Go**:
- Error handling issues
- Concurrency problems (race conditions)
- Security vulnerabilities

**TypeScript**:
- Type errors (`any` usage, missing types)
- React hooks violations
- Missing prop types

#### 🛑 Cannot Auto-Fix
**Go**:
- Logic errors
- Performance issues
- API design problems

**TypeScript**:
- Complex type inference issues
- State management problems

---

### Step 3: Apply Auto-Fixes

#### Backend Auto-Fix
```bash
cd backend
goimports -w .
gofmt -w .
golangci-lint run --fix
make lint
```

#### Frontend Auto-Fix
```bash
cd frontend
pnpm lint --fix
pnpm lint
```

---

### Step 4: Report Results

```markdown
## Lint Auto-Fix Report

**Scope**: [backend/frontend/all]
**Time**: [timestamp]

### Summary
- ✅ Auto-fixed: X errors
- ⚠️ Manual review: Y errors
- 🛑 Cannot fix: Z errors

### Auto-Fixed Errors (X)
1. Removed unused import in `internal/brain/stage1.go:5`
2. Fixed formatting in `pkg/logger/logger.go:45-67`

### Requires Manual Review (Y)
1. ⚠️ **Error handling**: Missing error check in `internal/execution/order.go:89`
```

---

## Aegis-Specific Rules

### Backend (Go)

#### Always Auto-Fix
- Unused imports in `internal/*`, `pkg/*`
- Formatting violations (gofmt)

#### Never Auto-Fix Without Review
- Changes to `internal/execution/` (money movement)
- Changes to `internal/brain/` scoring logic
- Error handling in API endpoints
- Concurrency primitives

### Frontend (TypeScript)

#### Always Auto-Fix
- Unused imports
- Missing semicolons
- Trailing commas

#### Never Auto-Fix Without Review
- Type assertions (`as` keyword)
- `any` type replacements
- React hooks order changes

---

## Safety Guarantees

### What Auto-Fix WILL Do
✅ Fix formatting (whitespace, indentation)
✅ Remove unused imports
✅ Apply simple linter suggestions

### What Auto-Fix WILL NOT Do
❌ Change business logic
❌ Modify function signatures
❌ Add/remove parameters
❌ Change concurrency patterns
❌ Alter error handling strategy
