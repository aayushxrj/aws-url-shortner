package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/aayushxrj/aws-url-shortner/internals/api/handlers"
	"github.com/aayushxrj/aws-url-shortner/internals/repository/db"
	pb "github.com/aayushxrj/aws-url-shortner/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load env
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatalf("Error loading .env file: %v", err)
	// }

	//TODO
	// Implement TLS

	// Connect Database
	client, err := db.NewDynamoClient()
	if err != nil {
		log.Fatal("Error:", err)
	}
	fmt.Println("✅ Connected to DynamoDB successfully!", client.DB)

	// Start gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterUrlShortenerServer(grpcServer, &handlers.Server{DB: client})
	reflection.Register(grpcServer)

	grpcPort := os.Getenv("SERVER_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		fmt.Printf("gRPC server is running on port %s\n", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Start HTTP redirect server
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// Function to query DynamoDB for a short URL. It also increments the click
	// counter asynchronously when a URL is found so redirects remain fast.
	getLongURL := func(shortKey string) (string, bool) {
		ctx := context.Background()
		longURL, err := client.GetLongURL(ctx, shortKey)
		if err != nil || longURL == "" {
			return "", false
		}

		// increment click count in background; log error if it fails
		go func(k string) {
			if err := client.IncrementClick(ctx, k); err != nil {
				log.Printf("failed to increment click for %s: %v", k, err)
			}
		}(shortKey)

		return longURL, true
	}

	// Wrap redirect handler with CORS middleware so browser preflight (OPTIONS)
	// requests receive Access-Control-Allow-* headers. This helps when the
	// frontend mistakenly calls the backend HTTP port directly (8080) instead
	// of going through Envoy gRPC-Web proxy.
	redirectHandler := handlers.RedirectHandler(getLongURL)
	http.HandleFunc("/", corsMiddleware(redirectHandler))

	fmt.Printf("HTTP redirect server is running on port %s\n", httpPort)
	if err := http.ListenAndServe(":"+httpPort, nil); err != nil {
		log.Fatalf("Failed to serve HTTP: %v", err)
	}
}

// corsMiddleware sets CORS headers and handles OPTIONS preflight requests.
// It reads the request Origin and allows it if it's in the allowed list.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	// allowed origins for dev -- adjust as needed or make configurable via env
	allowed := map[string]bool{
		"http://localhost:5173": true,
		"http://127.0.0.1:5173": true,
		"http://localhost:3000": true,
		"http://127.0.0.1:3000": true,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// no origin, just proceed
			next(w, r)
			return
		}

		if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Grpc-Web, X-User-Agent, grpc-timeout, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "grpc-status, grpc-message")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			// preflight request - respond and return
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
