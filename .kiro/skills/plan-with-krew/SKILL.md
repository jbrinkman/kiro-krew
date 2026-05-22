---
name: plan-with-krew
description: Collaboratively refine user requests into well-structured GitHub issues for development work. Use when planning features, creating issues, or preparing work for kiro-krew automated processing.
---

# plan-with-krew

## Purpose
Collaboratively refine user requests into well-structured GitHub issues for development work.

## Behavior
1. **Analyze Request**: Take user's message describing a feature/problem
2. **Explore Codebase**: Investigate relevant code to understand context and constraints
3. **Collaborate**: Work with user to refine and clarify requirements
4. **Generate Issue**: Create GitHub issue body with:
   - Problem statement
   - User Story
   - Acceptance Criteria
   - Constraints
   - Context/references
5. **Create Issue**: Use `gh issue create` to submit the issue
6. **Label Option**: Ask "Should I label this for kiro-krew to start working on it?"
7. **Apply Label**: If yes, apply `kiro-krew` label to the issue

## Usage
- `/plan-with-krew [description]` - Plan and create issue
- `/plan-with-krew Build auth and start immediately` - Plan, create, and auto-label for immediate work

## Implementation Notes
- Focus on problem space, not solution details
- Ensure acceptance criteria are testable
- Include relevant code references and constraints
- Support immediate labeling for urgent work
