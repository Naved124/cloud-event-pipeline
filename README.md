# Cloud Event Pipeline & Server Health Logger

A production-grade, containerized Go backend integrated with a PostgreSQL database, automatically provisioned on AWS via Infrastructure as Code (IaC), and deployed through a continuous deployment (CD) pipeline.
> This project's focus is infrastructure and deployment automation — Docker, Terraform, CI/CD, and secrets handling — rather than application features. The API itself is intentionally minimal (health check + a simple events table).
---

## Architecture & Tech Stack

* **Backend & API:** Go (Golang), standard library REST routing
* **Database:** PostgreSQL (containerized with volume persistence)
* **Containerization:** Docker & Docker Compose (multi-container orchestration)
* **Infrastructure as Code (IaC):** Terraform (AWS EC2 `t3.micro`, Security Groups, IAM Role/Instance Profile, Key Pair) — provisioned in `ap-south-1`
* **CI/CD Automation:** GitHub Actions with AWS Systems Manager (SSM) Run Command — pushes trigger a tag-targeted deploy with no inbound SSH required
* **Cloud Provider:** AWS

---

## Project Structure

```text
cloud-event-pipeline/
├── .github/
│   └── workflows/
│       └── deploy.yml       # GitHub Actions CD pipeline (SSM-based)
├── terraform/
│   └── main.tf              # AWS infrastructure blueprint
├── Dockerfile               # Multi-stage build for Go API
├── docker-compose.yml       # Multi-container service stack (API + DB)
├── go.mod                   # Go module dependencies
├── main.go                  # REST API implementation & DB logic
└── .gitignore                # Excludes local state, secrets, and binary caches
```

---

## Deployment Architecture: Decisions & Reasoning

**Why AWS Systems Manager instead of direct SSH.**
The pipeline originally deployed via SCP + SSH from GitHub Actions. After locking the EC2 security group's SSH access down to a single trusted IP — a deliberate hardening step — GitHub's hosted runners (which use rotating IPs) could no longer reach the instance, breaking deploys.

Rather than reopening SSH to the world, the deploy mechanism was switched to AWS Systems Manager (SSM) Run Command. GitHub Actions authenticates to AWS via a narrowly-scoped IAM user, and the instance's own IAM role lets its pre-installed SSM agent pull the latest code and redeploy containers — no inbound SSH involved. Port 22 stays locked to a single IP for manual debugging access only; it is no longer required for CI/CD to function.

**Why the deploy targets by tag, not instance ID.**
The SSM command targets `Key=tag:Name,Values=CloudEventPipelineServer` rather than a hardcoded instance ID. This means the pipeline keeps working automatically even if the underlying EC2 instance is destroyed and recreated (e.g. during infrastructure iteration) — no secrets need updating afterward, as long as the new instance carries the same tag. The IAM policy is scoped the same way, using a `ssm:resourceTag/Name` condition instead of a fixed instance ARN.

**Secrets management.**
Database credentials (`DB_USER`, `DB_PASSWORD`, `DB_NAME`) and the API key are stored in AWS Systems Manager Parameter Store (`SecureString`, under `/cloud-event-pipeline/*`), not as GitHub Actions secrets. At deploy time, the EC2 instance's IAM role — scoped to `ssm:GetParameter` on that path only — pulls the values itself and writes them into a local `.env` file. This means secret values never appear in the SSM command text sent from GitHub Actions, only parameter *names* do, keeping them out of CloudTrail/SSM command history. Locally, the same variables live in a gitignored `.env` file in the project root.

**Other hardening decisions.**
- PostgreSQL's port is not published to the host — the API reaches it only over Docker's internal network.
- The EC2 key pair's public key is read dynamically via Terraform's `file()` function rather than hardcoded, so rotating keys never requires editing `main.tf`.
- The `/events` API key check uses `crypto/subtle.ConstantTimeCompare` rather than a plain `!=` string comparison, removing a timing side-channel on the key value.
- Errors from the startup `CREATE TABLE IF NOT EXISTS` are now logged instead of silently discarded, so a schema-creation failure shows up in the container logs immediately rather than as a confusing `500` on the first `/events` call.

---

## Local Development

Create a `.env` file in the project root (not committed — see `.gitignore`):

DB_USER=appuser
DB_PASSWORD=your-local-password
DB_NAME=eventdb
API_KEY=your-local-api-key

Generate an API key with `openssl rand -hex 32`.

Then:
```bash
docker compose up --build
```
Check `http://localhost:8080/health` — should report `"database": "connected"`.

`/events` requires an API key:
```bash
curl -H "X-API-Key: your-local-api-key" http://localhost:8080/events

curl -X POST -H "X-API-Key: your-local-api-key" -H "Content-Type: application/json" \
  -d '{"payload":"hello"}' http://localhost:8080/events
```