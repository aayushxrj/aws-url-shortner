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

	// Function to query DynamoDB for a short URL
	getLongURL := func(shortKey string) (string, bool) {
		longURL, err := client.GetLongURL(context.Background(), shortKey)
		if err != nil || longURL == "" {
			return "", false
		}
		return longURL, true
	}

	http.HandleFunc("/", handlers.RedirectHandler(getLongURL))

	fmt.Printf("HTTP redirect server is running on port %s\n", httpPort)
	if err := http.ListenAndServe(":"+httpPort, nil); err != nil {
		log.Fatalf("Failed to serve HTTP: %v", err)
	}
}
