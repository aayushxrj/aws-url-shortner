package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/aayushxrj/aws-url-shortner/internals/api/handlers"
	"github.com/aayushxrj/aws-url-shortner/internals/repository/db"
	pb "github.com/aayushxrj/aws-url-shortner/proto/gen"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	// Load env
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	//TODO
	// Implement TLS

	//TODO
	// Connect Database
	client, err := db.NewDynamoClient()
	if err != nil {
		log.Fatal("Error:", err)
	}
	fmt.Println("✅ Connected to DynamoDB successfully!", client.DB)
	
	s := grpc.NewServer()

	// pb.RegisterUrlShortenerServer(s, &handlers.Server{})
	pb.RegisterUrlShortenerServer(s, &handlers.Server{DB: client})

	reflection.Register(s)

	// go get github.com/joho/godotenv
	port := os.Getenv("SERVER_PORT")

	fmt.Printf("Server is running on port %s\n", port)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

}
