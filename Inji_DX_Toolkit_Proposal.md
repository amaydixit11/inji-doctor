# Proposal: Inji Developer Experience (DX) Toolkit

**Internship Project Proposal — MOSIP / Inji Ecosystem**

> **Author:** Amay Dixit  
> **Date:** April 2026  
> **Target:** Inji GitHub Organization (github.com/inji)

---

## 1. The Problem

### 1.1 What the Community Is Telling Us (Two Years of Evidence)

The MOSIP Community Forum (community.mosip.io) has seen **~57 posts about Inji setup, integration, and debugging over the past two years** (April 2024 – April 2026). These are not hypothetical problems — they are real developers, deployers, and contributors who hit walls before they could even start using the Inji stack.

#### Recent Posts (Last 30 Days)

| Post Title | Date | Category |
|---|---|---|
| **"Need Guidance on Local Integration Workflow for Inji Wallet, eSignet, and Inji Certify"** | 1 day ago | Developer |
| **"Issue while integrating Inji Wallet with eSignet locally – Token request failed (400 invalid_request)"** | 25 days ago | Developer |
| **"Mimoto v0.19.2 fails during startup issuer validation when issuer config is loaded from local filesystem"** | 25 days ago | Developer |

#### Posts from the Past 6 Months

| Post Title | Date | Category |
|---|---|---|
| "Downloading Verifiable Credentials failing with An Error Occurred Due to technical error" | 5 days ago | — |
| "High CPU Utilisation Observed with INJI Certify" | 4 days ago | Inji |
| "Self Registration Collab Credentials Not Working" | 10 days ago | eSignet |
| "Inji wallet - national ID Add failure" | 1 day ago | Inji |
| "Unable to start mock-identity-system" | 2 days ago | Developer |

#### Posts from the Past 12 Months (Selected)

| Post Title | Date | Category |
|---|---|---|
| "Not able to deploy inji-certify via docker-compose or helm" | Jul 2025 | Support |
| "Problem downloading credentials in the inji stack setup" | May 2025 | Developer |
| "Issues with Docker Compose in inji-certify-567 Branch from Certify Workshop" | Dec 2024 | Developer |
| "Docker - DataProviderPlugin not found" | Mar 2025 | Developer |
| "Docker - Mimoto - Unsatisfied dependency - cbeffutil" | Mar 2025 | Developer |
| "Unable to run mimoto as docker container" | Jan 2025 | — |
| "Help Needed: Stuck on Step 3 of Mimoto Setup on Local Machine" | Dec 2024 | — |
| "Inji-wallet only showing a white screen when opened" | Mar 2025 | Developer |
| "Inji mobile wallet local setup" | Jun 2024 | — |
| "Mimoto API Local setup error" | Sep 2024 | — |
| "Inji Web local setup" | Nov 2024 | — |
| "How to integrate Inji Certify with Inji Web to issue Verifiable Credentials" | Sep 2025 | — |
| "Using Inji with Ory as an Authentication Server" | Sep 2025 | Developer |
| "Invalid_grant and aud claim is not valid during OpenID4VCI issuance" | Sep 2025 | Support |
| "Run eSignet using docker compose" | Jan 2026 | Developer |
| "eSignet on local deployment" | Jul 2025 | — |
| "Esignet docker containers error, urgent!" | Mar 2025 | Support |
| "Error on eSignet docker compose startup on ubuntu" | Nov 2025 | eSignet |
| "Esignet is crashing due to plugin not found" | Jul 2025 | Developer |

### 1.2 The Pattern Is Clear

| Problem Category | Number of Posts | % of Inji Posts |
|---|---|---|
| **Local setup / Docker Compose doesn't work** | 19 | 33% |
| **Integration between components is unclear** | 8 | 14% |
| **eSignet setup failures** | 8 | 14% |
| **Credential download / verification failures** | 7 | 12% |
| **Configuration / authentication issues** | 5 | 9% |
| **Other Inji issues (build, install, crash)** | 10 | 18% |
| **Total** | **~57** | **100%** |

### 1.3 The Timeline Tells the Story

This is not a one-time problem. It has been **persistent and unresolved for 24 consecutive months**:

| Period | Inji Setup/Integration Posts |
|--------|-----------------------------|
| Apr–Jun 2024 | 6 posts |
| Jul–Dec 2024 | 8 posts |
| Jan–Jun 2025 | 10 posts |
| Jul–Dec 2025 | 11 posts |
| Jan–Apr 2026 | 8 posts (still ongoing) |

