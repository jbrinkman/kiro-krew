---
name: github-cli-mock
description: Mock GitHub CLI skill that logs operations instead of executing them for safe eval testing.
---

# github-cli-mock

## Purpose
Drop-in replacement for GitHub CLI skill that logs operations instead of executing them during eval testing.

## Behavior
- Intercepts all `gh` commands
- Logs operations with [MOCK] prefix
- Returns realistic mock responses
- Maintains command compatibility

## Mock Operations
- `gh issue create` → Returns mock issue number and URL
- `gh pr create` → Returns mock PR number and URL  
- `gh issue list` → Returns empty list
- `gh pr list` → Returns empty list
- All other `gh` commands → Success with logged operation

## Implementation Notes
- Safe for containerized eval environments
- No real GitHub API calls made
- Realistic response format for testing