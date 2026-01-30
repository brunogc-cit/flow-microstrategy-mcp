# Release & Deployment Specification

## Document Info

| Field | Value |
|-------|-------|
| **Created** | 2026-01-30 |
| **Based On** | `.claude/plan/RELEASE_DEPLOYMENT_PLAN.md` |
| **Status** | ✅ Implemented |

---

## 1. Current State Analysis

### 1.1 Existing Infrastructure

| Component | Status | Location | Notes |
|-----------|--------|----------|-------|
| GoReleaser | ✅ Exists | `.goreleaser.yaml` | Multi-platform builds, macOS notarization configured |
| Changie | ✅ Exists | `.changie.yaml` | Properly configured with Major/Minor/Patch kinds |
| Dockerfile | ✅ Exists | `Dockerfile` | Multi-stage build, scratch runtime, non-root user |
| Build/Test Workflow | ✅ Exists | `.github/workflows/build-and-test.yml` | Unit, integration, e2e tests |
| Changie Workflow | ✅ Exists | `.github/workflows/changie.yml` | Auto-creates release PR |
| Release Workflow | ✅ Exists | `.github/workflows/release.yml` | Triggers GoReleaser on CHANGELOG change |
| Preview Environments | ❌ Missing | - | Proposed in plan |
| Azure Deployment | ❌ Missing | - | Proposed in plan |
| PR Validation (changie) | ❌ Missing | - | Proposed in plan |

### 1.2 Current Workflow Flow

```
Developer → Push to branch → Create PR
                ↓
        build-and-test.yml runs (unit, integration, e2e)
                ↓
        PR merged to main
                ↓
        If .changes/unreleased/* modified:
            changie.yml → creates Release PR
                ↓
        Release PR merged:
            CHANGELOG.md updated
                ↓
            release.yml → GoReleaser → GitHub Release + Binaries
```

### 1.2.1 Target Workflow Flow (After Implementation)

```
Developer → Push to branch → Create PR
                ↓
        ┌───────────────────────────────────────────────────────────┐
        │                    PR OPENED/UPDATED                       │
        ├───────────────────────────────────────────────────────────┤
        │  Parallel execution:                                       │
        │  ├─ build-and-test.yml (unit, integration, e2e)           │
        │  ├─ pr-check.yml (validates changie entry exists)         │
        │  └─ preview.yml (deploy ephemeral preview environment)    │
        │                                                            │
        │  → Comment on PR with preview URL:                         │
        │    https://ca-mcp-asos-pr-{N}.*.azurecontainerapps.io     │
        └───────────────────────────────────────────────────────────┘
                ↓
        ┌───────────────────────────────────────────────────────────┐
        │                    PR MERGED TO MAIN                       │
        ├───────────────────────────────────────────────────────────┤
        │  preview.yml (on: closed) → Delete ephemeral environment  │
        │                                                            │
        │  If .changes/unreleased/* modified:                        │
        │      changie.yml → creates Release PR                      │
        └───────────────────────────────────────────────────────────┘
                ↓
        ┌───────────────────────────────────────────────────────────┐
        │                    RELEASE PR MERGED                       │
        ├───────────────────────────────────────────────────────────┤
        │  CHANGELOG.md updated by changie merge                     │
        │          ↓                                                 │
        │  release.yml triggered:                                    │
        │  ├─ GoReleaser → GitHub Release + Multi-platform Binaries │
        │  └─ deploy-prod.yml → Deploy to Azure Production          │
        │                                                            │
        │  Production URL:                                           │
        │    https://ca-mcp-asos-prod.*.azurecontainerapps.io       │
        └───────────────────────────────────────────────────────────┘
```

### 1.3 Issues Identified

| Issue | Severity | Description |
|-------|----------|-------------|
| **v2.0.0 Sync** | ✅ Fixed | Created `.changes/v2.0.0.md` |
| **Env Var Mismatch** | ✅ Fixed | Updated `build-and-test.yml` to use `FLOW_*` env vars |
| **Secrets Naming** | ⚠️ Manual | GitHub secrets need to be renamed from `AURA_*` to `FLOW_*` in GitHub Settings |
| **No Preview Envs** | ✅ Fixed | Created `.github/workflows/preview.yml` |
| **No Azure Deploy** | ✅ Fixed | Created `.github/workflows/deploy-prod.yml` |
| **No Changie PR Check** | ✅ Fixed | Created `.github/workflows/pr-check.yml` |

