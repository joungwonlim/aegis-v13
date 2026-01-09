# /suggest-tidyings

Suggest safe tidyings for the given scope in the Aegis codebase.

---

## Command Signature

```
/suggest-tidyings [scope] [--intent=<intent>] [--limit=<n>]
```

### Parameters

- **scope** (required): Directory or file path(s) to analyze
  - Examples: `internal/brain`, `backend/pkg`, `frontend/src/app`
  - Special: `.` for entire repository (not recommended, too broad)

- **--intent** (optional): Focus area for tidying suggestions
  - `readability` (default): Naming, formatting, comment clarity
  - `maintainability`: Deduplication, extraction, structural improvements
  - `lint`: Auto-fixable lint errors and type issues
  - `testability`: Making code easier to test

- **--limit** (optional): Maximum number of suggestions
  - Default: 10
  - Range: 1-20

---

## Usage Examples

### Example 1: Suggest tidyings for brain module
```
/suggest-tidyings internal/brain
```
**Output**: 10 tidying suggestions focused on readability

### Example 2: Focus on testability in execution module
```
/suggest-tidyings internal/execution --intent=testability
```

### Example 3: Quick lint fixes
```
/suggest-tidyings internal/selection --intent=lint --limit=5
```

---

## Output Format

```markdown
## Tidying Suggestions for [Scope]

**Intent**: [readability/maintainability/lint/testability]
**Files scanned**: X
**Suggestions**: Y (showing top [limit])

---

### Low Risk (Safe to apply immediately)
- [ ] **Extract magic number to constant**
  - Location: `internal/brain/stage1.go:78`
  - Change: Move `0.75` to `const MinConfidenceScore = 0.75`
  - Verify: `make test`

### Medium Risk (Review recommended before applying)
- [ ] **Extract 80-line function into helpers**
  - Location: `internal/brain/stage2.go:120-200`
  - Risk: Need to verify extracted functions don't change behavior
  - Verify: `make test`

### High Risk (Requires explicit approval)
- [ ] **Reorder struct fields for memory optimization**
  - Risk: Breaks if struct is serialized with reflection

### Potential Bugs (NOT tidying - flag separately)
⚠️ **Unused error return**
  - Location: `internal/data/fetcher.go:123`
  - Issue: Silently ignores errors

---

**Next Steps**:
1. Review suggestions and select items to apply
2. Say "apply all low-risk" to proceed automatically
3. Or select specific items: "apply item 1 and 3"
```

---

## Rules & Constraints

### 1. Respect `.claude/rules/tidy-rule.md`
- Obey constraints defined in tidy-rule.md
- Never suggest touching forbidden areas:
  - `internal/execution/` (critical path)
  - `db/migrations/` (irreversible)

### 2. No Behavior Changes
- All suggestions MUST preserve existing behavior

### 3. Verification Commands Required
- Every suggestion MUST include exact command to verify

### 4. Risk Assessment Required
- Low: Mechanical change (rename, extract constant)
- Medium: Structural change (extract function, reorder code)
- High: Cross-cutting change (affects multiple modules)

---

## Error Handling

### If No Tidyings Found
```markdown
## Tidying Suggestions for [Scope]

**Result**: No tidyings suggested ✨
**Conclusion**: This code is already tidy! 🎉
```

### If Scope Too Large
```markdown
## Error: Scope Too Large

**Files found**: 150+ files in [Scope]
**Recommendation**: Narrow the scope to a specific module.
```
