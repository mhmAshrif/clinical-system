# System Architecture

## Overview

The AI-Powered Clinic Notes & Billing System is built with a modern three-tier architecture:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   React Frontend │◄──►│   Go Backend    │◄──►│ PostgreSQL DB   │
│   (Port 5173)    │    │   (Port 8080)   │    │   (Supabase)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                       │                       │
        ▼                       ▼                       ▼
   Voice/Text Input        AI Parsing Logic       Structured Storage
   Results Display         Billing Calculation    Relationships
   Print Reports           REST API Endpoints    Normalization
```

## Component Architecture

### Frontend Layer (React + Vite)

#### Components
- **App.jsx**: Main application component
- **Form Handling**: Patient ID and note input
- **Voice Integration**: Web Speech API wrapper
- **Results Display**: Categorized parsed items
- **Print Functionality**: Browser-based report generation

#### State Management
- React hooks for local state
- Direct API calls (no Redux needed for demo)

#### Styling
- CSS modules with responsive design
- Clean, medical-themed UI
- Accessible form controls

### Backend Layer (Go + Gin)

#### API Structure
```
GET  /                    # Health check
GET  /test-db            # Database connectivity test
POST /parse-note         # Main parsing endpoint
GET  /billing/:id        # Patient billing history
```

#### Core Logic
- **Database Layer**: GORM ORM with PostgreSQL
- **Parsing Engine**: Heuristic-based AI classification
- **Billing Calculator**: Automatic total computation
- **CORS Middleware**: Cross-origin request handling

#### Models
- `Patient`: Basic patient information
- `MedicalRecord`: Raw notes and billing totals
- `ParsedItem`: Categorized items with pricing
- `PriceList`: Reference pricing data

### Database Layer (PostgreSQL)

#### Schema Design

```sql
patients (id, name, age, gender, created_at)
    ↓
medical_records (id, patient_id→, raw_note, total_bill, created_at)
    ↓
parsed_items (id, record_id→, category, item_name, dosage, price)
    ↑
price_list (item_name, category, price)
```

#### Relationships
- **One-to-Many**: Patient → Medical Records
- **One-to-Many**: Medical Record → Parsed Items
- **Many-to-One**: Parsed Items → Price List (lookup)

#### Data Flow
1. Raw note stored in `medical_records`
2. AI parsing creates `parsed_items` entries
3. Total calculated and updated in `medical_records`
4. Price references pulled from `price_list`

## AI Integration Architecture

### Current Implementation

#### Heuristic Parser
```
Input Text
    ↓
Text Normalization (lowercase, trim)
    ↓
Price List Matching
    ↓
Category Assignment (Drug/Lab Test/Observation)
    ↓
Dosage Extraction (regex patterns)
    ↓
Structured Output
```

#### Algorithm Details
- **Matching**: String containment against price list
- **Categorization**: Predefined categories with pricing
- **Fallback**: Unmatched text → "Clinical Notes"
- **Performance**: O(n*m) where n=input length, m=price list size

### Upgrade Path

#### OpenAI Integration
```
Input Text → OpenAI API → JSON Response → Validation → Database
```

#### Custom ML Model
```
Input Text → Tokenization → Model Inference → Post-processing → Database
```

## Security Architecture

### Data Protection
- **Input Validation**: Server-side validation of all inputs
- **SQL Injection Prevention**: GORM parameterized queries
- **CORS Configuration**: Restricted to frontend origin
- **Environment Variables**: Sensitive data in .env files

### Access Control
- **No Authentication**: Demo system (add OAuth for production)
- **API Rate Limiting**: Not implemented (add for production)
- **Audit Logging**: Basic logging (enhance for production)

## Performance Architecture

### Optimization Strategies
- **Database Indexing**: Primary keys and foreign keys indexed
- **Connection Pooling**: GORM default connection management
- **Caching**: None (add Redis for production)
- **Async Processing**: Synchronous for demo (add queues for production)

### Scalability Considerations
- **Horizontal Scaling**: Stateless backend design
- **Database Sharding**: Single database for demo
- **CDN**: Static assets served locally
- **Load Balancing**: Not implemented (add for production)

## Deployment Architecture

### Development Environment
```
Local Machine
├── Frontend (Vite dev server)
├── Backend (Go run)
└── Database (Supabase cloud)
```

### Production Environment
```
Cloud Infrastructure
├── Frontend (Static hosting/CDN)
├── Backend (Containerized Go app)
├── Database (Managed PostgreSQL)
└── Load Balancer (API Gateway)
```

### Containerization
```dockerfile
# Backend Container
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o main .
EXPOSE 8080
CMD ["./main"]

# Frontend Container
FROM node:18-alpine
WORKDIR /app
COPY . .
RUN npm install && npm run build
EXPOSE 80
CMD ["npm", "start"]
```

## Monitoring & Observability

### Logging
- **Application Logs**: Gin framework logging
- **Database Logs**: GORM debug logging
- **Error Tracking**: Basic error responses
- **Performance Metrics**: Response time logging

### Health Checks
- **Application Health**: `/` endpoint
- **Database Health**: `/test-db` endpoint
- **Dependency Checks**: Database connectivity

## Error Handling Architecture

### Error Types
- **Validation Errors**: 400 Bad Request
- **Database Errors**: 500 Internal Server Error
- **Not Found Errors**: 404 Not Found
- **CORS Errors**: 403 Forbidden

### Error Responses
```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": "Additional context"
}
```

## Testing Architecture

### Unit Tests
- **Backend**: Go testing framework
- **Frontend**: Jest + React Testing Library
- **Database**: Test database with migrations

### Integration Tests
- **API Testing**: Postman/Insomnia collections
- **E2E Testing**: Cypress for full workflows
- **Performance Testing**: Load testing scripts

## Future Enhancements

### Architecture Improvements
- **Microservices**: Split parsing into separate service
- **Event-Driven**: Message queues for async processing
- **GraphQL**: Flexible API queries
- **Real-time**: WebSocket updates

### Scalability Features
- **Caching Layer**: Redis for frequently accessed data
- **CDN**: Global asset distribution
- **Auto-scaling**: Container orchestration
- **Database Replication**: Read/write splitting

### Security Enhancements
- **Authentication**: JWT/OAuth integration
- **Authorization**: Role-based access control
- **Encryption**: Data at rest and in transit
- **Audit Trail**: Comprehensive logging

This architecture provides a solid foundation for the clinic system while being extensible for future requirements and production deployment.