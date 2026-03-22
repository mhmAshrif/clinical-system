# AI-Powered Clinic Management System

## 🚀 Quick Start
```bash
# Backend
cd backend && go run main.go database.go

# Frontend (new terminal)
cd frontend && npm install && npm run dev
```

## 🏗️ Tech Stack
- **Backend**: Go, Gin, GORM, PostgreSQL
- **Frontend**: React, Vite, Modern CSS
- **AI**: Gemini API for text classification
- **Database**: Supabase PostgreSQL

## ✨ Key Features
- **AI Text Parsing**: Converts doctor notes into structured medical data
- **Full CRUD**: Create, Read, Update, Delete patients & records
- **Billing System**: Automatic price calculation & history
- **Voice Input**: Speech-to-text for clinic notes
- **Print Reports**: Generate formatted medical reports

## 🔧 Setup Requirements
- Go 1.21+
- Node.js 18+
- PostgreSQL database
- Gemini API key (in backend/.env)

## 📋 Environment Setup
Create `backend/.env`:
```
DB_URL=postgresql://user:pass@host:port/db
GEMINI_API_KEY=your_api_key_here
```

## 🎯 What This Demonstrates
- Full-stack development (Go + React)
- AI integration (Gemini API)
- Database design & ORM (GORM)
- REST API design
- Modern UI/UX with responsive design
- Real-time data processing
- CRUD operations with proper error handling

## 🏥 Usage
1. Create patient → Enter medical notes → Parse with AI → View billing history
2. Supports both heuristic and AI-powered parsing
3. Edit/delete patients and medical records
4. Print formatted reports

---
**Built for healthcare professionals with AI-powered efficiency**