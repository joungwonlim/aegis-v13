# i18n Synchronization Automation

**Purpose**: Keep translation files synchronized across all languages when new keys are added.

**Trigger**:
- New component created with hardcoded text
- i18n key added to one language but not others
- User request: "sync i18n" or "check translations"

---

## Problem Statement

When developers add new UI text, they often:
1. ❌ Hardcode text in components (`<h1>Welcome</h1>`)
2. ❌ Add keys only to `en.json`, forgetting `ko.json`
3. ❌ Use inconsistent key naming

**Result**: Missing translations, broken UI in non-English locales.

---

## Workflow

### Step 1: Detect New Text

Scan for:
- **Hardcoded text** in components
- **New keys** in one language but missing in others
- **Unused keys** (defined but never referenced)

```bash
cd frontend

# Find hardcoded text (heuristic)
grep -r ">[A-Z][a-z].*<" src/app/ src/modules/ | \
  grep -v "i18n\|t("
```

---

### Step 2: Extract Translation Keys

For each new text found:

#### Example: Component with Hardcoded Text
```tsx
// ❌ Before
export function WelcomeBanner() {
  return <h1>Welcome to Aegis</h1>
}

// ✅ After
export function WelcomeBanner() {
  const { t } = useTranslation()
  return <h1>{t('welcome.banner.title')}</h1>
}
```

---

### Step 3: Synchronize Keys Across Languages

#### Algorithm
```
1. Load all i18n JSON files (en, ko)
2. Extract all keys from each file
3. Find missing keys:
   - Keys in en.json but not in ko.json
4. For each missing key:
   - Add to target language with placeholder
   - Format: "[TRANSLATE] {english_value}"
5. Sort keys alphabetically
6. Write updated JSON files
```

---

### Step 4: Validate Translations

Ensure no keys are still using placeholders:

```bash
cd frontend
grep -r "\[TRANSLATE\]" src/i18n/*.json
```

---

## Aegis-Specific Rules

### Key Naming Convention
Follow this structure for consistency:

```
{module}.{component}.{element}.{variant}
```

**Examples**:
```json
{
  "ranking": {
    "page": {
      "title": "Market Rankings"
    },
    "table": {
      "header": {
        "rank": "Rank",
        "name": "Stock Name"
      }
    }
  }
}
```

### Priority Languages
1. **English (en)**: Source of truth
2. **Korean (ko)**: Primary user base

### Translation Workflow
1. Developer adds key to `en.json` with English text
2. Auto-sync adds `[TRANSLATE]` placeholder to other languages
3. Translator updates `ko.json`
4. Remove `[TRANSLATE]` prefix after translation

---

## Reporting

### Sync Report Format

```markdown
## i18n Sync Report

**Date**: 2026-01-03 20:30:00 KST
**Trigger**: Component `WelcomeBanner` added

### Changes
- ✅ Added 5 new keys to en.json
- ✅ Synced to ko.json (5 keys with [TRANSLATE] placeholder)

### New Keys
1. `welcome.banner.title` - "Welcome to Aegis"
2. `welcome.banner.subtitle` - "AI-powered trading system"

### Action Required
⚠️ Translate 5 keys in ko.json
```

---

## Validation Rules

### Required Checks
- [ ] All keys in `en.json` exist in `ko.json`
- [ ] No `[TRANSLATE]` placeholders in production build
- [ ] No unused keys
- [ ] Consistent key naming (lowercase, dot-separated)
