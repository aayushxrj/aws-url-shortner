# AWS URL Shortener

A high-performance URL shortening service built with Go, gRPC, and AWS DynamoDB. This microservice provides both gRPC and HTTP interfaces for creating, managing, and tracking shortened URLs with built-in analytics.

## 🌟 Features

- **URL Shortening**: Convert long URLs into short, shareable links
- **Click Tracking**: Automatic click counting and analytics
- **Expiration Management**: Set custom expiration times for URLs
- **Dual Interface**: 
  - gRPC API for efficient service-to-service communication
  - HTTP redirect server for browser-based link redirection
- **Analytics Dashboard**: Track URL statistics including clicks, creation date, and expiration
- **CRUD Operations**: Full management of shortened URLs (Create, Read, Update, Delete)
- **Pagination Support**: Efficiently list all URLs with pagination
- **Health Checks**: Built-in health check endpoints
- **CORS Support**: Configured for cross-origin requests
- **Containerized**: Docker support with multi-stage builds
- **gRPC-Web Ready**: Envoy proxy configuration for browser compatibility

## 🏗️ Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ├──────────────┐
       │              │
   ┌───▼────┐    ┌───▼────────┐
   │  HTTP  │    │ gRPC/gRPC-Web│
   │ :8080  │    │   :50051    │
   └───┬────┘    └───┬─────────┘
       │              │
       │         ┌────▼────┐
       │         │ Envoy   │
       │         │ :8081   │
       │         └────┬────┘
       │              │
       └──────┬───────┘
              │
       ┌──────▼──────┐
       │  gRPC Server │
       │  (Handlers)  │
       └──────┬───────┘
              │
       ┌──────▼──────┐
       │  DynamoDB   │
       │   (AWS)     │
       └─────────────┘
```

## 📋 Prerequisites

- **Go** 1.25 or higher
- **Protocol Buffers** compiler (protoc)
- **Docker** (optional, for containerization)
- **AWS Account** with DynamoDB access
- **AWS CLI** configured (or environment variables set)

## 🚀 Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/aayushxrj/aws-url-shortner.git
cd aws-url-shortner
```

### 2. Initialize Go Module

```bash
go mod init github.com/aayushxrj/aws-url-shortner
```

### 3. Install Dependencies

```bash
# Core dependencies
go get github.com/joho/godotenv

# AWS SDK v2
go get github.com/aws/aws-sdk-go-v2@v1.23.0 github.com/aws/aws-sdk-go-v2/config@v1.18.0 github.com/aws/aws-sdk-go-v2/service/dynamodb@v1.21.0

# DynamoDB attribute value helper
go get github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue

# Tidy up dependencies
go mod tidy
```

### 4. Generate Protocol Buffers

```bash
protoc `
  -I proto `
  --go_out=proto/gen --go_opt=paths=source_relative `
  --go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative `
  proto/main.proto
```

### 5. Configure Environment Variables

Create a `.env` file in the project root:

```env
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=us-east-1
SERVER_PORT=50051
HTTP_PORT=8080
```

### 6. Set Up DynamoDB Table

Create a DynamoDB table named `Urls` with the following schema:

- **Table Name**: `Urls`
- **Partition Key**: `short_id` (String)

**Attributes**:
- `short_id` (String) - Primary key
- `original_url` (String) - The full URL to redirect to
- `created_at` (String) - Timestamp of creation
- `expire_at` (Number) - Unix timestamp for expiration
- `clicks` (Number) - Click counter

### 7. Run the Application

#### Local Development

```bash
go run cmd/grpcapi/server.go
```

The application will start:
- gRPC Server on port `50051`
- HTTP Redirect Server on port `8080`

#### Using Docker

##### Option 1: Simple Docker Run

Build the Docker image:

```bash
docker build -t url-shortener:latest .
```

Run the container:

```bash
docker run --rm `
  -p 50051:50051 `
  -p 8080:8080 `
  -e "AWS_ACCESS_KEY_ID=your_access_key" `
  -e "AWS_SECRET_ACCESS_KEY=your_secret_key" `
  -e "AWS_REGION=us-east-1" `
  -e "SERVER_PORT=50051" `
  url-shortener:latest
```

##### Option 2: Docker with Networking and Envoy Proxy (Recommended)

This setup creates a Docker network and runs both the backend service and Envoy proxy for gRPC-Web support.

**Step 1: Create Docker Network**
```bash
docker network create url-net
```

**Step 2: Build the Application Image**
```bash
docker build -t url-shortener:latest .
```

**Step 3: Run the Backend Service**
```bash
docker rm -f url-backend 2>$null

