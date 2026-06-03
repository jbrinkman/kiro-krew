#!/bin/bash

# Integration Test Demonstration Script
# This script demonstrates the complete hotkey toggle workflow

set -e

echo "🔥 Hotkey Toggle Integration Test Demonstration"
echo "=============================================="
echo

# Test 1: Basic functionality
echo "📋 Test 1: Basic Hotkey Functionality"
echo "--------------------------------------"
cd internal/hotkey
go test -run TestHotkeyIntegrationEndToEnd -v
echo "✅ Basic hotkey functionality tests passed"
echo

# Test 2: Session state preservation  
echo "📋 Test 2: Session State Preservation"
echo "-------------------------------------"
go test -run TestSessionStatePreservation -v
echo "✅ Session state preservation tests passed"
echo

# Test 3: Process management and cleanup
echo "📋 Test 3: Process Management & Cleanup"
echo "---------------------------------------"
go test -run TestProcessManagementAndCleanup -v
echo "✅ Process management and cleanup tests passed"
echo

# Test 4: Error handling and validation
echo "📋 Test 4: Error Handling & Validation"
echo "--------------------------------------"
go test -run TestHotkeyValidation -v
echo "✅ Error handling and validation tests passed"
echo

# Test 5: Complete workflow
echo "📋 Test 5: Complete End-to-End Workflow"
echo "---------------------------------------"
go test -run TestCompleteWorkflow -v
echo "✅ Complete workflow tests passed"
echo

cd ../..

# Test 6: Cross-component integration
echo "📋 Test 6: Cross-Component Integration"
echo "-------------------------------------"
echo "Running hotkey tests..."
cd internal/hotkey && go test -v >/dev/null 2>&1
echo "✅ Hotkey component tests passed"

echo "Running session tests..."  
cd ../session && go test -v >/dev/null 2>&1
echo "✅ Session component tests passed"

cd ../..

# Test 7: Build verification
echo "📋 Test 7: Build Verification"
echo "-----------------------------"
echo "Building project to verify integration..."
go build ./cmd/kiro-krew >/dev/null 2>&1
echo "✅ Project builds successfully with hotkey integration"
echo

echo "🎉 All Integration Tests Passed!"
echo "================================"
echo
echo "Summary:"
echo "- ✅ Hotkey detection and processing"
echo "- ✅ Mode switching (Console ↔ Planning)"
echo "- ✅ Session state preservation"
echo "- ✅ Process management and cleanup"
echo "- ✅ Error handling and validation"
echo "- ✅ Complete end-to-end workflow"
echo "- ✅ Cross-component integration"
echo "- ✅ Build verification"
echo
echo "The hotkey toggle system is fully functional and ready for use!"
echo "Use Ctrl+Alt+P to toggle between console and planning modes."