# Credit Service API

A production-ready credit service API built with Go, Fiber framework, and MongoDB for managing user credits and token usage tracking.

## 🚀 Features

- Token usage tracking and credit management
- MongoDB integration for data persistence
- Portkey integration for AI gateway
- Clean Architecture implementation
- RESTful API endpoints
- Request logging and error recovery middleware

## 📋 Prerequisites

Before running this service, ensure you have the following installed:

- **Go** 1.25.4 or higher ([Download](https://golang.org/dl/))
- **MongoDB** (local instance or MongoDB Atlas)
- **Git** (for cloning the repository)

## 🛠️ Installation

### 1. Clone the Repository

```bash
git clone <your-repository-url>
cd credit-service-go
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Environment Configuration

Create a `.env` file in the root directory by copying the example file:

```bash
cp .env.example .env
```

Edit the `.env` file with your configuration:

```env
# MongoDB Configuration
MONGO_URL=mongodb://localhost:27017/my_database
MONGO_DB_NAME=my_database

# Portkey Configuration (AI Gateway)
PORTKEY_URL=your_portkey_url_here
PORTKEY_API_KEY=your_portkey_api_key_here
PORTKEY_WORKSPACE_SLUG=your_portkey_workspace_slug_here

# API Security
X_API_KEY=your_x_api_key_here
```

#### Configuration Details:

- **MONGO_URL**: MongoDB connection string
  - Local: `mongodb://localhost:27017/my_database`
  - Atlas: `mongodb+srv://<username>:<password>@cluster.mongodb.net/<database>`
- **MONGO_DB_NAME**: Database name for the service
- **PORTKEY_URL**: Portkey API endpoint URL
- **PORTKEY_API_KEY**: Your Portkey API key
- **PORTKEY_WORKSPACE_SLUG**: Your Portkey workspace identifier
- **X_API_KEY**: API key for authenticating requests

## 🚀 Running the Service

### Development Mode

Run the service directly with Go:

```bash
go run cmd/api/main.go
```

The API will start on **http://localhost:3000**

### Production Build

Build and run the optimized binary:

```bash
# Build the binary
go build -o bin/credit-service cmd/api/main.go

# Run the binary
./bin/credit-service
```

On Windows:
```powershell
# Build the binary
go build -o bin/credit-service.exe cmd/api/main.go

# Run the binary
.\bin\credit-service.exe
```

## 📡 API Endpoints

### Root Endpoint
```http
GET /
```
Health check endpoint to verify the service is running.

### Token Usage Endpoint
```http
POST /api/v1/token_used
```

## 🏗️ Project Structure

```
credit-service-go/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── adapter/
│   │   ├── client/              # External service clients
│   │   ├── handler/
│   │   │   └── http/            # HTTP handlers
│   │   └── repository/
│   │       └── mongodb/         # MongoDB repositories
│   ├── config/                  # Configuration management
│   ├── core/                    # Business logic/domain
│   └── service/                 # Application services
├── pkg/                         # Public packages
├── .env.example                 # Environment variables template
├── .gitignore
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
└── README.md
```

## 🧪 Testing

Run all tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

## 🔧 Environment Variables Reference

| Variable | Description | Required | Example |
|----------|-------------|----------|---------|
| `MONGO_URL` | MongoDB connection string | Yes | `mongodb://localhost:27017/my_database` |
| `MONGO_DB_NAME` | Database name | Yes | `my_database` |
| `PORTKEY_URL` | Portkey API URL | Yes | `https://api.portkey.ai` |
| `PORTKEY_API_KEY` | Portkey authentication key | Yes | `pk_xxx` |
| `PORTKEY_WORKSPACE_SLUG` | Portkey workspace identifier | Yes | `my-workspace` |
| `X_API_KEY` | API authentication key | Yes | `your-secret-key` |

## 🐳 Docker Deployment (Optional)

If you want to containerize the application, create a `Dockerfile`:

```dockerfile
FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bin/credit-service cmd/api/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bin/credit-service .
COPY --from=builder /app/.env .

EXPOSE 3000
CMD ["./credit-service"]
```

Build and run with Docker:
```bash
docker build -t credit-service .
docker run -p 3000:3000 --env-file .env credit-service
```

## 🌐 Cloud Deployment

### Google Cloud Run

This service is designed to run on Google Cloud Run. Deploy using:

```bash
gcloud run deploy credit-service \
  --source . \
  --platform managed \
  --region asia-southeast1 \
  --allow-unauthenticated
```

Make sure to configure environment variables in Cloud Run settings.

## 🔒 Security Considerations

- Always use environment variables for sensitive data
- Never commit `.env` file to version control
- Use strong API keys
- Enable HTTPS in production
- Implement rate limiting for production deployments
- Validate and sanitize all input data

## 📝 Development

### Code Style

This project follows Go best practices and Clean Architecture principles:
- Clear separation of concerns
- Dependency injection
- Interface-based design
- Repository pattern for data access

### Adding New Features

1. Define domain entities in `internal/core`
2. Create service layer in `internal/service`
3. Implement repository in `internal/adapter/repository`
4. Add HTTP handlers in `internal/adapter/handler/http`
5. Register routes in `router.go`

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 📞 Support

For issues and questions:
- Create an issue in the repository
- Contact the development team

## 🔄 Version History

- **v1.0.0** - Initial release with token usage tracking

---

**Built with ❤️ using Go and Fiber**
