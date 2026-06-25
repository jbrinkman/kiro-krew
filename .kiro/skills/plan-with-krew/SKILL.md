---
name: plan-with-krew
description: Collaboratively refine user requests into well-structured GitHub issues with mandatory workflow gates. Use when planning features, creating issues, or preparing work for kiro-krew automated processing.
---

# plan-with-krew

## Purpose
Collaboratively refine user requests into well-structured GitHub issues with mandatory workflow gates.

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
6. **Draft Review Gate**: MANDATORY - Present the complete issue draft to the user and require explicit approval before creation
7. **Label Confirmation Gate**: MANDATORY - After issue creation, explicitly ask for confirmation before applying the kiro-krew label

## Usage
- `/plan-with-krew [description]` - Plan and create issue with mandatory workflow gates

## Implementation Notes
- Focus on problem space, not solution details
- Ensure acceptance criteria are testable
- Include relevant code references and constraints
- Both workflow gates must be completed - no bypassing allowed
