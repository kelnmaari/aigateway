# DEVOPS-01: CI/CD Pipeline

**Приоритет:** MEDIUM  
**Версия:** 1.4.0  
**Оценка:** 6-8 часов  

---

## Цель

Автоматизация тестирования, сборки и релизов через GitHub Actions.

---

## Workflows

### 1. Test Pipeline (`.github/workflows/test.yml`)

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: go test -v -race -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out
```

### 2. Build Pipeline (`.github/workflows/build.yml`)

```yaml
name: Build
on:
  push:
    branches: [main]
jobs:
  build:
    strategy:
      matrix:
        os: [linux, windows, darwin]
        arch: [amd64, arm64]
    steps:
      - run: go build -o bin/server-${{ matrix.os }}-${{ matrix.arch }}
      - uses: actions/upload-artifact@v4
```

### 3. Release Pipeline (`.github/workflows/release.yml`)

```yaml
name: Release
on:
  push:
    tags: ['v*']
jobs:
  release:
    steps:
      - run: goreleaser release --clean
      - uses: actions/create-release@v1
```

---

## Features

- ✅ Automated testing on PR
- ✅ Coverage reporting (Codecov)
- ✅ Cross-platform builds
- ✅ Automated releases (tags)
- ✅ Docker image builds
- ✅ Security scanning (gosec)
- ✅ Dependency updates (Dependabot)

---

**Tools:** GitHub Actions, GoReleaser, Codecov  
**Estimated Time:** 6-8 hours