docker run -d --name url-backend `
  --network url-net `
  -p 50051:50051 `
  -p 8080:8080 `
  -e "AWS_ACCESS_KEY_ID=your_access_key" `
  -e "AWS_SECRET_ACCESS_KEY=your_secret_key" `
  -e "AWS_REGION=eu-north-1" `
  -e "SERVER_PORT=50051" `
  -e "HTTP_PORT=8080" `
  url-shortener:latest
```

**Step 4: Run Envoy Proxy**
```bash
docker rm -f envoy-proxy2 envoy-proxy 2>$null

docker run -d --name envoy-proxy `
  --network url-net `
  -p 8081:8081 `
  -p 9901:9901 `
  -v ${PWD}/envoy.yaml:/etc/envoy/envoy.yaml:ro `
  envoyproxy/envoy:v1.31.0
```

**Step 5: View Logs**
```bash
# Backend logs
docker logs -f url-backend

# Envoy proxy logs
docker logs -f envoy-proxy
```

**Ports Exposed:**
- `50051` - gRPC server
- `8080` - HTTP redirect server
- `8081` - Envoy gRPC-Web proxy
- `9901` - Envoy admin interface

## 📡 API Reference

### gRPC Service Methods

#### 1. ShortenURL
Create a short URL for a given long URL.

```protobuf
rpc ShortenURL (ShortenURLRequest) returns (ShortenURLResponse);
```

#### 2. GetOriginalURL
Retrieve the original URL using the short ID.

```protobuf
rpc GetOriginalURL (GetOriginalURLRequest) returns (GetOriginalURLResponse);
```

#### 3. IncrementClick
Increment click counter whenever a short link is used.

```protobuf
rpc IncrementClick (IncrementClickRequest) returns (IncrementClickResponse);
```

#### 4. HealthCheck
Health check endpoint.

```protobuf
rpc HealthCheck (HealthCheckRequest) returns (HealthCheckResponse);
```

#### 5. GetURLStats
Get analytics and metadata for a specific URL.

```protobuf
rpc GetURLStats (GetURLStatsRequest) returns (GetURLStatsResponse);
```

#### 6. UpdateURL
Update an existing short URL (change destination or expiry).

```protobuf
rpc UpdateURL (UpdateURLRequest) returns (UpdateURLResponse);
```

#### 7. DeleteURL
Delete a short URL by ID.

```protobuf
rpc DeleteURL (DeleteURLRequest) returns (DeleteURLResponse);
```

#### 8. ListAllURLs
List all shortened URLs with optional pagination.

```protobuf
rpc ListAllURLs (ListAllURLsRequest) returns (ListAllURLsResponse);
```

### HTTP Endpoint

**Redirect Endpoint**: `GET /{short_id}`

Redirects to the original URL and increments the click counter asynchronously.

## 🗂️ Project Structure

```
aws-url-shortner/
├── cmd/
│   └── grpcapi/
│       └── server.go           # Main server entry point
├── internals/
│   ├── api/
│   │   ├── handlers/           # gRPC handler implementations
│   │   │   ├── server_struct.go
│   │   │   ├── url_handler.go
│   │   │   └── http_redirect_handler.go
│   │   └── interceptors/       # gRPC interceptors (middleware)
│   ├── models/                 # Data models
│   └── repository/
│       └── db/
│           └── dynamo.go       # DynamoDB client and operations
├── pkg/
│   └── utils/
│       └── error_handler.go    # Error handling utilities
├── proto/
│   ├── main.proto              # Protocol buffer definitions
│   ├── gen/                    # Generated protobuf code
│   │   ├── main.pb.go
│   │   └── main_grpc.pb.go
│   └── validate/               # Validation rules
├── Dockerfile                  # Multi-stage Docker build
├── envoy.yaml                  # Envoy proxy configuration (gRPC-Web)
├── go.mod                      # Go module dependencies
├── go.sum                      # Dependency checksums
└── README.md                   # This file
```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `AWS_ACCESS_KEY_ID` | AWS access key for DynamoDB | Required |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key for DynamoDB | Required |
| `AWS_REGION` | AWS region for DynamoDB | Required |
| `SERVER_PORT` | gRPC server port | `50051` |
| `HTTP_PORT` | HTTP redirect server port | `8080` |

### CORS Configuration

The HTTP server includes CORS middleware that allows requests from:
- `http://localhost:5173` (Vite default)
- Other origins can be configured in `cmd/grpcapi/server.go`

### Envoy Proxy

The included `envoy.yaml` configures:
- gRPC-Web support for browser clients
- CORS headers for cross-origin requests
- HTTP/2 for gRPC communication
- Port `8081` for gRPC-Web endpoint

## 🧪 Testing

Run tests:

```bash
go test ./...
```

Run with coverage:

```bash
go test -cover ./...
```

## 🚢 Deployment

### Docker Deployment

