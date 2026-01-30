# Release & Deployment Plan

## Overview

This document outlines the complete CI/CD strategy for automated semantic versioning, GitHub Releases, and Azure deployments across Stage and Production environments.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Azure Infrastructure](#azure-infrastructure)
3. [Environment Configuration](#environment-configuration)
4. [GitHub Actions Workflows](#github-actions-workflows)
5. [Changie Workflow Rules](#changie-workflow-rules)
6. [Fix v2.0.0 Release](#fix-v200-release)
7. [Implementation Checklist](#implementation-checklist)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Developer Workflow                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   1. Make code changes                                                      │
│   2. Run: changie new (select Major/Minor/Patch)                           │
│   3. Commit BOTH code + .changes/unreleased/*.yaml                         │
│   4. Push to feature branch → Create PR to main                            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CI/CD Pipeline                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                        PR OPENED / UPDATED                           │  │
│   │                               │                                      │  │
│   │                               ▼                                      │  │
│   │                    ┌─────────────────────┐                          │  │
│   │                    │   Deploy Preview    │                          │  │
│   │                    │   (per PR, auto)    │                          │  │
│   │                    │                     │                          │  │
│   │                    │  pr-123.preview.    │                          │  │
│   │                    │  azurecontainer...  │                          │  │
│   │                    └─────────────────────┘                          │  │
│   │                               │                                      │  │
│   │                    PR Comment with URL                               │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                        PR MERGED TO MAIN                             │  │
│   │                               │                                      │  │
│   │         ┌─────────────────────┼─────────────────────┐               │  │
│   │         ▼                     ▼                     ▼               │  │
│   │  ┌─────────────┐    ┌─────────────────┐    ┌─────────────┐         │  │
│   │  │   Cleanup   │    │     Changie     │    │  GoReleaser │         │  │
│   │  │   Preview   │    │   batch auto    │    │  (on tag)   │         │  │
│   │  │ Environment │    │   + Release PR  │    │             │         │  │
│   │  └─────────────┘    └─────────────────┘    └─────────────┘         │  │
│   │                                                    │                │  │
│   │                                                    ▼                │  │
│   │                                         ┌───────────────────┐      │  │
│   │                                         │  GitHub Release   │      │  │
│   │                                         │  + Binary Assets  │      │  │
│   │                                         └───────────────────┘      │  │
│   │                                                    │                │  │
│   │                                                    ▼                │  │
│   │                                         ┌───────────────────┐      │  │
│   │                                         │   Deploy PROD     │      │  │
│   │                                         │   (automatic)     │      │  │
│   │                                         └───────────────────┘      │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Azure Infrastructure                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   PREVIEW Environments (ephemeral)           PROD Environment (permanent)   │
│   ┌─────────────────────┐                   ┌─────────────────────┐        │
│   │  Azure Container    │                   │  Azure Container    │        │
│   │  Apps (per PR)      │                   │  Apps              │        │
│   │                     │                   │                     │        │
│   │  ca-mcp-asos-pr-123 │                   │  ca-mcp-asos-prod   │        │
│   │  ca-mcp-asos-pr-456 │                   │                     │        │
│   │  ca-mcp-asos-pr-789 │                   │  FLOW_URI (prod)    │        │
│   │         ...         │                   │  FLOW_USERNAME      │        │
│   │                     │                   │  FLOW_PASSWORD      │        │
│   │  Uses STAGE config  │                   │  FLOW_DATABASE      │        │
│   │  (FLOW_* variables) │                   └─────────────────────┘        │
│   └─────────────────────┘                             │                     │
│            │                                          ▼                     │
│            │                                ┌─────────────────────┐        │
│            │                                │    DNS Proxy        │        │
│            ▼                                │ mcp-asos.ciandt.com │        │
│   Auto-deleted when PR                      └─────────────────────┘        │
│   is merged or closed                                                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Deployment Flow Summary

| Trigger | Environment | URL | Lifecycle |
|---------|-------------|-----|-----------|
| PR opened/updated | Preview | `ca-mcp-asos-pr-{number}.{region}.azurecontainerapps.io` | Ephemeral (deleted on PR close) |
| PR merged to main | Prod | `mcp-asos.ciandt.com` | Permanent |

---

## Azure Infrastructure

### Azure Account Details

| Setting | Value |
|---------|-------|
| **Subscription ID** | `4ca620bb-e12a-40d7-af1e-6b3f8dff8074` |
| **Resource Group** | `ASOS_AI` (existing, shared) |

### Recommended Azure Services

| Service | Purpose | Why |
|---------|---------|-----|
| **Azure Container Apps** | Host the MCP server | Serverless containers, auto-scaling, easy CI/CD integration, cost-effective for variable workloads, supports dynamic preview environments |
| **Azure Container Registry** | Store Docker images | Native integration with Container Apps, secure, geo-replication available |
| **Azure Key Vault** | Store secrets | Secure secret management, integrates with Container Apps environment variables |

### Why Container Apps for Preview Environments

Azure Container Apps is ideal for ephemeral preview environments because:
- **Instant provisioning** - New container apps spin up in seconds
- **Scale to zero** - Preview environments cost nothing when idle
- **Automatic HTTPS** - Each preview gets a unique HTTPS URL automatically
- **Easy cleanup** - Containers can be deleted programmatically via CLI
- **Shared environment** - All previews share the same Container Apps Environment, reducing overhead

### Resource Naming Convention

```
Resource Group:         ASOS_AI                         (existing, shared)
Container Registry:     crmcpasos                       (shared)
CA Environment:         cae-mcp-asos                    (shared for all)
Container App (Prod):   ca-mcp-asos-prod                (permanent)
Container App (Preview): ca-mcp-asos-pr-{PR_NUMBER}     (ephemeral)
Key Vault:              kv-mcp-asos                     (shared)
```

### Environment URLs

| Environment | URL Pattern | Notes |
|-------------|-------------|-------|
| **Preview (PR)** | `ca-mcp-asos-pr-{number}.{ca-env-unique-id}.{region}.azurecontainerapps.io` | Auto-generated by Azure, unique per PR |
| **Production** | `ca-mcp-asos-prod.{ca-env-unique-id}.{region}.azurecontainerapps.io` | Auto-generated by Azure, **never changes** |

### DNS Proxy Setup (Post-Deployment)

Once Production is deployed, you will have a permanent Azure-generated URL like:
```
ca-mcp-asos-prod.{unique-id}.{region}.azurecontainerapps.io
```

**Action required:** Provide this URL to the DNS team to configure:
```
mcp-asos.ciandt.com  →  CNAME  →  ca-mcp-asos-prod.{unique-id}.{region}.azurecontainerapps.io
```

> **Note:** The Production Container App name (`ca-mcp-asos-prod`) is fixed, so the Azure-generated URL will remain stable. The DNS team can safely point to it.

---

## Environment Configuration

### Environment Variables

The application supports environment-injected variables with user override capability:

| Variable | Description | Preview Value | Prod Value |
|----------|-------------|---------------|------------|
| `FLOW_URI` | Flow service endpoint | `https://flow-stage.example.com` | `https://flow.example.com` |
| `FLOW_USERNAME` | Service account username | (from Key Vault - preview) | (from Key Vault - prod) |
| `FLOW_PASSWORD` | Service account password | (from Key Vault - preview) | (from Key Vault - prod) |
| `FLOW_DATABASE` | Database name | `mcp_stage` | `mcp_prod` |

### Variable Resolution Order

```go
// Pseudocode for variable resolution
func GetConfig(key string) string {
    // 1. User-provided config takes precedence
    if userConfig[key] != "" {
        return userConfig[key]
    }
    // 2. Fall back to environment-injected value
    return os.Getenv(key)
}
```

This allows:
- **Preview/Prod environments**: Use pipeline-injected values automatically
- **Local development**: User provides their own config
- **Testing overrides**: User can override any injected value

### DNS Configuration

| Environment | Endpoint | Managed By |
|-------------|----------|------------|
| Preview (PR) | `ca-mcp-asos-pr-{N}.*.azurecontainerapps.io` (auto-generated) | Azure (automatic) |
| **Production** | `ca-mcp-asos-prod.*.azurecontainerapps.io` (auto-generated, stable) | Azure (automatic) |
| **Production (future)** | `mcp-asos.ciandt.com` → CNAME to Azure URL | External DNS team |

---

## GitHub Actions Workflows

### 1. Preview Environment Workflow (PR)

**File:** `.github/workflows/preview.yml`

```yaml
name: Preview Environment

on:
  pull_request:
    types: [opened, synchronize, reopened, closed]
    branches: [main]

permissions:
  contents: read
  pull-requests: write
  packages: write

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}
  AZURE_SUBSCRIPTION: 4ca620bb-e12a-40d7-af1e-6b3f8dff8074
  AZURE_RG: ASOS_AI
  CONTAINER_APP_ENV: cae-mcp-asos

jobs:
  # ============================================
  # BUILD & DEPLOY PREVIEW (on PR open/update)
  # ============================================
  deploy-preview:
    if: github.event.action != 'closed'
    runs-on: ubuntu-latest
    outputs:
      preview-url: ${{ steps.deploy.outputs.url }}
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build and Test
        run: |
          go build -v ./...
          go test -v ./...

      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=pr

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          build-args: |
            VERSION=pr-${{ github.event.pull_request.number }}

      - name: Azure Login
        uses: azure/login@v2
        with:
          creds: ${{ secrets.AZURE_CREDENTIALS }}

      - name: Deploy Preview Environment
        id: deploy
        run: |
          PR_NUMBER=${{ github.event.pull_request.number }}
          APP_NAME="ca-mcp-asos-pr-${PR_NUMBER}"
          IMAGE="${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:pr-${PR_NUMBER}"
          
          # Set subscription
          az account set --subscription ${{ env.AZURE_SUBSCRIPTION }}
          
          # Check if container app exists
          EXISTS=$(az containerapp show \
            --name $APP_NAME \
            --resource-group ${{ env.AZURE_RG }} \
            --query "name" -o tsv 2>/dev/null || echo "")
          
          if [ -z "$EXISTS" ]; then
            # Create new container app
            az containerapp create \
              --name $APP_NAME \
              --resource-group ${{ env.AZURE_RG }} \
              --environment ${{ env.CONTAINER_APP_ENV }} \
              --image $IMAGE \
              --target-port 8080 \
              --ingress external \
              --min-replicas 0 \
              --max-replicas 1 \
              --env-vars \
                "FLOW_URI=${{ secrets.FLOW_URI_PREVIEW }}" \
                "FLOW_USERNAME=${{ secrets.FLOW_USERNAME_PREVIEW }}" \
                "FLOW_PASSWORD=${{ secrets.FLOW_PASSWORD_PREVIEW }}" \
                "FLOW_DATABASE=${{ secrets.FLOW_DATABASE_PREVIEW }}"
          else
            # Update existing container app
            az containerapp update \
              --name $APP_NAME \
              --resource-group ${{ env.AZURE_RG }} \
              --image $IMAGE
          fi
          
          # Get the URL (Azure auto-generates this)
          URL=$(az containerapp show \
            --name $APP_NAME \
            --resource-group ${{ env.AZURE_RG }} \
            --query "properties.configuration.ingress.fqdn" -o tsv)
          
          echo "url=https://${URL}" >> $GITHUB_OUTPUT

      - name: Comment PR with Preview URL
        uses: actions/github-script@v7
        with:
          script: |
            const url = '${{ steps.deploy.outputs.url }}';
            const prNumber = context.payload.pull_request.number;
            
            // Find existing comment
            const comments = await github.rest.issues.listComments({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: prNumber
            });
            
            const botComment = comments.data.find(c => 
              c.user.type === 'Bot' && c.body.includes('Preview Environment')
            );
            
            const body = `## 🚀 Preview Environment

| Status | URL |
|--------|-----|
| ✅ Deployed | [${url}](${url}) |

**Commit:** \`${context.sha.substring(0, 7)}\`
**Updated:** ${new Date().toISOString()}

> This preview uses **staging** configuration (FLOW_* variables).
> Environment will be automatically deleted when PR is closed.`;
            
            if (botComment) {
              await github.rest.issues.updateComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                comment_id: botComment.id,
                body
              });
            } else {
              await github.rest.issues.createComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                issue_number: prNumber,
                body
              });
            }

  # ============================================
  # CLEANUP PREVIEW (on PR close/merge)
  # ============================================
  cleanup-preview:
    if: github.event.action == 'closed'
    runs-on: ubuntu-latest
    steps:
      - name: Azure Login
        uses: azure/login@v2
        with:
          creds: ${{ secrets.AZURE_CREDENTIALS }}

      - name: Delete Preview Environment
        run: |
          PR_NUMBER=${{ github.event.pull_request.number }}
          APP_NAME="ca-mcp-asos-pr-${PR_NUMBER}"
          
          # Set subscription
          az account set --subscription ${{ env.AZURE_SUBSCRIPTION }}
          
          echo "Deleting preview environment: $APP_NAME"
          
          az containerapp delete \
            --name $APP_NAME \
            --resource-group ${{ env.AZURE_RG }} \
            --yes || echo "Container app not found or already deleted"

      - name: Comment PR with Cleanup Notice
        uses: actions/github-script@v7
        with:
          script: |
            const prNumber = context.payload.pull_request.number;
            const merged = context.payload.pull_request.merged;
            
            const status = merged 
              ? '🎉 PR merged! Preview environment deleted.' 
              : '🗑️ PR closed. Preview environment deleted.';
            
            await github.rest.issues.createComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: prNumber,
              body: `## Preview Environment Cleanup\n\n${status}`
            });
```

### 2. Changie Batch Workflow

**File:** `.github/workflows/changie.yml`

```yaml
name: Changie Release PR

on:
  push:
    branches: [main]
    paths:
      - '.changes/unreleased/**'

permissions:
  contents: write
  pull-requests: write

jobs:
  release-pr:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install Changie
        run: go install github.com/miniscruff/changie@latest

      - name: Check for unreleased changes
        id: check
        run: |
          if [ -z "$(ls -A .changes/unreleased/*.yaml 2>/dev/null)" ]; then
            echo "has_changes=false" >> $GITHUB_OUTPUT
          else
            echo "has_changes=true" >> $GITHUB_OUTPUT
          fi

      - name: Batch changes
        if: steps.check.outputs.has_changes == 'true'
        run: |
          changie batch auto
          changie merge

      - name: Get new version
        if: steps.check.outputs.has_changes == 'true'
        id: version
        run: echo "version=$(changie latest)" >> $GITHUB_OUTPUT

      - name: Create Release PR
        if: steps.check.outputs.has_changes == 'true'
        uses: peter-evans/create-pull-request@v6
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          commit-message: "chore(release): v${{ steps.version.outputs.version }}"
          title: "chore(release): v${{ steps.version.outputs.version }}"
          body: |
            ## Release v${{ steps.version.outputs.version }}
            
            This PR was automatically created by the Changie workflow.
            
            ### Changes
            See CHANGELOG.md for details.
            
            ### Deployment
            - Merging this PR will trigger GoReleaser and deploy to **PRODUCTION**
            - Preview environment will be created for this PR
          branch: release/v${{ steps.version.outputs.version }}
          base: main
          labels: release,automated
```

### 3. Production Release Workflow

**File:** `.github/workflows/release.yml`

```yaml
name: Release & Deploy Production

on:
  push:
    branches: [main]
    paths:
      - 'CHANGELOG.md'
  workflow_dispatch:
    inputs:
      version:
        description: 'Version to release (e.g., 2.0.1)'
        required: true

permissions:
  contents: write
  packages: write

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}
  AZURE_SUBSCRIPTION: 4ca620bb-e12a-40d7-af1e-6b3f8dff8074
  AZURE_RG: ASOS_AI

jobs:
  # ============================================
  # CREATE RELEASE
  # ============================================
  release:
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.version.outputs.version }}
      tag: ${{ steps.version.outputs.tag }}
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install Changie
        run: go install github.com/miniscruff/changie@latest

      - name: Get version
        id: version
        run: |
          if [ "${{ github.event_name }}" == "workflow_dispatch" ]; then
            VERSION="${{ github.event.inputs.version }}"
          else
            VERSION=$(changie latest)
          fi
          echo "version=$VERSION" >> $GITHUB_OUTPUT
          echo "tag=v$VERSION" >> $GITHUB_OUTPUT

      - name: Check if tag exists
        id: tag_check
        run: |
          if git rev-parse "v${{ steps.version.outputs.version }}" >/dev/null 2>&1; then
            echo "exists=true" >> $GITHUB_OUTPUT
          else
            echo "exists=false" >> $GITHUB_OUTPUT
          fi

      - name: Create and push tag
        if: steps.tag_check.outputs.exists == 'false'
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git tag -a "v${{ steps.version.outputs.version }}" -m "Release v${{ steps.version.outputs.version }}"
          git push origin "v${{ steps.version.outputs.version }}"

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  # ============================================
  # BUILD PRODUCTION IMAGE
  # ============================================
  build-image:
    needs: release
    runs-on: ubuntu-latest
    outputs:
      image: ${{ steps.meta.outputs.tags }}
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          ref: v${{ needs.release.outputs.version }}

      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=semver,pattern={{version}},value=${{ needs.release.outputs.version }}
            type=semver,pattern={{major}}.{{minor}},value=${{ needs.release.outputs.version }}
            type=raw,value=latest

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          build-args: |
            VERSION=${{ needs.release.outputs.version }}

  # ============================================
  # DEPLOY TO PRODUCTION
  # ============================================
  deploy-prod:
    needs: [release, build-image]
    runs-on: ubuntu-latest
    environment: prod
    steps:
      - name: Azure Login
        uses: azure/login@v2
        with:
          creds: ${{ secrets.AZURE_CREDENTIALS }}

      - name: Deploy to Production
        id: deploy
        run: |
          # Set subscription
          az account set --subscription ${{ env.AZURE_SUBSCRIPTION }}
          
          # Update production container app (assumes it already exists)
          az containerapp update \
            --name ca-mcp-asos-prod \
            --resource-group ${{ env.AZURE_RG }} \
            --image ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ needs.release.outputs.version }}

      - name: Set environment variables
        run: |
          az containerapp update \
            --name ca-mcp-asos-prod \
            --resource-group ${{ env.AZURE_RG }} \
            --set-env-vars \
              "FLOW_URI=${{ secrets.FLOW_URI_PROD }}" \
              "FLOW_USERNAME=${{ secrets.FLOW_USERNAME_PROD }}" \
              "FLOW_PASSWORD=${{ secrets.FLOW_PASSWORD_PROD }}" \
              "FLOW_DATABASE=${{ secrets.FLOW_DATABASE_PROD }}"

      - name: Get Production URL
        id: url
        run: |
          URL=$(az containerapp show \
            --name ca-mcp-asos-prod \
            --resource-group ${{ env.AZURE_RG }} \
            --query "properties.configuration.ingress.fqdn" -o tsv)
          echo "url=https://${URL}" >> $GITHUB_OUTPUT

      - name: Verify deployment
        run: |
          echo "============================================"
          echo "Production deployment complete!"
          echo "Version: v${{ needs.release.outputs.version }}"
          echo "URL: ${{ steps.url.outputs.url }}"
          echo ""
          echo "DNS Team: Point mcp-asos.ciandt.com to:"
          echo "  ${{ steps.url.outputs.url }}"
          echo "============================================"
```

### 4. PR Validation Workflow

**File:** `.github/workflows/pr-check.yml`

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
          # Skip check for release PRs
          if [[ "${{ github.head_ref }}" == release/* ]]; then
            echo "✅ Release PR - skipping changie check"
            exit 0
          fi
          
          # Get changed files in this PR
          CHANGED_FILES=$(git diff --name-only origin/main...HEAD)
          
          if echo "$CHANGED_FILES" | grep -q "^\.changes/unreleased/"; then
            echo "✅ Change entry found"
          else
            echo "❌ ERROR: No change entry found!"
            echo ""
            echo "=========================================="
            echo "  REQUIRED: Run 'changie new' before commit"
            echo "=========================================="
            echo ""
            echo "Steps:"
            echo "  1. changie new"
            echo "  2. Select: Major | Minor | Patch"
            echo "  3. Enter description"
            echo "  4. git add .changes/unreleased/"
            echo "  5. git commit --amend (or new commit)"
            echo ""
            exit 1
          fi

  build-test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build
        run: go build -v ./...

      - name: Test
        run: go test -v ./...

      - name: Lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
```

---

## Changie Workflow Rules

### ⚠️ MANDATORY: Before Every Commit

```bash
# 1. Make your code changes
# 2. ALWAYS run changie new BEFORE committing
changie new

# 3. Select the appropriate change kind:
#    - Major: Breaking changes (API changes, removed features)
#    - Minor: New features (backward compatible)
#    - Patch: Bug fixes, documentation, refactoring

# 4. Commit BOTH your code AND the change file
git add .
git add .changes/unreleased/
git commit -m "feat: your descriptive message"
```

### ✅ DO

| Action | Command |
|--------|---------|
| Add change entry before commit | `changie new` |
| Commit change file with code | `git add .changes/unreleased/` |
| Push to feature branch | `git push origin feature/my-feature` |
| Create PR to main | Via GitHub UI |

### ❌ DON'T

| Action | Why |
|--------|-----|
| Run `changie batch` locally | CI handles this automatically |
| Commit without change entry | PR validation will fail |
| Edit `.changes/` files manually | Use `changie new` instead |
| Skip changie for "small" changes | Every change needs tracking |

### Version Bump Logic

| Unreleased Changes Contain | Version Bump | Example |
|---------------------------|--------------|---------|
| Any `Major` | Major bump | 1.2.3 → 2.0.0 |
| Any `Minor` (no Major) | Minor bump | 1.2.3 → 1.3.0 |
| Only `Patch` | Patch bump | 1.2.3 → 1.2.4 |

### What Happens If `.changes/unreleased/` Is Empty?

- CI workflow detects no changes
- **No Release PR is created**
- No version bump occurs
- This is expected behavior for non-code changes (CI updates, README fixes, etc.)

---

## Fix v2.0.0 Release

### Problem

Version v2.0.0 was committed with a Major breaking change but didn't follow the Changie workflow. This means:
- No proper CHANGELOG entry
- Changie version tracking is out of sync
- Future `changie batch auto` may calculate wrong version

### Solution

#### Option A: Retroactive Fix (Recommended)

1. **Create a manual changelog entry for v2.0.0:**

```bash
# Create the version directory if it doesn't exist
mkdir -p .changes/v2.0.0

# Create a retroactive change entry
cat > .changes/v2.0.0/v2.0.0.md << 'EOF'
## v2.0.0 - $(date +%Y-%m-%d)

### Major
- [Retroactive] Major breaking change description here
EOF
```

2. **Update `.changes/.changefile.yaml`** (or equivalent config) to recognize v2.0.0:

```bash
# Check current latest version Changie knows about
changie latest

# If it shows a version before 2.0.0, you need to sync
```

3. **Create header file:**

```bash
cat > .changes/v2.0.0/header.yaml << 'EOF'
version: 2.0.0
date: 2025-01-30
EOF
```

4. **Merge into CHANGELOG:**

```bash
changie merge
```

5. **Commit the fix:**

```bash
git add .changes/
git add CHANGELOG.md
git commit -m "chore: retroactive changelog entry for v2.0.0"
git push origin main
```

#### Option B: Reset and Re-release

If Option A doesn't work cleanly:

1. **Delete the v2.0.0 tag:**

```bash
# Delete local tag
git tag -d v2.0.0

# Delete remote tag
git push origin :refs/tags/v2.0.0
```

2. **Delete the GitHub Release** (via GitHub UI)

3. **Create proper change entry:**

```bash
changie new --kind Major -m "Description of breaking change"
git add .changes/unreleased/
git commit -m "chore: add missing change entry for v2.0.0"
git push origin main
```

4. **Let CI create proper release PR**

#### Option C: Manual Version Override

If you want to keep v2.0.0 as-is and continue forward:

1. **Ensure Changie knows current version:**

Check `.changie.yaml` for version source, typically it reads from a file or git tags.

2. **Verify:**

```bash
changie latest
# Should output: 2.0.0
```

3. **If not showing 2.0.0, manually create version marker:**

```bash
mkdir -p .changes/v2.0.0
touch .changes/v2.0.0/.gitkeep
echo "version: 2.0.0" > .changes/v2.0.0/header.yaml
```

4. **Next changes will be 2.0.1, 2.1.0, or 3.0.0 correctly**

---

## Implementation Checklist

### Phase 1: Fix Current State

- [ ] Fix v2.0.0 release (choose Option A, B, or C above)
- [ ] Verify `changie latest` returns correct version
- [ ] Ensure CHANGELOG.md is up to date

### Phase 2: GitHub Setup

- [ ] Create `.github/workflows/preview.yml`
- [ ] Create `.github/workflows/changie.yml`
- [ ] Create `.github/workflows/release.yml`
- [ ] Create `.github/workflows/pr-check.yml`
- [ ] Create GitHub Environment: `prod`
- [ ] Add repository secrets (see below)

### Phase 3: Azure Infrastructure (All in ASOS_AI Resource Group)

**One-time setup commands:**

```bash
# Set subscription
az account set --subscription 4ca620bb-e12a-40d7-af1e-6b3f8dff8074

# Create Container Registry (if not exists)
az acr create \
  --name crmcpasos \
  --resource-group ASOS_AI \
  --sku Basic

# Create Container Apps Environment (shared for all)
az containerapp env create \
  --name cae-mcp-asos \
  --resource-group ASOS_AI \
  --location uksouth

# Create Production Container App (one-time, will be updated by CI)
az containerapp create \
  --name ca-mcp-asos-prod \
  --resource-group ASOS_AI \
  --environment cae-mcp-asos \
  --image mcr.microsoft.com/azuredocs/containerapps-helloworld:latest \
  --target-port 8080 \
  --ingress external \
  --min-replicas 1 \
  --max-replicas 3

# Get Production URL (provide this to DNS team)
az containerapp show \
  --name ca-mcp-asos-prod \
  --resource-group ASOS_AI \
  --query "properties.configuration.ingress.fqdn" -o tsv

# Create Key Vault (optional, for secret management)
az keyvault create \
  --name kv-mcp-asos \
  --resource-group ASOS_AI \
  --location uksouth
```

**Checklist:**

- [ ] Create Container Registry: `crmcpasos`
- [ ] Create Container Apps Environment: `cae-mcp-asos`
- [ ] Create Production Container App: `ca-mcp-asos-prod`
- [ ] Note Production URL for DNS team
- [ ] Create Key Vault: `kv-mcp-asos` (optional)
- [ ] Create Service Principal for GitHub Actions
- [ ] Add `AZURE_CREDENTIALS` to GitHub Secrets

### Phase 4: Service Principal for GitHub Actions

```bash
# Create Service Principal with Contributor role on ASOS_AI resource group
az ad sp create-for-rbac \
  --name "github-mcp-asos-deploy" \
  --role contributor \
  --scopes /subscriptions/4ca620bb-e12a-40d7-af1e-6b3f8dff8074/resourceGroups/ASOS_AI \
  --sdk-auth

# Output JSON goes into AZURE_CREDENTIALS secret
```

### Phase 5: Secrets Configuration

**GitHub Repository Secrets:**

| Secret | Description |
|--------|-------------|
| `AZURE_CREDENTIALS` | Service Principal JSON (from Phase 4) |
| `FLOW_URI_PREVIEW` | Flow URI for Preview/Staging |
| `FLOW_URI_PROD` | Flow URI for Production |
| `FLOW_USERNAME_PREVIEW` | Flow username for Preview/Staging |
| `FLOW_USERNAME_PROD` | Flow username for Production |
| `FLOW_PASSWORD_PREVIEW` | Flow password for Preview/Staging |
| `FLOW_PASSWORD_PROD` | Flow password for Production |
| `FLOW_DATABASE_PREVIEW` | Flow database for Preview/Staging |
| `FLOW_DATABASE_PROD` | Flow database for Production |

### Phase 6: DNS Configuration (External Team)

Once Production is deployed, provide the DNS team with:

```
Source:      mcp-asos.ciandt.com
Type:        CNAME
Target:      ca-mcp-asos-prod.{unique-id}.uksouth.azurecontainerapps.io
```

**To get the exact target URL:**
```bash
az containerapp show \
  --name ca-mcp-asos-prod \
  --resource-group ASOS_AI \
  --query "properties.configuration.ingress.fqdn" -o tsv
```

### Phase 7: Testing

- [ ] Create test feature branch
- [ ] Run `changie new` and commit
- [ ] Create PR → Verify PR check passes
- [ ] Verify Preview environment is created (`ca-mcp-asos-pr-{N}`)
- [ ] Verify PR comment with Preview URL
- [ ] Merge PR → Verify Preview environment is deleted
- [ ] Verify Release PR is created (if changes exist)
- [ ] Merge Release PR → Verify GoReleaser runs
- [ ] Verify Prod deployment
- [ ] Note Production URL and send to DNS team

### Phase 8: Documentation

- [ ] Update README.md with contribution workflow
- [ ] Document Changie rules for team
- [ ] Create onboarding guide for new developers

---

## Quick Reference Card

```
┌─────────────────────────────────────────────────────────┐
│                  DEVELOPER WORKFLOW                      │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  1. Make code changes                                   │
│                                                         │
│  2. ALWAYS run:  changie new                           │
│     └── Select: Major | Minor | Patch                  │
│     └── Enter description                              │
│                                                         │
│  3. Commit BOTH:                                        │
│     git add .                                          │
│     git add .changes/unreleased/                       │
│     git commit -m "feat: description"                  │
│                                                         │
│  4. Push & Create PR                                    │
│     └── Preview environment auto-created               │
│     └── PR comment with preview URL                    │
│                                                         │
│  5. Merge PR                                            │
│     └── Preview environment auto-deleted               │
│     └── Release PR created (if changes)                │
│                                                         │
│  6. Merge Release PR                                    │
│     └── PROD deployed automatically                    │
│                                                         │
│  ❌ NEVER run: changie batch (CI handles this)         │
│                                                         │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                  ENVIRONMENT SUMMARY                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  PREVIEW (per PR)                                       │
│  ├── Trigger: PR opened/updated                        │
│  ├── URL: ca-mcp-asos-pr-{N}.*.azurecontainerapps.io  │
│  ├── Config: Uses FLOW_*_PREVIEW secrets               │
│  └── Lifecycle: Deleted when PR closed/merged          │
│                                                         │
│  PRODUCTION                                             │
│  ├── Trigger: Release PR merged to main                │
│  ├── URL: ca-mcp-asos-prod.*.azurecontainerapps.io    │
│  ├── DNS: mcp-asos.ciandt.com (external team)         │
│  ├── Config: Uses FLOW_*_PROD secrets                  │
│  └── Lifecycle: Permanent                              │
│                                                         │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                  AZURE DETAILS                           │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Subscription: 4ca620bb-e12a-40d7-af1e-6b3f8dff8074    │
│  Resource Group: ASOS_AI                                │
│  Container Apps Env: cae-mcp-asos                      │
│  Production App: ca-mcp-asos-prod                      │
│  Preview Apps: ca-mcp-asos-pr-{PR_NUMBER}              │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## Support

For issues with this workflow:
1. Check GitHub Actions logs
2. Verify Changie configuration: `changie latest`
3. Check Azure deployment logs
4. Contact: [Your Team Contact]