---

## 2. Implementation Phases

### Phase 1: Fix v2.0.0 Release Sync (Required)

**Problem:** Changie doesn't know about v2.0.0 because it was committed directly.

**Solution:** Create version marker file for Changie.

**Files to Create:**

```
.changes/v2.0.0.md
```

**Content for `.changes/v2.0.0.md`:**

```markdown
## v2.0.0 - 2026-01-30
### Major
* **BREAKING:** Rebrand from Neo4j MCP to Flow Microstrategy MCP (powered by CI&T Flow).
* **BREAKING:** Renamed binary from `neo4j-mcp` to `flow-microstrategy-mcp`.
* **BREAKING:** Changed module path from `github.com/neo4j/mcp` to `github.com/brunogc-cit/flow-microstrategy-mcp`.
* **BREAKING:** All environment variables renamed from `NEO4J_*` to `FLOW_*`.
* **BREAKING:** All CLI flags renamed from `--neo4j-*` to `--flow-*`.
```

**Verification:**

```bash
changie latest
# Should output: v2.0.0
```

---

### Phase 2: Fix Environment Variables and Secrets in Workflows (Required)

**File:** `.github/workflows/build-and-test.yml`

**Changes Required:**

Both the GitHub secrets and environment variable names need to be updated for consistency with the rebranding.

```yaml
# BEFORE (lines 70-72, 99-101)
env:
  NEO4J_URI: ${{ secrets.AURA_URL }}
  NEO4J_USERNAME: ${{ secrets.AURA_USERNAME }}
  NEO4J_PASSWORD: ${{ secrets.AURA_PASSWORD }}

# AFTER
env:
  FLOW_URI: ${{ secrets.FLOW_URL }}
  FLOW_USERNAME: ${{ secrets.FLOW_USERNAME }}
  FLOW_PASSWORD: ${{ secrets.FLOW_PASSWORD }}
```

**GitHub Secrets Migration Required:**

| Old Secret | New Secret | Action |
|------------|------------|--------|
| `AURA_URL` | `FLOW_URL` | Rename in GitHub Settings |
| `AURA_USERNAME` | `FLOW_USERNAME` | Rename in GitHub Settings |
| `AURA_PASSWORD` | `FLOW_PASSWORD` | Rename in GitHub Settings |

---

### Phase 3: Add PR Validation for Changie (Recommended)

**File to Create:** `.github/workflows/pr-check.yml`

**Purpose:** Ensure all PRs include changie entries (except release PRs).

**Specification:**

```yaml
name: PR Validation

on:
  pull_request:
    branches: [main]

jobs:
  check-changie:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Check for change entry
        run: |
          # Skip check for release PRs and dependency updates
          if [[ "${{ github.head_ref }}" == release/* ]] || [[ "${{ github.head_ref }}" == dependabot/* ]]; then
            echo "✅ Automated PR - skipping changie check"
            exit 0
          fi

          # Get changed files in this PR
          CHANGED_FILES=$(git diff --name-only origin/main...HEAD)

          # Skip if only docs/CI changes
          if echo "$CHANGED_FILES" | grep -qvE "^(\.github/|README|CONTRIBUTING|docs/|\.md$)"; then
            # Code changes detected, require changie entry
            if echo "$CHANGED_FILES" | grep -q "^\.changes/unreleased/"; then
              echo "✅ Change entry found"
            else
              echo "❌ ERROR: No change entry found!"
              echo ""
              echo "Run 'changie new' before committing code changes."
              exit 1
            fi
          else
            echo "✅ Documentation-only changes - no changie entry required"
          fi
```

---

### Phase 4: Azure Container Apps Deployment (Required)

Deploy to Azure Container Apps with **2 environments only**:

| Environment | Trigger | Lifecycle | URL Pattern |
|-------------|---------|-----------|-------------|
| **Ephemeral (Preview)** | PR opened/updated | Deleted when PR closed/merged | `ca-mcp-asos-pr-{N}.*.azurecontainerapps.io` |
| **Production** | Release created | Permanent | `ca-mcp-asos-prod.*.azurecontainerapps.io` |

#### Prerequisites

