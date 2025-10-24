# Implementation Plan for WEB-FETCH-01: Web Content Fetcher

## Overview
The goal of this task is to implement a web content fetcher that can:
1. Fetch web pages using an HTTP client with retries and timeout
2. Parse HTML content to extract clean text, metadata, and links
3. Implement URL validation and security features (SSRF prevention)
4. Add rate limiting per domain
5. Store fetched content in a PostgreSQL database
6. Create API endpoints for fetching web pages and integrating with the chat system

## Implementation Plan

### Phase 1: Setup Project Structure (1 hour)
- Create necessary directories and files:
  - `internal/webfetch/`
    - config.go
    - interface.go
    - fetcher.go
    - parser.go
    - metadata.go
    - cleaner.go
    - validator.go
    - ratelimiter.go
    - cache.go
    - service_test.go

### Phase 2: HTTP Fetcher & Validation (3 hours)
- Implement URLValidator with SSRF prevention
- Create HTTP client with retry logic and timeout
- Implement rate limiting per domain
- Write unit tests for validator, fetcher, and rate limiter

### Phase 3: HTML Parser (3 hours)
- Integrate goquery for HTML parsing
- Extract clean text from HTML
- Extract metadata (title, description, Open Graph tags)
- Clean HTML by removing scripts, styles, and other unwanted elements
- Write unit tests for the parser

### Phase 4: Database Schema & Caching (2 hours)
- Create database schema for storing fetched web pages
- Implement caching mechanism with TTL
- Write migration script for creating tables

### Phase 5: API Endpoints (3 hours)
- Create API endpoints for:
  - Fetching a single web page
  - Batch fetching multiple web pages
  - Getting cached content
- Integrate with chat system to allow fetching and summarizing web content in chat

### Phase 6: Testing & Documentation (2 hours)
- Write integration tests for the entire fetcher service
- Update API documentation
- Add comments and documentation to code

## Dependencies
- Go libraries:
  - `github.com/PuerkitoBio/goquery` for HTML parsing
  - `net/http`, `net/url` for HTTP client
  - `golang.org/x/time/rate` for rate limiting

## Success Metrics
- Fetch speed: <2s for most websites
- Parse accuracy: >90% for clean content
- SSRF prevention: 100%
- Rate limiting: No API blocks
- Test coverage: >85%

## Timeline
Total estimated time: 6-8 hours

| Phase | Duration |
|-------|----------|
| Setup Project Structure | 1 hour |
| HTTP Fetcher & Validation | 3 hours |
| HTML Parser | 3 hours |
| Database Schema & Caching | 2 hours |
| API Endpoints | 3 hours |
| Testing & Documentation | 2 hours |

## Next Steps
1. Create the project structure
2. Start implementing the URLValidator and HTTP client