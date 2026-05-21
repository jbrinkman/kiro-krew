# Kiro Krew: GitHub Issue-Driven AI Orchestration

A GitHub issue-driven orchestration system that transforms issues into working code through AI agent collaboration.

## What is Kiro Krew?

Kiro Krew is an AI orchestration platform that reads GitHub issues, creates technical specifications, and coordinates specialized AI agents to implement solutions. Issues become requirements, agents create designs, builders implement code, and the system generates pull requests automatically.

```
GitHub Issue ──→ Architect ──→ Builders ──→ Pull Request
     │              │            │             │
Requirements    Design      Implementation   Review
```

## Quick Start

### Installation

**Via Homebrew:**
```bash
brew install kiro-krew
```

**Via Go:**
```bash
go install github.com/kiro-dev/kiro-krew@latest
```

### Initialize Project

```bash
cd your-project
kiro-krew init
```

This creates `.kiro-krew/config.yaml` and agent configurations.

### Configure

Edit `.kiro-krew/config.yaml`:

```yaml
github:
  token: ghp_your_token_here
  owner: your-username
  repo: your-repo

agents:
  architect: claude-sonnet-4
  builder: claude-sonnet-4
  validator: claude-sonnet-4
  documenter: claude-haiku-3

orchestration:
  max_parallel_builders: 4
  auto_create_pr: true
```

### Run

```bash
# Process a specific issue
kiro-krew run --issue 42

# Process all open issues with 'krew' label
kiro-krew run --label krew

# Interactive mode
kiro-krew repl
```

## How It Works

Kiro Krew follows a structured pipeline that transforms GitHub issues into working code:

### 1. Issue Analysis
The **krew-lead** reads GitHub issues and determines if they're actionable:
- Extracts requirements and acceptance criteria
- Identifies dependencies and scope
- Routes to appropriate workflow

### 2. Architecture Phase
The **architect** creates technical specifications:
- Analyzes existing codebase
- Designs implementation approach
- Creates detailed task breakdown
- Defines validation criteria

### 3. Implementation Phase
**Builders** execute the implementation:
- Work in parallel on independent tasks
- Follow architectural specifications
- Create/modify code files
- Run tests and validation

### 4. Validation Phase
The **validator** ensures quality:
- Verifies implementation matches specs
- Runs comprehensive tests
- Checks integration points
- Validates acceptance criteria

### 5. Documentation Phase
The **documenter** creates documentation:
- Updates README and docs
- Documents API changes
- Creates usage examples
- Updates changelog

### 6. Pull Request
System creates PR with:
- Implementation code
- Test coverage
- Documentation updates
- Links to original issue

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    GitHub Integration                        │
│  Issues ──→ Webhooks ──→ Kiro Krew ──→ Pull Requests       │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────┴───────────────────────────────────┐
│                   Kiro Krew Core                            │
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │ Krew Lead   │───▶│ Architect   │───▶│ Builders    │     │
│  │ (Router)    │    │ (Designer)  │    │ (Workers)   │     │
│  └─────────────┘    └─────────────┘    └──────┬──────┘     │
│                                               │            │
│  ┌─────────────┐    ┌─────────────┐          │            │
│  │ Documenter  │◄───│ Validator   │◄─────────┘            │
│  │ (Writer)    │    │ (Tester)    │                       │
│  └─────────────┘    └─────────────┘                       │
└─────────────────────────────────────────────────────────────┘
```

## Configuration Reference

### .kiro-krew/config.yaml

```yaml
# GitHub Integration
github:
  token: ${GITHUB_TOKEN}           # GitHub personal access token
  owner: your-org                  # Repository owner
  repo: your-repo                  # Repository name
  base_branch: main               # Default branch for PRs

# Agent Configuration
agents:
  krew_lead: claude-sonnet-4      # Issue routing and coordination
  architect: claude-sonnet-4      # Technical design and planning
  builder: claude-sonnet-4        # Code implementation
  validator: claude-sonnet-4      # Testing and validation
  documenter: claude-haiku-3      # Documentation generation

# Orchestration Settings
orchestration:
  max_parallel_builders: 4        # Maximum concurrent builders
  auto_create_pr: true           # Automatically create pull requests
  require_tests: true            # Require test coverage
  require_docs: true             # Require documentation updates

# Issue Processing
issues:
  labels:
    - krew                       # Process issues with this label
    - enhancement               # Also process enhancements
  ignore_labels:
    - wontfix                   # Skip issues with these labels
    - duplicate
  
# Validation Rules
validation:
  min_test_coverage: 80          # Minimum test coverage percentage
  required_checks:
    - lint                      # Code linting
    - type_check               # Type checking
    - security_scan            # Security vulnerability scan

