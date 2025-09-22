#!/bin/bash
set -e

echo "🔄 Multi-Framework Demo - Platform Flexibility"
echo "=============================================="

echo ""
echo "📋 Demo Overview:"
echo "   • Generate Go + Gin API"
echo "   • Generate Python + FastAPI with different features"
echo "   • Show database schema variations"
echo "   • Demonstrate platform adaptability"
echo ""

read -p "Press Enter to start multi-framework demo..."

echo ""
echo "🎯 Demo 1: Go + Gin API Generation"
echo "⏳ Generating Go microservice..."

./bin/qlf generate 'Create a Go microservice for user management with Gin framework' \
    --provider=bedrock \
    --model=anthropic.claude-3-sonnet-20240229-v1:0 \
    --verbose

GO_PROJECT=$(find ./generated -maxdepth 1 -type d -name "project-*" | sort | tail -1)
GO_FILES=$(find "$GO_PROJECT" -type f -name "*.go" | wc -l)
echo "✅ Go project: $GO_FILES Go files generated"

echo ""
echo "🎯 Demo 2: Python + FastAPI with Different Features"
echo "⏳ Generating FastAPI with blog features..."

./bin/qlf generate 'Create a FastAPI blog platform with posts, comments, and user profiles' \
    --provider=bedrock \
    --model=anthropic.claude-3-sonnet-20240229-v1:0 \
    --verbose

BLOG_PROJECT=$(find ./generated -maxdepth 1 -type d -name "project-*" | sort | tail -1)
BLOG_FILES=$(find "$BLOG_PROJECT" -type f -name "*.py" | wc -l)
echo "✅ Blog project: $BLOG_FILES Python files generated"

echo ""
echo "🔍 Framework Comparison:"
echo "┌─────────────────┬──────────────┬──────────────┐"
echo "│ Framework       │ Files        │ Features     │"
echo "├─────────────────┼──────────────┼──────────────┤"
printf "│ %-15s │ %-12s │ %-12s │\n" "Go + Gin" "$GO_FILES files" "Microservice"
printf "│ %-15s │ %-12s │ %-12s │\n" "Python + FastAPI" "$BLOG_FILES files" "Blog Platform"
echo "└─────────────────┴──────────────┴──────────────┘"

echo ""
echo "📂 Projects generated:"
echo "   • Go API: $GO_PROJECT"
echo "   • Blog API: $BLOG_PROJECT"

echo ""
echo "🎉 Multi-framework demo complete!"
echo "   Platform successfully adapts to different technology stacks"