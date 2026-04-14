package futureproof

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DeploymentGenerator struct{}

func NewDeploymentGenerator() *DeploymentGenerator {
	return &DeploymentGenerator{}
}

func (g *DeploymentGenerator) GenerateDockerfile(language, framework string) string {
	templates := map[string]string{
		"go": `FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /siby ./cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /siby /app/
EXPOSE 8080
ENTRYPOINT ["/app/siby"]`,

		"node": `FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
EXPOSE 3000
CMD ["node", "dist/index.js"]`,

		"python": `FROM python:3.12-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .
EXPOSE 8000
CMD ["python", "main.py"]`,

		"rust": `FROM rust:1.75-alpine AS builder

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release && rm -rf src

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/target/release/siby /app/
EXPOSE 8080
CMD ["/app/siby"]`,
	}

	if tmpl, ok := templates[language]; ok {
		return tmpl
	}
	return templates["go"]
}

func (g *DeploymentGenerator) GenerateGitHubActions(language, projectName string) string {
	return fmt.Sprintf(`name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  PROJECT_NAME: %s

jobs:
  lint:
    name: Lint & Format
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup
        uses: %s/setup-action@v2
      - name: Lint
        run: |
          make lint
          make fmt-check

  test:
    name: Tests
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - uses: actions/checkout@v4
      - name: Setup
        uses: %s/setup-action@v2
      - name: Test
        run: make test-coverage
      - name: Upload Coverage
        uses: codecov/codecov-action@v3

  build:
    name: Build
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - name: Setup
        uses: %s/setup-action@v2
      - name: Build
        run: make build
      - name: Upload Artifact
        uses: actions/upload-artifact@v4
        with:
          name: binaries
          path: bin/*

  deploy:
    name: Deploy
    runs-on: ubuntu-latest
    needs: build
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to Cloud
        run: |
          echo "Deploying $PROJECT_NAME..."
          make deploy

  security:
    name: Security Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Security Scan
        run: make security-scan
`, projectName, g.getSetupAction(language), g.getSetupAction(language), g.getSetupAction(language))
}

func (g *DeploymentGenerator) getSetupAction(language string) string {
	switch language {
	case "go":
		return " actions/setup-go"
	case "node":
		return " actions/setup-node"
	case "python":
		return " actions/setup-python"
	case "rust":
		return " dtolnay/rust-toolchain"
	default:
		return " actions/setup-go"
	}
}

func (g *DeploymentGenerator) GenerateVercelConfig() string {
	return `{
  "buildCommand": "npm run build",
  "outputDirectory": "dist",
  "framework": "nextjs",
  "regions": ["fra1", "cdg"],
  "env": {
    "NODE_ENV": "production"
  },
  "headers": [
    {
      "source": "/api/(.*)",
      "headers": [
        { "key": "X-Content-Type-Options", "value": "nosniff" },
        { "key": "X-Frame-Options", "value": "DENY" },
        { "key": "X-XSS-Protection", "value": "1; mode=block" }
      ]
    }
  ]
}`
}

func (g *DeploymentGenerator) GenerateRailwayConfig() string {
	return `{
  "build": {
    "builder": "NIXPACKS"
  },
  "deploy": {
    "numReplicas": 2,
    "restartPolicyType": "ON_FAILURE",
    "restartPolicyMaxRetries": 10
  }
}`
}

func (g *DeploymentGenerator) GenerateMakefile() string {
	return `.PHONY: all build test lint fmt docker deploy

all: test lint build

build:
	go build -ldflags="-s -w" -o bin/siby ./cmd

test:
	go test -v -race -cover ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

docker:
	docker build -t siby-projet .

docker-run:
	docker run -p 8080:8080 siby-projet

deploy:
	@echo "Deploying to production..."
	./scripts/deploy.sh

security-scan:
	golangci-lint run --security-exp ./...

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build binary"
	@echo "  test           - Run tests"
	@echo "  lint           - Run linter"
	@echo "  docker         - Build Docker image"
	@echo "  deploy         - Deploy to production"
	@echo "  security-scan  - Run security audit"
`
}

func (g *DeploymentGenerator) GenerateK8sManifests(projectName string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
data:
  APP_ENV: "production"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  labels:
    app: %s
spec:
  replicas: 3
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: %s-config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: %s-service
spec:
  selector:
    app: %s
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: %s-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: %s
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
`, projectName, projectName, projectName, projectName, projectName, projectName, projectName, projectName, projectName, projectName, projectName, projectName)
}

func (g *DeploymentGenerator) SaveAll(projectPath, language, projectName string) error {
	templates := map[string]string{
		"Dockerfile":        g.GenerateDockerfile(language, ""),
		".github/workflows/ci.yml": g.GenerateGitHubActions(language, projectName),
		"Makefile":          g.GenerateMakefile(),
		"k8s/deployment.yaml": g.GenerateK8sManifests(projectName),
		"vercel.json":       g.GenerateVercelConfig(),
		"railway.json":      g.GenerateRailwayConfig(),
	}

	for path, content := range templates {
		fullPath := filepath.Join(projectPath, path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create dir %s: %w", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	return nil
}