The project includes a multi-stage Dockerfile for optimized production builds:

1. **Build stage**: Compiles the Go binary
2. **Run stage**: Uses distroless image for minimal attack surface

### AWS ECR (Elastic Container Registry) Deployment

Deploy your Docker image to AWS ECR for production use:

**Step 1: Configure AWS CLI**
```bash
aws configure
```

**Step 2: Authenticate Docker to ECR**
```bash
aws ecr get-login-password --region eu-north-1 | docker login --username AWS --password-stdin 123456789012.dkr.ecr.eu-north-1.amazonaws.com
```

**Step 3: Tag Your Image**
```bash
docker tag url-shortener:latest 123456789012.dkr.ecr.eu-north-1.amazonaws.com/url-shortener:latest
```

**Step 4: Push to ECR**
```bash
docker push 123456789012.dkr.ecr.eu-north-1.amazonaws.com/url-shortener:latest
```

**ECR Repository**: `123456789012.dkr.ecr.eu-north-1.amazonaws.com/url-shortener`

> **Note**: Replace the AWS account ID and region with your own ECR repository details.

### AWS Deployment Considerations

- **DynamoDB**: Ensure proper IAM permissions for table access
- **Networking**: Configure security groups for gRPC (50051) and HTTP (8080) ports
- **Scaling**: DynamoDB auto-scaling can be configured based on traffic
- **Monitoring**: Use CloudWatch for logs and metrics
- **ECR**: Use ECR for secure container image storage and versioning
- **ECS/EKS**: Deploy containers using Amazon ECS or EKS for production orchestration

## 🛠️ Development Commands

### Initialize Project
```bash
go mod init github.com/aayushxrj/aws-url-shortner
```

### Install Dependencies
```bash
go get github.com/joho/godotenv
go get github.com/aws/aws-sdk-go-v2@v1.23.0 github.com/aws/aws-sdk-go-v2/config@v1.18.0 github.com/aws/aws-sdk-go-v2/service/dynamodb@v1.21.0
go get github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue
go mod tidy
```

### Generate Protobuf Code
```bash
protoc `
  -I proto `
  --go_out=proto/gen --go_opt=paths=source_relative `
  --go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative `
  proto/main.proto
```

### Docker Commands

#### Quick Start (Simple)
```bash
docker build -t url-shortener:latest .

docker run --rm `
  -p 50051:50051 `
  -p 8080:8080 `
  -e "AWS_ACCESS_KEY_ID=your_key" `
  -e "AWS_SECRET_ACCESS_KEY=your_secret" `
  -e "AWS_REGION=us-east-1" `
  -e "SERVER_PORT=50051" `
  url-shortener:latest
```

#### Full Setup with Docker Network and Envoy
```bash
# Create network
docker network create url-net

# Build image
docker build -t url-shortener:latest .

# Remove existing containers (if any)
docker rm -f url-backend 2>$null

# Run backend service
docker run -d --name url-backend `
  --network url-net `
  -p 50051:50051 `
  -p 8080:8080 `
  -e "AWS_ACCESS_KEY_ID=your_access_key" `
  -e "AWS_SECRET_ACCESS_KEY=your_secret_key" `
  -e "AWS_REGION=eu-north-1" `
  -e "SERVER_PORT=50051" `
  -e "HTTP_PORT=8080" `
  url-shortener:latest

# Remove existing Envoy containers (if any)
docker rm -f envoy-proxy2 envoy-proxy 2>$null

# Run Envoy proxy
docker run -d --name envoy-proxy `
  --network url-net `
  -p 8081:8081 `
  -p 9901:9901 `
  -v ${PWD}/envoy.yaml:/etc/envoy/envoy.yaml:ro `
  envoyproxy/envoy:v1.31.0

# View logs
docker logs -f url-backend
docker logs -f envoy-proxy
```

## 📝 TODO

- [ ] Implement TLS/SSL for secure communication
- [ ] Add custom domain support for short URLs
- [ ] Implement rate limiting
- [ ] Add authentication and authorization
- [ ] Create comprehensive test suite
- [ ] Add metrics and observability (Prometheus/Grafana)
- [ ] Implement URL validation and sanitization
- [ ] Add bulk URL operations
- [ ] Create admin dashboard
- [ ] Implement URL analytics export

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 👤 Author

**Aayush** - [@aayushxrj](https://github.com/aayushxrj)

## 🙏 Acknowledgments

- AWS SDK for Go v2
- gRPC and Protocol Buffers
- Envoy Proxy for gRPC-Web support
- The Go community

## 📞 Support

For support, please open an issue in the GitHub repository.

---

**Note**: Remember to keep your AWS credentials secure and never commit them to version control. Always use environment variables or AWS IAM roles for production deployments.