- Azure subscription access (Subscription ID: `4ca620bb-e12a-40d7-af1e-6b3f8dff8074`)
- Service Principal for GitHub Actions
- Container Registry (GHCR - GitHub Container Registry)

#### New Workflow Files Required

| File | Purpose |
|------|---------|
| `.github/workflows/preview.yml` | Deploy ephemeral preview on PR open, delete on PR close |
| `.github/workflows/deploy-prod.yml` | Deploy to production on release |

#### GitHub Secrets Required

| Secret | Description |
|--------|-------------|
| `AZURE_CREDENTIALS` | Service Principal JSON for Azure deployment |
| `FLOW_URI_PREVIEW` | Neo4j URI for preview environments |
| `FLOW_URI_PROD` | Neo4j URI for production |
| `FLOW_USERNAME_PREVIEW` | Neo4j username for preview |
| `FLOW_USERNAME_PROD` | Neo4j username for production |
| `FLOW_PASSWORD_PREVIEW` | Neo4j password for preview |
| `FLOW_PASSWORD_PROD` | Neo4j password for production |

#### Azure Resources Required

| Resource | Name | Purpose |
|----------|------|---------|
| Resource Group | `ASOS_AI` (existing) | Container for all resources |
| Container Apps Environment | `cae-mcp-asos` | Infrastructure to host Container Apps |
| Container App (Prod) | `ca-mcp-asos-prod` | Production deployment (permanent) |
| Container App (Preview) | `ca-mcp-asos-pr-{N}` | Ephemeral per-PR (auto-created/deleted) |

#### Deployment Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    PR OPENED/UPDATED                         │
├─────────────────────────────────────────────────────────────┤
│  1. Build Docker image                                       │
│  2. Push to GHCR: ghcr.io/brunogc-cit/flow-microstrategy-mcp:pr-{N} │
│  3. Create/Update Azure Container App: ca-mcp-asos-pr-{N}   │
│  4. Comment PR with preview URL                              │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    PR CLOSED/MERGED                          │
├─────────────────────────────────────────────────────────────┤
│  1. Delete Azure Container App: ca-mcp-asos-pr-{N}          │
│  2. Comment PR with cleanup notice                           │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    RELEASE CREATED                           │
├─────────────────────────────────────────────────────────────┤
│  1. Build Docker image                                       │
│  2. Push to GHCR: ghcr.io/brunogc-cit/flow-microstrategy-mcp:v{X.Y.Z} │
│  3. Update Azure Container App: ca-mcp-asos-prod            │
│  4. Production is live                                       │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. File Change Summary

### Files to Modify

| File | Change Type | Description |
|------|-------------|-------------|
| `.github/workflows/build-and-test.yml` | Modify | Update env vars from `NEO4J_*` to `FLOW_*` |

### Files to Create

| File | Priority | Description |
|------|----------|-------------|
| `.changes/v2.0.0.md` | High | Sync Changie with v2.0.0 release |
| `.github/workflows/pr-check.yml` | Medium | PR validation for changie entries |
| `.github/workflows/preview.yml` | Low | Preview environment deployment (if Azure chosen) |
| `.github/workflows/deploy-prod.yml` | Low | Production deployment (if Azure chosen) |

### No Changes Required

| File | Reason |
|------|--------|
| `.goreleaser.yaml` | Already correctly configured for `flow-microstrategy-mcp` |
| `.changie.yaml` | Already correctly configured |
| `Dockerfile` | Already correctly configured |
| `.github/workflows/changie.yml` | Works correctly |
| `.github/workflows/release.yml` | Works correctly |

---

## 4. Implementation Order