# Notification Settings
notifications:
  slack_webhook: ${SLACK_WEBHOOK} # Optional Slack notifications
  email: team@company.com        # Optional email notifications
```

## Agent Roles

### Krew Lead
**Purpose:** Issue routing and workflow coordination
- Analyzes GitHub issues for actionability
- Routes issues to appropriate workflows
- Coordinates agent handoffs
- Manages overall process state

**Capabilities:**
- Read GitHub issues and comments
- Create and manage specifications
- Spawn and coordinate other agents
- Update issue status and labels

### Architect
**Purpose:** Technical design and planning
- Analyzes existing codebase architecture
- Creates detailed implementation specifications
- Defines task breakdown and dependencies
- Establishes validation criteria

**Capabilities:**
- Read and analyze code repositories
- Create technical specifications
- Design system architecture
- Plan implementation approach

### Builder
**Purpose:** Code implementation and development
- Implements features according to specifications
- Creates and modifies source code files
- Writes tests and validation code
- Handles build and deployment tasks

**Capabilities:**
- Read and write code files
- Execute build and test commands
- Install dependencies
- Run development tools

### Validator
**Purpose:** Quality assurance and testing
- Verifies implementation matches specifications
- Runs comprehensive test suites
- Validates integration points
- Ensures acceptance criteria are met

**Capabilities:**
- Execute test suites
- Analyze code coverage
- Run security scans
- Validate system behavior

### Documenter
**Purpose:** Documentation and communication
- Updates project documentation
- Creates API documentation
- Writes usage examples
- Maintains changelog

**Capabilities:**
- Read implementation code
- Generate documentation
- Update README files
- Create usage examples

## Planning Skill Usage

Use the `@plan-with-krew` skill to create implementation plans from GitHub issues:

```bash
# In Kiro CLI
@plan-with-krew https://github.com/owner/repo/issues/42

# Or reference issue number if in project context
@plan-with-krew #42

# Create plan from issue description
@plan-with-krew Add user authentication with JWT tokens
```

The planning skill:
1. Analyzes the issue or description
2. Creates technical specifications
3. Defines task breakdown
4. Establishes validation criteria
5. Saves plan for execution

## CLI Commands Reference

### Main Commands

```bash
# Initialize project
kiro-krew init

# Process specific issue
kiro-krew run --issue 42

# Process issues by label
kiro-krew run --label enhancement

# Interactive REPL mode
kiro-krew repl

# Show configuration
kiro-krew config show

# Validate configuration
kiro-krew config validate
```

### REPL Commands

```bash
# List available issues
> issues list

# Process an issue
> process #42

# Show current status
> status

# List active agents
> agents list

# Show agent status
> agent status builder-1

# Cancel running process
> cancel

# Show help
> help
```

## Separation of Concerns

### Issues = Requirements
GitHub issues serve as the requirements specification:
- **What** needs to be built
- **Why** it's needed (business value)
- **Acceptance criteria** for completion
- **User stories** and use cases

### Specs = Design
Technical specifications define the implementation approach:
- **How** the solution will be built
- **Architecture** and design patterns
- **Task breakdown** and dependencies
- **Validation approach** and test strategy

### Code = Implementation
Source code implements the designed solution:
- **Working software** that meets requirements
- **Tests** that validate functionality
- **Documentation** that explains usage
- **Integration** with existing systems

## Troubleshooting

### Common Issues

**"GitHub token invalid"**
- Verify token has required permissions: `repo`, `issues`, `pull_requests`
- Check token is not expired
- Ensure token is correctly set in config or environment

**"Issue not found"**
- Verify issue number exists in specified repository
- Check repository owner/name in configuration
- Ensure issue is not private (if using public token)

**"Agent spawn failed"**
- Check agent model availability and API keys
- Verify agent configuration in `.kiro-krew/config.yaml`
- Review agent-specific error logs

**"Build validation failed"**
- Check test coverage meets minimum requirements
- Verify all required checks pass (lint, type check, security)
- Review validation logs for specific failures

**"PR creation failed"**
- Verify GitHub token has `pull_requests` permission
- Check base branch exists and is accessible
- Ensure no conflicting PR exists for same issue

### Debug Mode

Enable debug logging:

```bash
export KIRO_KREW_DEBUG=true
kiro-krew run --issue 42
```

### Log Files

Logs are written to:
- `~/.kiro-krew/logs/krew-lead.log`
- `~/.kiro-krew/logs/architect.log`
- `~/.kiro-krew/logs/builder-{id}.log`
- `~/.kiro-krew/logs/validator.log`
- `~/.kiro-krew/logs/documenter.log`

### Support

- Documentation: https://docs.kiro-krew.dev
- Issues: https://github.com/kiro-dev/kiro-krew/issues
- Discussions: https://github.com/kiro-dev/kiro-krew/discussions