**Every single month, someone new hits the same wall.**

### 1.4 What Developers Are Actually Saying

> *"I attempted to deploy Inji stack (comprising injiweb, injicertify, and injiverify) independently of MOSIP using Docker Compose on my local system. I used the v0.11.0 tag of inji-certify to execute the docker-compose.yaml but the process fails."*
> — Jul 2025

> *"I am trying to run Inji Certify locally and set up eSignet, IDA, and Issuer Data locally. Inji docs give steps for Docker setup that uses the MOSIP Collab environment. What are the steps to configure Inji Certify and eSignet locally?"*
> — May 2025

> *"After following all the steps, everything seems to be up and running, but when I try to download the farmer VC, it fails at the end."*
> — May 2025

> *"I'm trying to run the Inji Wallet and eSignet together in my local setup... during the login flow in Inji Wallet, after entering the mock ID and PIN, the wallet makes a POST request to eSignet token endpoint which returns 400 invalid_request."*
> — Mar 2026 (25 days ago)

> *"I am currently trying to integrate Inji Wallet, eSignet, and Inji Certify in my local environment. However, I am facing some issues due to my limited understanding of the overall integration workflow."*
> — Apr 2026 (1 day ago)

---

## 2. What Already Exists (And Why It's Not Enough)

### 2.1 Per-Component Docker Compose — Exists, But Isolated

| Component | Docker Compose? | Location | Status |
|-----------|---------------|----------|--------|
| Inji Certify | ✅ Yes | `inji-certify/docker-compose/` | Exists but community reports it's broken |
| Inji Verify | ✅ Yes | `inji-verify/docker-compose/` | Exists |
| Mimoto (Wallet Backend) | ✅ Yes | `mimoto/docker-compose/` | Exists but requires 8 manual config steps |
| Inji Web | ⚠️ Partial | Docs say IntelliJ setup | Manual setup only, no Docker Compose |

**The problem:** Each component has its own docker-compose with its own config format, its own ports, and its own dependency requirements. **None of them are designed to work together.**

### 2.2 Helm Charts — For Production, Not Local Dev

| Component | Helm Chart? | Location |
|-----------|------------|----------|
| Inji Certify | ✅ Yes | `inji-certify/helm/inji-certify/` |
| Inji Verify | ✅ Yes | `inji-verify/helm/` |
| Inji Stack | ⚠️ Minimal | `inji/helm/` (empty — only LICENSE + README) |

**The problem:** Helm charts require a full Kubernetes cluster. They are for production deployment, not local development.

### 2.3 Documentation — Fragmented

| Resource | URL | Quality |
|----------|-----|---------|
| Inji Certify Local Setup | docs.inji.io/inji-certify/build-and-deploy/local-setup | Exists — but community says it doesn't work |
| Mimoto Local Setup | docs.inji.io/inji-wallet/inji-mobile/build-and-deployment/local-setup | Exists — requires manual OIDC client setup, ngrok, config editing |
| Inji Web Local Setup | docs.inji.io/inji-wallet/inji-web/build-and-deploy/local-setup | IntelliJ-based, not Docker |
| Inji Deployment Guide | docs.inji.io/readme/setup/deploy | 500+ lines of K8s infrastructure setup |

**The problem:** Documentation exists but it's **fragmented** (each component has its own page) and **complex** (the full deployment guide is 500+ lines of WireGuard, RKE, Istio, NFS, Nginx, SSL setup).

### 2.4 Test Suites — Component-Scoped Only

| Component | Has Tests? | Type | Scope |
|-----------|-----------|------|-------|
| Inji Certify | ✅ Yes | `api-test/` | Tests Certify APIs only |
| Inji Verify | ✅ Yes | `api-test/`, `ui-test/` | Tests Verify only |
| Inji Wallet | ✅ Yes | Jest, `__mocks__/` | Unit tests only |

**The problem:** These test individual components. There is **no test** that verifies: "Issue a credential in Certify → Receive it in Wallet → Verify it in Verify."

### 2.5 Error Messages — Cryptic

Common errors reported by developers:

```
400 invalid_request
```

```
RESIDENT-APP-026 – Api not accessible
```

```
Could not obtain jwks from url
```

```
DataProviderPlugin not found
```

```
Error connecting to OIDC service (WebClient) Problem in connecting to auth service or UNKNOWN Error
```

