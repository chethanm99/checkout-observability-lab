# From log.Printf to Trace Context: Go Checkout Observability Lab

A Go-based synthetic checkout service built to explore **OpenTelemetry tracing**, **high-cardinality telemetry**, and **SigNoz-powered debugging workflows**.

This project was created as part of the **Agents of SigNoz Hackathon** to understand how connected telemetry can reduce the time required to investigate production failures.

The experiment simulates a checkout service that generates realistic traffic, injects controlled failures, and exports distributed traces to SigNoz using OpenTelemetry.

---

## Why This Project?

During production incidents, engineers often jump between:

- Metrics dashboards
- Application logs
- Distributed traces

The goal of this experiment was to test whether a trace-first workflow could keep enough debugging context together to quickly answer:

- Which checkout request failed?
- Which user was affected?
- Which product was involved?
- Which downstream operation caused the failure?

---

# Architecture

```
                    Go Checkout Simulator

                             |
                             |
                             | OTLP gRPC
                             |
                             v

              OpenTelemetry Collector

                             |
                             |
                             v

                         ClickHouse

                             |
                             |
                             v

                         SigNoz UI
```

## How It Works

1. The Go application generates synthetic checkout requests.
2. OpenTelemetry SDK creates traces and spans.
3. Telemetry is exported through OTLP gRPC.
4. SigNoz Collector receives and processes telemetry.
5. ClickHouse stores the trace data.
6. SigNoz UI provides trace exploration and debugging.

---

# Features

## Synthetic Checkout Traffic

The simulator creates:

- HTTP checkout requests
- Database child spans
- Dynamic user identifiers
- Dynamic product identifiers
- Simulated database failures

Example workflow:

```
HTTP POST /checkout

        |
        |
        v

SQL SELECT clusters_meta
```

---

## High-Cardinality Testing

Every request generates unique attributes:

Example:

```
user.id:
usr-000630-691

product.id:
prod-d8ce200a36bb3a17
```

These values are attached to traces instead of metrics.

This demonstrates why high-cardinality information is useful during investigation but dangerous as metric labels.

---

# Failure Injection

Every 15th request intentionally fails:

```
database connection timeout on pool allocation
```

The failure is recorded using OpenTelemetry:

- Span status
- Exception details
- Structured span events

Example:

```go
dbSpan.RecordError(err)

dbSpan.SetStatus(
    codes.Error,
    err.Error(),
)
```

---

# Project Structure

```
.
├── cmd
│   └── main.go
│
├── internal
│   └── telemetry
│       └── tracer.go
│
├── go.mod
├── go.sum
└── README.md
```

---

# Running the Project

## Prerequisites

Required:

- Go
- Docker
- SigNoz running locally

Tested environment:

```
OS:
Ubuntu running inside WSL2

Container Runtime:
Docker Desktop

Language:
Go

Telemetry:
OpenTelemetry Go SDK

Observability:
SigNoz

Storage:
ClickHouse
```

---

# Start SigNoz

Deploy SigNoz locally using your preferred deployment method.

The traffic generator expects:

OTLP gRPC collector:

```
localhost:4317
```

SigNoz UI:

```
localhost:8080
```

Verify containers:

```bash
docker ps
```

Expected services:

```
signoz
signoz-otel-collector
clickhouse
postgres
clickhouse-keeper
```

---

# Run Traffic Generator

Clone repository:

```bash
git clone https://github.com/<your-username>/checkout-observability-lab.git

cd checkout-observability-lab
```

Run:

```bash
go run cmd/main.go
```

Example output:

```
Connecting to SigNoz OpenTelemetry Collector...

OpenTelemetry pipeline successfully wired to SigNoz backend!

Starting synthetic traffic simulation.

[ERROR] Request ID 630 failed
database connection timeout on pool allocation
```

---

# Debugging Workflow

## Before Adding Trace Context

The debugging flow looked like:

```
Find failing metric

        ↓

Search application logs

        ↓

Find request identifier

        ↓

Locate trace

        ↓

Manually connect information
```

---

## After Adding Structured Trace Events

The workflow became:

```
Open failed checkout trace

        ↓

Inspect checkout span

        ↓

View failure event

        ↓

Open database span

        ↓

Understand root cause
```

---

# Important Discovery

Initially, the application used:

```go
log.Printf()
```

for failures.

The assumption was:

> If traces are exported through OpenTelemetry, application logs should automatically appear alongside them.

That was incorrect.

Logs and traces are separate telemetry signals.

The application needed explicit instrumentation to attach debugging context to traces.

The solution was using:

- Span attributes
- Span events
- Recorded exceptions

---

# SigNoz Features Used

This experiment used:

## Service Overview

Used to observe:

- Request rate
- Latency
- Error rate

## Trace Explorer

Used to:

- Filter failed checkout requests
- Inspect distributed spans
- Follow parent-child relationships

## Span Attributes

Used to search debugging information:

Example:

```
user.id
product.id
request.id
```

## Span Events

Used to attach business context:

Example:

```
checkout_failure

error.message:
database connection timeout on pool allocation
```

---

# Example Trace Information

Failed checkout span:

```
checkout_failure

error.message:
database connection timeout on pool allocation

user.id:
usr-000630-691

product.id:
prod-d8ce200a36bb3a17
```

Database span:

```
exception.message:
database connection timeout on pool allocation

exception.type:
*errors.errorString
```

---

# Lessons Learned

## 1. Instrumentation is only the beginning

Adding OpenTelemetry instrumentation is not enough.

The important question is:

> What information will I need when debugging a failure?

That context should be added while building the application.

---

## 2. Observability is about reducing investigation time

The goal is not collecting unlimited telemetry.

The goal is reducing the path from:

```
Something failed
```

to:

```
I know why it failed
```

---

# Why SigNoz?

By using SigNoz, I was able to demonstrate how high-cardinality attributes like user.id and product.id can be indexed and searched efficiently—a task that would be prohibitively expensive or impossible using traditional log-based indexing.

---

# Future Improvements

Possible next steps:

- Add metrics instrumentation
- Add OpenTelemetry log correlation
- Build an AI SRE agent using telemetry data
- Query ClickHouse telemetry through MCP
- Automate failure investigation workflows

---

# Related Blog Post

Full write-up:

https://dev.to/chethanblgs99/from-logprintf-to-trace-context-what-i-learned-debugging-a-go-checkout-service-with-opentelemetry-1nlh

---

# Technologies Used

- Go
- OpenTelemetry
- SigNoz
- ClickHouse
- Docker
- OTLP gRPC
- WSL2

---

# License

MIT License
