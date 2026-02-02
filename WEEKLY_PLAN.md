# Detailed Weekly Plan: Mini Lambda (FaaS)

This document provides a granular breakdown of the 8-week internship project. It includes specific learning resources, task divisions for Mentors and Mentees, and guidelines on how work should be assigned.

## General Workflow
*   **Monday:** Sprint Planning. Review the week's goals. Mentor provides a high-level overview of concepts. (Mentor's 1st weekly Sync, 30 mins)
*   **Tue-Thu:** Implementation. Async communication over chat to flag blockers. (Mentor's buddy can help with this)
*   **Friday:** Code Review & Demo. Show what was built to the mentor. (Mentor's 2nd Weekly Sync, 30 mins)

---

## Phase 1: Foundation (Weeks 1-2)

### Week 1: Project Alignment & FaaS Fundamentals
**Goal:** Understand the FaaS landscape, define the project architecture, and control the Docker Daemon programmatically using Go.

*   **Learning Resources:**
    *   [How AWS Lambda Works](https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtime-environment.html)
    *   [Firecracker MicroVMs (Advanced Reading)](https://firecracker-microvm.github.io/)
    *   [Cloudflare Workers Architecture](https://developers.cloudflare.com/workers/learning/how-workers-works/)
    *   [Docker Engine API SDK for Go](https://pkg.go.dev/github.com/docker/docker/client)
    *   [Understanding Cold Starts](https://mikhail.io/serverless/coldstarts/aws/)

*   **Tasks:**
    *   **Mentor:**
        *   Deep dive into the Problem Statement: What is FaaS? How is it different from PaaS?
        *   Explain cold starts, warm pools, and the execution model.
        *   Discuss container vs. isolate-based approaches (Docker vs. V8 Isolates).
        *   Verify Mentee's machine environment (Go + Docker Desktop/Engine installed).
    *   **Mentee:**
        *   **Task 1.1: Research.** Research how AWS Lambda handles function invocations. What happens during a cold start vs warm start? Write a 1-page summary.
        *   **Task 1.2: Architecture Design.** Create a detailed sequence diagram showing the flow from `lambda deploy` to function invocation.
        *   **Task 1.3: Design Pitch.** Present the proposed approach to the mentor. Discuss the Gateway, Runtime Manager, and CLI components.
        *   **Task 1.4: SDK Basics.** Initialize Go module (`go mod init`) and write a script to connect to the Docker client and print the Docker version.
        *   **Task 1.5: Container Lifecycle.** Write functions to create, start, stop, and remove containers. Measure startup times.

*   **Task Assignment:**
    *   Tasks 1.1 - 1.3: Collaborative. Both mentees work together on the research and design document.
    *   Tasks 1.4 - 1.5: Split individually. One mentee focuses on "Container Creation & Startup", the other on "Container Inspection & Cleanup".

*   **Deliverable:** A Go binary that spins up an Alpine container, runs a command, and reports the total latency.

---

### Week 2: Function Packaging & Build Pipeline
**Goal:** Implement the packaging pipeline—the logic that turns function source code into a Docker Image.

*   **Learning Resources:**
    *   [Go `archive/tar` Package](https://pkg.go.dev/archive/tar)
    *   [Docker ImageBuild API Reference](https://docs.docker.com/engine/api/v1.43/#tag/Image/operation/ImageBuild)
    *   [Building Minimal Docker Images](https://blog.codeship.com/building-minimal-docker-containers-for-go-applications/)
    *   [Multi-stage Docker Builds](https://docs.docker.com/build/building/multi-stage/)

*   **Tasks:**
    *   **Mentor:**
        *   Explain "Build Context" in Docker: Why do we need to send the whole folder?
        *   Review the Tar creation logic (a common source of bugs: ensuring relative paths are correct).
        *   Discuss base image strategies: Alpine vs. Distroless vs. Scratch.
    *   **Mentee:**
        *   **Task 2.1: Sample Functions.** Create 3 sample functions:
            *   `hello`: Returns `{"message": "Hello, World!"}`
            *   `echo`: Returns the request body back
            *   `calculator`: Performs basic math operations
            Each should have a `Dockerfile` and a `main.go`.
        *   **Task 2.2: The Packager.** Implement a Go function `PackageFunction(path string) (io.Reader, error)` that creates a tar archive of the function directory.
        *   **Task 2.3: Base Image.** Create a minimal base image for Go functions. It should include only what's needed to run a Go binary.
        *   **Task 2.4: The Builder.** Connect the packaging logic to Docker SDK's `ImageBuild()` function.
            *   *Success Criteria:* When the builder runs, a new image appears in `docker images`.

*   **Task Assignment:**
    *   Mentee A: **The Packager.** Focuses on File I/O, recursion, and creating a valid Tar stream.
    *   Mentee B: **The Builder.** Focuses on the Docker SDK integration and base image optimization.
    *   **Integration:** On Thursday, connect Mentee A's "Packager" into Mentee B's "Builder".

*   **Deliverable:** A Go function that takes a function directory path and outputs a Docker Image ID.

---

## Phase 2: Execution Engine (Weeks 3-4)

### Week 3: HTTP Gateway
**Goal:** Route incoming HTTP requests to the correct function container.

*   **Learning Resources:**
    *   [Go httputil.ReverseProxy](https://pkg.go.dev/net/http/httputil#ReverseProxy)
    *   [Writing a Reverse Proxy in Go](https://blog.joshsoftware.com/2021/05/25/simple-and-powerful-reverseproxy-in-go/)
    *   [HTTP Request Lifecycle](https://developer.mozilla.org/en-US/docs/Web/HTTP/Session)
    *   [Context in Go](https://go.dev/blog/context)

*   **Tasks:**
    *   **Mentor:**
        *   Explain reverse proxy patterns and how routing based on Host header works.
        *   Discuss request/response transformation patterns.
        *   Review timeout and error handling strategies.
    *   **Mentee:**
        *   **Task 3.1: Basic Gateway.** Implement an HTTP server that listens on port 80 and extracts the function name from the Host header (e.g., `greet.localhost` → function "greet").
        *   **Task 3.2: Container Routing.** Look up the function's container IP from the database (or in-memory map for now) and forward the request.
        *   **Task 3.3: Context Injection.** Ensure query parameters, headers, and body are properly forwarded to the function container.
        *   **Task 3.4: Timeout Handling.** Implement a configurable timeout (default 30s). If the function doesn't respond, return 504 Gateway Timeout.
        *   **Task 3.5: Error Handling.** Handle cases: function not found (404), function crashed (500), function timeout (504).

*   **Task Assignment:**
    *   Mentee A: **The Router.** Tasks 3.1, 3.2 - Focuses on request routing and container lookup.
    *   Mentee B: **The Handler.** Tasks 3.3, 3.4, 3.5 - Focuses on request transformation and error handling.
    *   **Integration:** Combine the router and handler into a complete gateway.

*   **Deliverable:** HTTP requests to `func-name.localhost` invoke the corresponding function container and return the response.

---

### Week 4: Cloud Deployment & CI/CD
**Goal:** Deploy the platform to a cloud VM and set up automated deployments.

*   **Learning Resources:**
    *   [GitHub Actions for Go](https://docs.github.com/en/actions/automating-builds-and-tests/building-and-testing-go)
    *   [SCP File Transfer](https://linuxize.com/post/how-to-use-scp-command-to-securely-transfer-files/)
    *   [Systemd Services](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
    *   [nip.io for Wildcard DNS](https://nip.io/)

*   **Tasks:**
    *   **Mentor:**
        *   Provide Cloud VM access credentials.
        *   Review the CI/CD pipeline configuration.
        *   Help troubleshoot networking/firewall issues.
    *   **Mentee:**
        *   **Task 4.1: VM Setup.** SSH into the VM, install Docker and Go. Verify the environment.
        *   **Task 4.2: Systemd Service.** Create a systemd service file that runs the Gateway binary and restarts on failure.
        *   **Task 4.3: CI/CD Pipeline.** Create a GitHub Action that:
            1.  Builds the Go binary on push to main
            2.  SCPs the binary to the VM
            3.  Restarts the systemd service via SSH
        *   **Task 4.4: DNS Configuration.** Configure wildcard DNS using nip.io (e.g., `*.vm-ip.nip.io`).
        *   **Task 4.5: End-to-End Test.** Deploy a function from your laptop to the cloud VM and invoke it via HTTP.

*   **Task Assignment:**
    *   Mentee A: **Infrastructure.** Tasks 4.1, 4.2, 4.4 - Focuses on VM setup and DNS.
    *   Mentee B: **CI/CD.** Tasks 4.3, 4.5 - Focuses on GitHub Actions and deployment pipeline.
    *   **Integration:** Merge pushes should result in live deployments.

*   **Deliverable:** A working CI/CD pipeline where git push deploys to the cloud VM.

---

## Phase 3: Event System (Weeks 5-6)

### Week 5: Scheduled Triggers
**Goal:** Allow functions to run on a schedule (cron-style).

*   **Learning Resources:**
    *   [Cron Expression Syntax](https://crontab.guru/)
    *   [robfig/cron Go Library](https://pkg.go.dev/github.com/robfig/cron/v3)
    *   [Task Queues and Workers](https://blog.golang.org/pipelines)
    *   [Handling Concurrent Executions](https://gobyexample.com/worker-pools)

*   **Tasks:**
    *   **Mentor:**
        *   Explain cron expression syntax and common patterns.
        *   Discuss strategies for overlapping executions: skip vs. queue vs. allow parallel.
        *   Review concurrency handling in Go (goroutines, channels, mutexes).
    *   **Mentee:**
        *   **Task 5.1: Scheduler Core.** Integrate `robfig/cron` library. Create a scheduler that can trigger function invocations.
        *   **Task 5.2: CLI Command.** Implement `lambda schedule <function-name> --cron "<expression>"` command.
        *   **Task 5.3: Persistence.** Store scheduled jobs in SQLite so they survive restarts.
        *   **Task 5.4: Overlap Policy.** Implement configurable behavior: skip if function is still running OR allow parallel executions.
        *   **Task 5.5: Async Invocation.** Implement async invocation endpoint that returns immediately and processes in background.

*   **Task Assignment:**
    *   Mentee A: **Scheduler.** Tasks 5.1, 5.3, 5.4 - Focuses on the cron scheduler and persistence.
    *   Mentee B: **CLI & Async.** Tasks 5.2, 5.5 - Focuses on CLI integration and async processing.

*   **Deliverable:** Functions can be scheduled to run at specific intervals using cron expressions.

---

### Week 6: State & Versioning
**Goal:** Persist function metadata and support function versioning with rollback.

*   **Learning Resources:**
    *   [Go database/sql Package](https://pkg.go.dev/database/sql)
    *   [SQLite with Go (go-sqlite3)](https://github.com/mattn/go-sqlite3)
    *   [Semantic Versioning](https://semver.org/)
    *   [Database Migrations](https://github.com/golang-migrate/migrate)

*   **Tasks:**
    *   **Mentor:**
        *   Review database schema design.
        *   Discuss versioning strategies: incremental vs. semantic vs. hash-based.
        *   Review rollback implementation for edge cases.
    *   **Mentee:**
        *   **Task 6.1: Database Schema.** Design and implement SQLite schema for:
            *   Functions (name, current_version, created_at, updated_at)
            *   Versions (function_id, version_number, image_id, created_at)
            *   Schedules (function_id, cron_expression, enabled)
        *   **Task 6.2: Versioning Logic.** Each `lambda deploy` creates a new version. Keep the last N versions (configurable, default 5).
        *   **Task 6.3: CLI Commands.** Implement:
            *   `lambda versions <function-name>` - List all versions
            *   `lambda rollback <function-name> [version]` - Rollback to specific or previous version
        *   **Task 6.4: Atomic Deployments.** Ensure the gateway switches to new version only after successful deployment.
        *   **Task 6.5: Cleanup.** Implement automatic cleanup of old images/containers when versions are pruned.

*   **Task Assignment:**
    *   Mentee A: **Database.** Tasks 6.1, 6.2 - Focuses on schema design and versioning logic.
    *   Mentee B: **CLI & Cleanup.** Tasks 6.3, 6.4, 6.5 - Focuses on CLI commands and deployment atomicity.

*   **Deliverable:** Functions can be versioned, listed, and rolled back to previous versions.

---

## Phase 4: Polish (Weeks 7-8)

### Week 7: Observability & CLI Completion
**Goal:** Complete the CLI with all commands and add observability features.

*   **Learning Resources:**
    *   [Cobra CLI Framework](https://github.com/spf13/cobra)
    *   [Docker Container Logs API](https://docs.docker.com/engine/api/v1.43/#tag/Container/operation/ContainerLogs)
    *   [Streaming Logs in Go](https://medium.com/@matryer/writing-a-http-streaming-endpoint-in-go-c72ab2f86d0b)
    *   [API Key Authentication](https://www.alexedwards.net/blog/how-to-rate-limit-http-requests)

*   **Tasks:**
    *   **Mentor:**
        *   Review CLI UX and error messages.
        *   Discuss logging best practices and structured logging.
        *   Review authentication implementation for security issues.
    *   **Mentee:**
        *   **Task 7.1: CLI Completion.** Ensure all commands are implemented and consistent:
            *   `lambda deploy <path> --name <name>`
            *   `lambda invoke <name> [--data <json>]`
            *   `lambda list`
            *   `lambda delete <name>`
            *   `lambda logs <name> [--follow]`
            *   `lambda versions <name>`
            *   `lambda rollback <name> [version]`
            *   `lambda schedule <name> --cron <expr>`
            *   `lambda config set-host <url>`
        *   **Task 7.2: Log Streaming.** Implement real-time log streaming from containers using Docker's logs API.
        *   **Task 7.3: Basic Metrics.** Track per-function:
            *   Total invocations
            *   Average duration
            *   Error count
        *   **Task 7.4: Authentication.** Add API key authentication. Users must set `lambda config set-key <api-key>` to deploy.
        *   **Task 7.5: Error Messages.** Ensure all errors are user-friendly with actionable suggestions.

*   **Task Assignment:**
    *   Mentee A: **CLI & Auth.** Tasks 7.1, 7.4, 7.5 - Focuses on CLI polish and authentication.
    *   Mentee B: **Observability.** Tasks 7.2, 7.3 - Focuses on logging and metrics.

*   **Deliverable:** A complete, polished CLI with logging and basic metrics.

---

### Week 8: Documentation & Demo
**Goal:** Finalize the project and prepare for the demo presentation.

*   **Learning Resources:**
    *   [Writing Good Documentation](https://documentation.divio.com/)
    *   [Creating Effective Technical Demos](https://www.hashicorp.com/resources/how-to-give-a-great-technical-demo)

*   **Tasks:**
    *   **Mentor:**
        *   Final code review - ensure code quality and consistency.
        *   Help prepare demo script and talking points.
        *   Provide feedback on documentation.
    *   **Mentee:**
        *   **Task 8.1: Code Cleanup.** Remove dead code, add comments where needed, ensure consistent formatting.
        *   **Task 8.2: User Guide.** Write documentation covering:
            *   Installation instructions
            *   Quick start guide
            *   CLI command reference
            *   Architecture overview
        *   **Task 8.3: Demo Preparation.** Prepare a live demo that showcases:
            1.  Deploying a new function from scratch
            2.  Invoking it via HTTP
            3.  Viewing logs
            4.  Deploying a new version
            5.  Rolling back
            6.  Scheduling a function
        *   **Task 8.4: The Demo.** Present the project to stakeholders.

*   **Task Assignment:**
    *   Collaborative. Both mentees work together on documentation and demo preparation.
    *   Split the demo: Each mentee presents different features.

*   **Deliverable:** A polished project with documentation and a successful live demo.

---

## Success Criteria (End of Internship)

### Technical
- [ ] Functions can be deployed via CLI
- [ ] Functions can be invoked via HTTP
- [ ] Functions can be scheduled with cron expressions
- [ ] Function versions can be managed and rolled back
- [ ] Logs can be streamed in real-time
- [ ] Platform runs on cloud VM with CI/CD

### Process
- [ ] All code is reviewed and merged via PRs
- [ ] Daily standups maintained throughout
- [ ] Documentation is complete and accurate
- [ ] Successful live demo completed

### Learning
- [ ] Both mentees can explain every component they built
- [ ] Understanding of Docker internals and Go SDK
- [ ] Experience with cloud deployment and CI/CD
- [ ] Collaboration and pair programming practiced