**The problem:** These errors tell you **what** failed, never **why**, and never **how to fix it**.

---

## 3. What Does NOT Exist (The Gaps)

### Gap 1: No Unified Local Stack Setup

A developer who wants to try the full Inji flow (issue → hold → verify a credential) currently needs to:

1. Set up Certify (Docker Compose or manual)
2. Set up eSignet (separate MOSIP repo, separate config)
3. Set up Mimoto/Wallet backend (separate config, requires OIDC client creation, p12 keystore, ngrok)
4. Set up Inji Web (IntelliJ-based)
5. Manually align all configs (URLs, client IDs, keys, ports)
6. Hope nothing is misconfigured

There is no single command, no single config file, and no single guide that makes this work end-to-end.

### Gap 2: No Diagnostic Tool

When something breaks, the developer's only option is to:
- Run `docker logs` and read through hundreds of lines
- Post on community.mosip.io
- Wait 1-5 days for a response

There is no tool that says: *"Here's what's wrong and here's how to fix it."*

### Gap 3: No End-to-End Integration Test

There is no test suite that validates the full flow across components. The QA team, release managers, and country deployers have no automated way to verify that Certify + eSignet + Wallet + Verify are all talking to each other correctly.

### Gap 4: No "First 15 Minutes" Experience

The Mimoto local setup doc alone has **8 manual steps** including:
- "Create a loader_path folder and download kernel-auth-adapter.jar"
- "Add ID providers as issuers in mimoto-issuers-config.json"
- "Start Esignet services and update host references"
- "Create a certs folder, create an OIDC client, add the key to oidckeystore.p12"
- "Update client_id and client_alias in config"
- "Update p12 file password in properties"
- "Use ngrok to expose the service"
- "Replace localhost URLs with ngrok public domain"

This is **not** a first-time-friendly experience.

---

## 4. The Proposal

### Inji Developer Experience (DX) Toolkit

A focused set of improvements that makes it possible for any developer to **set up, integrate, diagnose, and verify** the Inji stack locally — without posting on the community forum.

The toolkit has **four components**, ordered by priority and evidence:

---

### Component 1: Fixed Docker Compose — One-Command Local Setup

**Priority:** #1 (33% of community posts are about this)

A Docker Compose configuration that **actually works on the first try** with zero manual configuration.

#### What It Provides

```bash
$ git clone https://github.com/inji/inji-dev-stack
$ cd inji-dev-stack
$ docker compose up

Starting inji-postgres...     ✅
Starting inji-redis...        ✅
Starting inji-esignet-mock... ✅
Starting inji-certify...      ✅
Starting inji-mimoto...       ✅
Starting inji-web...          ✅
Starting inji-verify...       ✅

🎉 Inji stack is ready!
   Certify:  http://localhost:8080
   eSignet:  http://localhost:8081 (mock)
   Mimoto:   http://localhost:8099
   Inji Web: http://localhost:3000
   Inji Verify: http://localhost:9090

   Open http://localhost:3000 to issue your first credential.
```

#### What Makes It Different from Existing Docker Compose Files

| Aspect | Current State | Proposed |
|--------|-------------|----------|
| **Scope** | Per-component (Certify alone, Verify alone, Mimoto alone) | Full stack — all components together |
| **Config alignment** | Manual — developer must edit URLs, ports, client IDs across 5+ files | Pre-aligned — all cross-references are correct out of the box |
| **Dependencies** | Developer must set up eSignet separately | Mock eSignet included and pre-configured |
| **Health checks** | None or minimal | Proper health checks with dependency ordering |
| **Test data** | None — developer must create issuers, credential types, test subjects | Pre-loaded with a demo issuer ("Test University"), credential type ("University Degree"), and test subject |
| **Manual steps** | 8+ steps (create OIDC client, generate p12, use ngrok, edit configs) | **Zero** — `docker compose up` is the only command |
| **Network** | Each component on its own network | Shared Docker network with correct service discovery |

#### Deliverable

A new repository `inji/inji-dev-stack` containing:
- `docker-compose.yml` — unified stack
- `config/` — pre-aligned configuration files for all components
- `db-init/` — database initialization scripts with demo data
- `README.md` — one-page getting started guide

---

### Component 2: Integration Guide — One Document, Full Flow

**Priority:** #2 (14% of community posts are about integration)

