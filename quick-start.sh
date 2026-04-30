#!/bin/bash

echo "=========================================="
echo "  User Management System - Quick Start   "
echo "=========================================="
echo ""

# Check Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

echo "✅ Docker and Docker Compose are installed"
echo ""

# Copy .env file if not exists
if [ ! -f .env ]; then
    echo "📝 Creating .env file from template..."
    cp .env.example .env
    echo "✅ .env file created"
else
    echo "✅ .env file already exists"
fi

echo ""
echo "🚀 Starting services..."
echo ""

# Start services
docker-compose up -d

echo ""
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check services
echo ""
echo "📊 Service Status:"
echo ""

# Check PostgreSQL
if docker-compose ps postgres | grep -q "Up"; then
    echo "✅ PostgreSQL is running"
else
    echo "❌ PostgreSQL failed to start"
fi

# Check Redis
if docker-compose ps redis | grep -q "Up"; then
    echo "✅ Redis is running"
else
    echo "❌ Redis failed to start"
fi

# Check Backend
if docker-compose ps backend | grep -q "Up"; then
    echo "✅ Backend is running"
else
    echo "❌ Backend failed to start"
fi

# Check Frontend
if docker-compose ps frontend | grep -q "Up"; then
    echo "✅ Frontend is running"
else
    echo "❌ Frontend failed to start"
fi

echo ""
echo "=========================================="
echo "  🎉 Services Started Successfully!      "
echo "=========================================="
echo ""
echo "📍 Access URLs:"
echo "   Frontend:     http://localhost:3000"
echo "   Backend API:  http://localhost:8080"
echo "   Health Check: http://localhost:8080/health"
echo ""
echo "📝 Default Credentials:"
echo "   PostgreSQL:"
echo "     Host: localhost:5432"
echo "     Database: user_system"
echo "     Username: admin"
echo "     Password: admin123"
echo ""
echo "   Redis:"
echo "     Host: localhost:6379"
echo "     Password: redis123"
echo ""
echo "📖 Documentation:"
echo "   API docs:     docs/API.md"
echo "   Deploy docs:  docs/DEPLOY.md"
echo "   SSO docs:     docs/SSO.md"
echo ""
echo "🔧 Useful Commands:"
echo "   View logs:    docker-compose logs -f"
echo "   Stop services: docker-compose down"
echo "   Restart:      docker-compose restart"
echo ""
echo "=========================================="
