#!/bin/bash

set -euo pipefail

# Validate planner tool restrictions - ensures write operations only occur in planning worktrees

check_current_branch() {
    local current_branch
    current_branch=$(git branch --show-current)
    
    # Check if we're on main branch
    if [[ "$current_branch" == "main" ]]; then
        echo "ERROR: Planner cannot perform write operations on main branch"
        return 1
    fi
    
    # Check if we're in a planning worktree
    if [[ "$current_branch" == planning-* ]]; then
        echo "✅ Planning worktree detected: $current_branch"
        return 0
    fi
    
    # Check if we're in an issue worktree (not allowed for planner writes)
    if [[ "$current_branch" == spec/issue-* ]]; then
        echo "ERROR: Planner cannot write in issue processing worktree: $current_branch"
        return 1
    fi
    
    echo "WARNING: Unknown branch pattern: $current_branch"
    return 1
}

check_worktree_location() {
    local pwd_path
    pwd_path=$(pwd)
    
    # Check if in planning worktree directory
    if [[ "$pwd_path" == */.worktrees/planning-* ]]; then
        echo "✅ In planning worktree directory"
        return 0
    fi
    
    # Check if in main directory (not allowed for writes)
    if [[ "$pwd_path" != */.worktrees/* ]]; then
        echo "ERROR: Planner write operations must occur in planning worktrees only"
        return 1
    fi
    
    echo "WARNING: Unknown worktree location: $pwd_path"
    return 1
}

check_planning_worktree_cleanup() {
    echo "Checking for leftover planning worktrees..."
    
    local planning_worktrees
    planning_worktrees=$(find .worktrees -name "planning-*" -type d 2>/dev/null || true)
    
    if [[ -n "$planning_worktrees" ]]; then
        echo "❌ Found leftover planning worktrees:"
        echo "$planning_worktrees"
        echo "ERROR: Planning worktrees must be cleaned up after use"
        return 1
    fi
    
    # Check for orphaned planning branches
    local planning_branches
    planning_branches=$(git branch --list "planning/*" 2>/dev/null || true)
    
    if [[ -n "$planning_branches" ]]; then
        echo "❌ Found leftover planning branches:"
        echo "$planning_branches"
        echo "ERROR: Planning branches must be cleaned up after use"
        return 1
    fi
    
    echo "✅ No leftover planning artifacts found"
    return 0
}

main() {
    echo "Validating planner tool restrictions..."
    
    local errors=0
    
    if ! check_current_branch; then
        ((errors++))
    fi
    
    if ! check_worktree_location; then
        ((errors++))
    fi
    
    if ! check_planning_worktree_cleanup; then
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        echo "✅ All planner tool restrictions validated"
        return 0
    else
        echo "❌ Found $errors violation(s)"
        return 1
    fi
}

main "$@"