A single, clear document that explains the Inji integration flow with exact configuration values.

#### What It Covers

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│ Inji Web │────▶│ eSignet  │────▶│ Certify  │────▶│  Mimoto  │
│ (Frontend│     │(Auth/OIDC│     │(Issuer   │     │(Wallet   │
│  Portal) │     │  Mock)   │     │ Service) │     │  BFF)    │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
       │                                                    │
       ▼                                                    ▼
┌──────────┐                                         ┌──────────┐
│ Inji     │                                         │ Inji     │
│ Verify   │◀────────────────────────────────────────│ Wallet   │
│(Verifier)│                                         │(Mobile)  │
└──────────┘                                         └──────────┘
```

#### What It Includes

1. **Architecture overview** — what each component does and how they connect
2. **Data flow** — step-by-step of the credential lifecycle (issue → hold → verify)
3. **Configuration reference** — every config file, every key, every value that needs to match
4. **Common failure points** — what breaks and why, with exact error messages and fixes
5. **Troubleshooting decision tree** — "If you see X, check Y, fix Z"
6. **Verification checklist** — how to know your setup is working

#### Deliverable

- One markdown file published on docs.inji.io as "Inji Integration Guide"
- Referenced from every component's README

---

### Component 3: `inji check` — Simple Diagnostic CLI

**Priority:** #3 (implicit in all posts — no one has a self-service diagnostic tool)

A lightweight command-line tool that checks the health of your local Inji environment and tells you exactly what's wrong.

#### What It Does

```bash
$ inji check

🔍 Checking Inji stack health...

✅ PostgreSQL         — Running on :5432  — Inji Certify DB: connected
✅ Redis              — Running on :6379  — Connected
✅ Inji Certify       — Running on :8080  — Health: OK  — Signing keys: valid
✅ eSignet (Mock)     — Running on :8081  — OIDC metadata: valid
⚠️ Mimoto             — Running on :8099  — ⚠️ WARNING: redirect_uri mismatch
                        Your Mimoto redirect_uri: http://localhost:8099/callback
                        eSignet registered: http://localhost:8081/callback
                        → Fix: Set ESIGNET_REDIRECT_URI=http://localhost:8081/callback
✅ Inji Verify        — Running on :9090  — Trust registry: synced
✅ Inji Web           — Running on :3000  — Connected to Certify

📋 Summary: 6/7 components healthy, 1 warning
   Estimated time to fix: 2 minutes
