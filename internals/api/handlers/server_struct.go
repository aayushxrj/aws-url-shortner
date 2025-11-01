package handlers

import (
	mainpb "github.com/aayushxrj/aws-url-shortner/proto/gen"
	"github.com/aayushxrj/aws-url-shortner/internals/repository/db"
)

type Server struct {
	mainpb.UnimplementedUrlShortenerServer
	DB *db.DynamoClient
}