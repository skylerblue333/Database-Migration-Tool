# Database-Migration-Tool

## Overview
A robust CLI tool written in Go for managing and executing database schema migrations in enterprise environments.

## Quick Start (1-Click Build)

```bash
git clone https://github.com/skylerblue333/Database-Migration-Tool.git
cd Database-Migration-Tool
go build -o db-migrate main.go
./db-migrate status
./db-migrate up
```

## Features
- Tracks applied vs pending migrations
- Deterministic sequential execution
- Easy integration into CI/CD pipelines