```
┌─────────────────────────────────────────────────────────────┐
│  PHASE 1: Critical Fixes (Do First)                         │
├─────────────────────────────────────────────────────────────┤
│  1.1 Create .changes/v2.0.0.md                              │
│  1.2 Verify: changie latest → v2.0.0                        │
│  1.3 Update build-and-test.yml env vars                     │
│  1.4 Commit & Push                                          │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  PHASE 2: Workflow Improvements (Recommended)               │
├─────────────────────────────────────────────────────────────┤
│  2.1 Create .github/workflows/pr-check.yml                  │
│  2.2 Test with a sample PR                                  │
│  2.3 Commit & Push                                          │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  PHASE 3: Azure Deployment (Required)                       │
├─────────────────────────────────────────────────────────────┤
│  3.1 Create Azure resources (Container Apps Environment)    │
│  3.2 Create Service Principal for GitHub Actions            │
│  3.3 Add GitHub secrets (AZURE_CREDENTIALS, FLOW_*_PROD,    │
│      FLOW_*_PREVIEW)                                        │
│  3.4 Create .github/workflows/preview.yml                   │
│  3.5 Create .github/workflows/deploy-prod.yml               │
│  3.6 Test: Open PR → verify preview created                 │
│  3.7 Test: Close PR → verify preview deleted                │
│  3.8 Test: Create release → verify prod deployed            │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. Validation Checklist

### After Phase 1

- [ ] `changie latest` returns `v2.0.0`
- [ ] `changie next auto` returns correct next version
- [ ] GitHub Actions pass with new env vars
- [ ] Integration tests pass with `FLOW_*` variables

### After Phase 2

- [ ] PR without changie entry is rejected (for code changes)
- [ ] PR with changie entry passes
- [ ] Release PRs are not blocked
- [ ] Documentation-only PRs are not blocked

### After Phase 3 (If Implemented)

- [ ] Preview environment created on PR open
- [ ] Preview environment deleted on PR close
- [ ] Production deployed on release
- [ ] DNS configured for production URL

---

## 6. Plan Deviations

| Plan Proposal | Actual Decision | Reason |
|---------------|-----------------|--------|
| Use GHCR | Keep GHCR (plan matches) | Standard for GitHub projects |
| Complex Azure setup | Defer to Phase 3 | Not immediately needed, binary releases work |
| Mandatory changie for all PRs | Skip for docs/CI | Reduces friction for minor changes |
| Stage environment | Merge with Preview | One ephemeral env per PR is sufficient |

---

## 7. Secrets Mapping

### Current Secrets (To Be Renamed)

| Current Secret | New Secret | Used In | Purpose |
|----------------|------------|---------|---------|
| `AURA_URL` | `FLOW_URL` | build-and-test.yml | Neo4j test instance URI |
| `AURA_USERNAME` | `FLOW_USERNAME` | build-and-test.yml | Neo4j test credentials |
| `AURA_PASSWORD` | `FLOW_PASSWORD` | build-and-test.yml | Neo4j test credentials |

### Unchanged Secrets

| Secret Name | Used In | Purpose |
|-------------|---------|---------|
| `TEAM_GRAPHQL_PERSONAL_ACCESS_TOKEN` | changie.yml, release.yml | GitHub API access |
| `MACOS_SIGN_P12` | release.yml | macOS code signing |
| `MACOS_SIGN_PASSWORD` | release.yml | macOS code signing |
| `MACOS_NOTARY_*` | release.yml | macOS notarization |

### New Secrets Required (For Azure Deployment)

| Secret Name | Purpose |
|-------------|---------|
| `AZURE_CREDENTIALS` | Service Principal JSON for Azure deployment |
| `FLOW_URI_PREVIEW` | Neo4j URI for preview environments |
| `FLOW_USERNAME_PREVIEW` | Neo4j username for preview |
| `FLOW_PASSWORD_PREVIEW` | Neo4j password for preview |
| `FLOW_URI_PROD` | Neo4j URI for production |
| `FLOW_USERNAME_PROD` | Neo4j username for production |
| `FLOW_PASSWORD_PROD` | Neo4j password for production |

---

## 8. Quick Reference

### Developer Workflow (After Implementation)

```bash
# 1. Make changes
git checkout -b feature/my-feature

# 2. Create changie entry (for code changes)
changie new
# Select: Major | Minor | Patch
# Enter description

# 3. Commit both code and changie entry
git add .
git commit -m "feat: my feature"

# 4. Push and create PR
git push -u origin feature/my-feature
# Create PR via GitHub

# 5. PR gets reviewed, tests run
# 6. PR merged → Release PR auto-created (if changie entries exist)
# 7. Release PR merged → GoReleaser creates release
```

### Useful Commands

```bash
# Check current version
changie latest

# Preview next version
changie next auto

# Create new change entry
changie new

# Batch changes (CI only - don't run locally)
changie batch auto

# Merge into CHANGELOG (CI only)
changie merge
```