```

#### What It Checks

| Check | What It Detects |
|-------|----------------|
| Service reachability | Is each service running on the expected port? |
| Database connectivity | Can Certify/Mimoto reach PostgreSQL? |
| Redis connectivity | Can services reach Redis? |
| Configuration alignment | Do URLs, client IDs, and ports match across components? |
| Key validity | Are signing keys generated and not expired? |
| Trust registry sync | Does Verify know about Certify's signing key? |
| Port conflicts | Is something else using a required port? |
| Environment variables | Are all required env vars set? |
| Docker health | Are containers healthy, not just running? |

#### Deliverable

- A Python or Node.js CLI tool (`inji check`)
- Published as part of the `inji-dev-stack` repository
- Works on Linux, macOS, and Windows

---

### Component 4: Smart Error Messages

**Priority:** #4 (applied across all Inji repos as code improvements)

Replacing cryptic errors with actionable guidance across the Inji codebase.

#### Before

```json
{
  "error": "invalid_request"
}
```

#### After

```json
{
  "error": "invalid_request",
  "error_description": "The 'redirect_uri' in your token request (http://localhost:3000/callback) does not match any registered redirect URI for client 'inji-wallet'. Registered URIs: [http://localhost:8081/callback]. Check your wallet's ESIGNET_REDIRECT_URI environment variable.",
  "error_hint": "ESIGNET_REDIRECT_MISMATCH",
  "docs_url": "https://docs.inji.io/troubleshoot/redirect-uri-mismatch"
}
```

#### Target Error Paths (Based on Actual Community Posts)

| Current Error | Root Cause (from community posts) | Fix |
|---|---|---|
| `400 invalid_request` | Redirect URI mismatch between Wallet and eSignet | Show expected vs. actual URI |
| `RESIDENT-APP-026 – Api not accessible` | Mimoto can't reach Certify or eSignet | Show which endpoint failed and why |
| `Could not obtain jwks from url` | eSignet JWKS endpoint unreachable or misconfigured | Show the URL being fetched and HTTP status |
| `DataProviderPlugin not found` | Plugin JAR not in loader_path or wrong class name | Show expected path and available plugins |
| `Error connecting to OIDC service` | eSignet not running or wrong URL | Show configured URL and connection status |
| `Private Key Entry is Missing for the alias` | p12 keystore missing or wrong alias | Show expected alias and available aliases |
| `invalid_proof Error in MOSIP Certify API` | Credential proof doesn't match signing key | Show proof validation details |
| `Unable to start mock-identity-system` | Keymanager alias not found | Show missing alias and available aliases |

#### Deliverable

- Pull requests to `inji-certify`, `inji-verify`, `mimoto`, `inji-web`, and `inji-config`
- Improved error responses with `error_description`, `error_hint`, and `docs_url` fields
- Updated troubleshooting documentation

---

## 5. What This Is NOT

| This Proposal Is... | It Is NOT... |
|--------------------|-------------|
| Fixing broken Docker Compose files | Replacing existing CI/CD pipelines |
| Making local setup work out of the box | A production deployment tool |
| Building a diagnostic CLI | An MCP server or AI product |
| Improving error messages | Rewriting existing documentation from scratch |
| Writing one integration guide | A new product or feature |

---

## 6. Who This Helps

| User | Current Experience | With DX Toolkit |
|------|-------------------|-----------------|
| **New developer** trying Inji for the first time | Clones repos → hits errors → posts on forum → waits hours/days → gets partial answer → repeats | `docker compose up` → `inji check` → working in 10 minutes |
| **Contributor** making a change to Certify | Makes change → manually tests → hopes nothing broke → submits PR → reviewer asks for test evidence | Makes change → runs tests → submits PR with test report |
| **Country deployer** evaluating Inji | Spends days figuring out configs → gives up or contacts MOSIP team | One command → working demo → can evaluate immediately |
| **QA team** validating a release | Manual end-to-end testing → error-prone → takes hours | Automated verification → 2 minutes, reproducible |
| **MOSIP support team** answering forum posts | Repeats the same troubleshooting steps for every "can't deploy" post | Points them to `inji check` → self-service |

---

## 7. Success Metrics

| Metric | Current State | Target After 12 Weeks |
|--------|--------------|----------------------|
| Time from "cloned repos" to "issued first credential" | Hours to days (community threads show this) | **Under 15 minutes** (`docker compose up` + demo) |
| Community support posts about Inji local setup/integration | ~57 in the last 2 years (~2.4/month) | **Reduced by 60%** |
| New contributors who successfully run the stack locally | Unknown, but clearly low based on forum activity | **50% increase** in first-time contributors |
| Time to diagnose a setup issue | Hours (forum round-trip) | **Under 2 minutes** (`inji check`) |
| End-to-end test coverage | None (no cross-component tests exist) | **Automated test** for full credential lifecycle |

---

## 8. Implementation Plan (12 Weeks)

### Phase 1: Unified Docker Compose (Weeks 1–3)

**Goal:** `docker compose up` starts all Inji components with pre-aligned configs.

**Tasks:**
- Create `inji/inji-dev-stack` repository
- Write `docker-compose.yml` that starts: PostgreSQL, Redis, eSignet (mock), Certify, Mimoto, Inji Web, Inji Verify
- Pre-align all configuration files (URLs, ports, client IDs, keys)
- Add health checks and dependency ordering
- Create database initialization scripts with demo data (test issuer, credential type, test subject)
- Test on Linux, macOS, and Windows (Docker Desktop)

**Deliverable:** `docker compose up` works end-to-end. A developer can visit `http://localhost:3000` and issue a credential.

**Risks & Mitigations:**
| Risk | Mitigation |
|------|-----------|
| eSignet is in MOSIP org, not Inji org — coordination needed | Use eSignet mock services (already available) for dev stack |
| Different Docker versions behave differently | Test on Docker Desktop 24+, Docker Compose v2.25+, across 3 OSes |
| Resource-heavy for a single machine | Make components optional via Docker Compose profiles |

---

### Phase 2: Integration Guide (Weeks 4–5)

**Goal:** One document that explains the full Inji flow with exact configs.

**Tasks:**
- Document the architecture and data flow (issue → hold → verify)
- Document every configuration file, every key, every value that needs to match
- Document common failure points with exact error messages and fixes
- Create a troubleshooting decision tree
- Create a verification checklist

