#!/bin/bash
set -e

echo "🚀 FastAPI Generation Demo - Week 9 Showcase"
echo "=============================================="

echo ""
echo "📋 Demo Overview:"
echo "   • Generate complete FastAPI application with authentication"
echo "   • Demonstrate 13+ file generation (exceeds 11 file target)"
echo "   • Show perfect router separation (auth.py vs users.py)"
echo "   • Create .qlcapsule package"
echo "   • Use real LLM integration (AWS Bedrock)"
echo ""

read -p "Press Enter to start generation..."

echo ""
echo "🎯 Generating: E-commerce API with user authentication"
echo "⏳ This will take about 60 seconds with real LLM..."

# Run the generation command
./bin/qlf generate 'Create a REST API for e-commerce with user authentication, product catalog, and order management' \
    --provider=bedrock \
    --model=anthropic.claude-3-sonnet-20240229-v1:0 \
    --package \
    --verbose

# Get the latest project directory
LATEST_PROJECT=$(find ./generated -maxdepth 1 -type d -name "project-*" | sort | tail -1)

if [ -z "$LATEST_PROJECT" ]; then
    echo "❌ No project generated"
    exit 1
fi

echo ""
echo "✅ Generation Complete!"
echo "📁 Project: $LATEST_PROJECT"

# Count files
TOTAL_FILES=$(find "$LATEST_PROJECT" -type f -name "*.py" | wc -l)
echo "📊 Python files generated: $TOTAL_FILES"

# Check router separation
echo ""
echo "🔍 Router Separation Check:"
if [ -f "$LATEST_PROJECT/routers/auth.py" ] && [ -f "$LATEST_PROJECT/routers/users.py" ]; then
    echo "✅ Separate router files created:"
    echo "   • auth.py (authentication endpoints)"
    echo "   • users.py (user management endpoints)"

    # Show content samples
    echo ""
    echo "📄 auth.py endpoints:"
    grep -E "@router\.(get|post|put|delete)" "$LATEST_PROJECT/routers/auth.py" | sed 's/^/   • /'

    echo ""
    echo "📄 users.py endpoints:"
    grep -E "@router\.(get|post|put|delete)" "$LATEST_PROJECT/routers/users.py" | sed 's/^/   • /'
else
    echo "⚠️  Router separation issue detected"
fi

# Check required files
echo ""
echo "📋 File Completeness Check:"
REQUIRED_FILES=(
    "main.py"
    "models.py"
    "schemas.py"
    "services.py"
    "repositories.py"
    "database.py"
    "config.py"
    "dependencies.py"
    "requirements.txt"
    "routers/auth.py"
    "routers/users.py"
)

MISSING_FILES=0
for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$LATEST_PROJECT/$file" ]; then
        echo "✅ $file"
    else
        echo "❌ $file (missing)"
        ((MISSING_FILES++))
    fi
done

if [ $MISSING_FILES -eq 0 ]; then
    echo "🎉 All $((${#REQUIRED_FILES[@]})) required files present!"
else
    echo "⚠️  $MISSING_FILES files missing"
fi

# Check package creation
echo ""
echo "📦 Package Check:"
if [ -f "$LATEST_PROJECT/packages"/*.qlcapsule ]; then
    PACKAGE_FILE=$(ls "$LATEST_PROJECT/packages"/*.qlcapsule)
    PACKAGE_SIZE=$(du -h "$PACKAGE_FILE" | cut -f1)
    echo "✅ .qlcapsule package created: $PACKAGE_SIZE"
else
    echo "❌ No .qlcapsule package found"
fi

# Show JWT authentication check
echo ""
echo "🔐 Authentication Features:"
if grep -q "jwt\|JWT" "$LATEST_PROJECT/routers/auth.py" 2>/dev/null; then
    echo "✅ JWT authentication implemented"
else
    echo "⚠️  JWT authentication not detected"
fi

if grep -q "bcrypt\|passlib" "$LATEST_PROJECT/requirements.txt" 2>/dev/null; then
    echo "✅ Password hashing with bcrypt"
else
    echo "⚠️  Password hashing not detected"
fi

# Show database features
echo ""
echo "🗄️  Database Features:"
if grep -q "UUID\|uuid" "$LATEST_PROJECT/models.py" 2>/dev/null; then
    echo "✅ UUID primary keys"
else
    echo "⚠️  UUID primary keys not detected"
fi

if grep -q "postgresql\|psycopg" "$LATEST_PROJECT/requirements.txt" 2>/dev/null; then
    echo "✅ PostgreSQL driver included"
else
    echo "⚠️  PostgreSQL driver not detected"
fi

echo ""
echo "🎯 Demo Results Summary:"
echo "   • Files Generated: $TOTAL_FILES (target: 11+)"
echo "   • Router Separation: $([ -f "$LATEST_PROJECT/routers/auth.py" ] && [ -f "$LATEST_PROJECT/routers/users.py" ] && echo "✅ Success" || echo "❌ Failed")"
echo "   • Package Created: $([ -f "$LATEST_PROJECT/packages"/*.qlcapsule ] && echo "✅ Success" || echo "❌ Failed")"
echo "   • Missing Files: $MISSING_FILES"

if [ $TOTAL_FILES -ge 11 ] && [ $MISSING_FILES -eq 0 ] && [ -f "$LATEST_PROJECT/routers/auth.py" ] && [ -f "$LATEST_PROJECT/routers/users.py" ]; then
    echo ""
    echo "🎉 DEMO SUCCESS! All Week 9 objectives achieved!"
    echo "   Ready for design partner presentation ✨"
else
    echo ""
    echo "⚠️  Some objectives not fully met - review results above"
fi

echo ""
echo "📂 Generated project available at: $LATEST_PROJECT"
echo "🔍 Explore with: ls -la $LATEST_PROJECT"