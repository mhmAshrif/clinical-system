# Setup Instructions

## Prerequisites

- **Go**: Version 1.21 or higher
- **Node.js**: Version 18 or higher
- **PostgreSQL**: Database instance (local or cloud like Supabase)
- **Git**: For version control

## Quick Setup (5 minutes)

### 1. Backend Setup

```bash
cd backend

# Install dependencies
go mod tidy

# Configure database
# Copy .env.example to .env and update DB_URL
cp .env.example .env

# Run database migrations (optional - GORM auto-migrates)
# Run the SQL in schema.sql if needed

# Start server
go run .
```

### 2. Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

### 3. Database Setup

Run the SQL commands in `backend/schema.sql` to create tables:

```sql
-- Copy and paste the contents of schema.sql into your PostgreSQL client
```

## Detailed Configuration

### Environment Variables

Create `backend/.env` with:

```env
DB_URL=postgresql://username:password@host:port/database
```

### Database Connection

The application supports:
- **Supabase**: Cloud PostgreSQL
- **Local PostgreSQL**: Standard installation
- **Docker PostgreSQL**: Containerized setup

### Port Configuration

- **Backend**: `http://localhost:8080`
- **Frontend**: `http://localhost:5174`
- **Database**: Default PostgreSQL port (5432)

## Testing the Setup

### 1. Health Check

```bash
# Backend health
curl http://localhost:8080/

# Database connection
curl http://localhost:8080/test-db
```

### 2. Full Workflow Test

1. Open `http://localhost:5174` in browser
2. Enter Patient ID: `1`
3. Enter Note: `"Patient needs Paracetamol 500mg and Full Blood Count"`
4. Click "Parse & Save"
5. Verify results display
6. Test "Print Report" functionality
7. Check "View Billing History"

### 3. API Testing

```bash
# Parse note
curl -X POST -H "Content-Type: application/json" \
  http://localhost:8080/parse-note \
  -d '{"patient_id":1,"raw_note":"Paracetamol 500mg, Full Blood Count"}'

# Get billing
curl http://localhost:8080/billing/1
```

## Troubleshooting

### Common Issues

**Backend won't start:**
- Check Go version: `go version`
- Verify `.env` file exists with correct DB_URL
- Ensure database is accessible

**Frontend build fails:**
- Check Node.js version: `node --version`
- Clear node_modules: `rm -rf node_modules && npm install`

**Database connection fails:**
- Verify connection string format
- Check firewall settings
- Ensure database server is running

**CORS errors:**
- Backend automatically enables CORS
- Check if backend is running on port 8080

### Debug Commands

```bash
# Check backend logs
cd backend && go run . 2>&1

# Check frontend logs
cd frontend && npm run dev

# Test database connectivity
psql "your-connection-string" -c "SELECT 1;"

# Check ports
netstat -ano | findstr :8080
netstat -ano | findstr :5174
```

## Production Deployment

### Docker Deployment

```dockerfile
# Backend Dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o main .
EXPOSE 8080
CMD ["./main"]

# Frontend Dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build
EXPOSE 80
CMD ["npm", "start"]
```

### Environment Variables for Production

```env
DB_URL=postgresql://prod-user:prod-pass@prod-host:5432/prod-db
GIN_MODE=release
NODE_ENV=production
```

## Performance Tuning

### Database
- Ensure proper indexing on frequently queried columns
- Configure connection pool settings
- Monitor query performance

### Backend
- Set GIN_MODE=release for production
- Configure appropriate timeouts
- Enable gzip compression

### Frontend
- Build optimized production bundle
- Enable service worker for caching
- Configure proper asset loading

## Security Checklist

- [ ] Database credentials encrypted
- [ ] CORS properly configured
- [ ] Input validation implemented
- [ ] HTTPS enabled in production
- [ ] Database backups configured
- [ ] Access logs enabled

## Support

For issues not covered here:
1. Check the logs for error messages
2. Verify all prerequisites are installed
3. Test individual components in isolation
4. Review the architecture documentation

The setup is designed to be straightforward - if you encounter persistent issues, please check the prerequisites and environment configuration first.