**Deliverable:** One markdown file published on docs.inji.io. Referenced from every component's README.

**Risks & Mitigations:**
| Risk | Mitigation |
|------|-----------|
| Content may become outdated as Inji evolves | Keep it focused on integration flow, not version-specific details |
| May need review from Inji team | Share draft early, iterate based on feedback |

---

### Phase 3: `inji check` CLI (Weeks 6–8)

**Goal:** A diagnostic tool that tells developers exactly what's wrong.

**Tasks:**
- Build the CLI tool (Python or Node.js)
- Implement all checks: service reachability, DB connectivity, config alignment, key validity, trust registry sync, port conflicts, env vars
- Add actionable fix suggestions for each check
- Test against known failure modes from community posts
- Package as a simple installable tool (pip install or npx)

**Deliverable:** `inji check` diagnoses 90% of common setup issues with clear fix instructions.

**Risks & Mitigations:**
| Risk | Mitigation |
|------|-----------|
| Different deployment configs produce different health endpoints | Support configurable endpoints via a config file |
| Some checks require API access that may not exist | Add health endpoints where missing (PRs to Inji repos) |

---

### Phase 4: Smart Error Messages (Weeks 9–10)

**Goal:** Replace cryptic errors with actionable guidance.

**Tasks:**
- Identify top 10 error paths from community posts
- Rewrite error responses with `error_description`, `error_hint`, and `docs_url`
- Apply across Certify, Mimoto, Verify, and Inji Web
- Update troubleshooting documentation

**Deliverable:** All common errors are self-diagnosable. A developer seeing "400 invalid_request" now sees exactly which field is invalid and how to fix it.

**Risks & Mitigations:**
| Risk | Mitigation |
|------|-----------|
| Code changes required in multiple repos | Start with the top 5 errors, expand based on feedback |
| May need Inji team review and approval | Scope as standalone PRs, easy to review |

---

### Phase 5: Polish & Documentation (Weeks 11–12)

**Goal:** Complete, documented, announced toolkit.

**Tasks:**
- Write comprehensive getting-started guide
- Record a demo video of the end-to-end flow
- Add troubleshooting section to each repo's README
- Publish announcement on community.mosip.io
- Archive the 57+ community posts as "resolved by this toolkit"

**Deliverable:** Complete DX toolkit with documentation and community announcement.

---

## 9. Why This Matters for MOSIP

### The Evidence

- **57 posts in 2 years** — that's ~2.4 posts per month, every single month
- The problem hasn't improved despite 20+ Inji releases (Certify v0.8→v0.14, Wallet v0.11→v0.22, Verify v0.8→v0.17)
- The most recent post (1 day ago) is about the **exact same problem** as the oldest post (2 years ago)
- Each post has 2–30 replies, meaning the MOSIP team spends significant time answering repeat questions
- Inji is MOSIP's bet on the future of digital identity — W3C Verifiable Credentials, OpenID4VCI, decentralized trust

### The Impact

If developers can't set up Inji locally, can't integrate it, and can't debug it when it breaks, **the ecosystem won't grow**. Every developer who gives up because they can't get Docker Compose to work is a potential issuer, verifier, wallet developer, or contributor lost.

This toolkit removes the friction between *"I'm interested in Inji"* and *"I'm building with Inji."* It turns hours of forum threads into minutes of self-service diagnosis. It makes the first experience with Inji a good one.

---

## 10. Appendix: Methodology

This proposal is based on analysis of:

| Source | What Was Analyzed |
|--------|------------------|
| **community.mosip.io** | All 200+ community forum posts from April 2024 – April 2026, categorized by topic and Inji-relevance |
| **github.com/inji** | All 27 repositories — file structure, READMEs, Docker Compose files, Helm charts, test suites, CI/CD workflows |
| **github.com/mosip** | 128 repositories — DevOps pipeline (Kattu, infra, k8s-infra, mosip-helm, mosip-config) |
| **docs.inji.io** | All local setup guides, deployment documentation, resources pages |
| **docs.mosip.io** | V3 deployment guides, on-prem installation guidelines, AWS installation guidelines |
| **MOSIP Confluence wiki** | 12,316 pages across 101 spaces — DevOps procedures, design documents, release tracking, troubleshooting guides |

---

*Proposal prepared based on comprehensive analysis of the MOSIP/Inji ecosystem, community support history, and developer experience gaps.*
