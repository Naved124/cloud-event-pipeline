# Cloud Event Pipeline & Server Health Logger

A production-grade, containerized Go backend integrated with a PostgreSQL database, automatically provisioned on AWS via Infrastructure as Code (IaC), and deployed through a continuous deployment (CD) pipeline.

---

## Architecture & Tech Stack

* **Backend & API:** Go (Golang), standard library REST routing
* **Database:** PostgreSQL (containerized with volume persistence)
* **Containerization:** Docker & Docker Compose (Multi-container orchestration)
* **Infrastructure as Code (IaC):** Terraform (AWS EC2 `t3.micro`, Security Groups, Key Pairs)
* **CI/CD Automation:** GitHub Actions (Automated SCP file transfer, remote SSH execution, and zero-downtime container redeployment)
* **Cloud Provider:** AWS (Free Tier eligible)

---
## Project Structure

```text
cloud-event-pipeline/
├── .github/
│   └── workflows/
│       └── deploy.yml       # GitHub Actions CD pipeline
├── terraform/
│   └── main.tf              # AWS infrastructure blueprint
├── Dockerfile               # Multi-stage build for Go API
├── docker-compose.yml       # Multi-container service stack (API + DB)
├── go.mod                   # Go module dependencies
├── main.go                  # REST API implementation & DB logic
└── .gitignore               # Excludes local states and binary caches
