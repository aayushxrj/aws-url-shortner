package handlers

import (
	"context"
	"fmt"
	"time"
	"math/rand"

	"github.com/aayushxrj/aws-url-shortner/internals/repository/db"
	mainpb "github.com/aayushxrj/aws-url-shortner/proto/gen"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Server struct {
	mainpb.UnimplementedUrlShortenerServer
	DB *db.DynamoClient
}

// random short id generator
func generateShortID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// ShortenURL creates a new short URL
func (s *Server) ShortenURL(ctx context.Context, req *mainpb.ShortenURLRequest) (*mainpb.ShortenURLResponse, error) {
	shortID := generateShortID(6)
	now := time.Now()
	expireAt := now.Add(time.Duration(req.ExpireInSeconds) * time.Second).Unix()

	item := map[string]types.AttributeValue{
		"short_id":     &types.AttributeValueMemberS{Value: shortID},
		"original_url": &types.AttributeValueMemberS{Value: req.OriginalUrl},
		"created_at":   &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"expire_at":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", expireAt)},
		"clicks":       &types.AttributeValueMemberN{Value: "0"},
	}

	_, err := s.DB.DB.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: awsString("Urls"),
		Item:      item,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert item: %v", err)
	}

	return &mainpb.ShortenURLResponse{
		ShortId:   shortID,
		ShortUrl:  fmt.Sprintf("http://localhost:8080/s/%s", shortID),
		CreatedAt: now.Format(time.RFC3339),
		ExpireAt:  expireAt,
	}, nil
}

// GetOriginalURL fetches the long URL from short ID
func (s *Server) GetOriginalURL(ctx context.Context, req *mainpb.GetOriginalURLRequest) (*mainpb.GetOriginalURLResponse, error) {
	out, err := s.DB.DB.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: awsString("Urls"),
		Key: map[string]types.AttributeValue{
			"short_id": &types.AttributeValueMemberS{Value: req.ShortId},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %v", err)
	}

	if out.Item == nil {
		return nil, fmt.Errorf("short_id not found")
	}

	var data struct {
		OriginalUrl string `dynamodbav:"original_url"`
	}
	err = attributevalue.UnmarshalMap(out.Item, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %v", err)
	}

	return &mainpb.GetOriginalURLResponse{
		OriginalUrl: data.OriginalUrl,
	}, nil
}

// IncrementClick increases click counter
func (s *Server) IncrementClick(ctx context.Context, req *mainpb.IncrementClickRequest) (*mainpb.IncrementClickResponse, error) {
	out, err := s.DB.DB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: awsString("Urls"),
		Key: map[string]types.AttributeValue{
			"short_id": &types.AttributeValueMemberS{Value: req.ShortId},
		},
		UpdateExpression: awsString("SET clicks = clicks + :incr"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":incr": &types.AttributeValueMemberN{Value: "1"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update click count: %v", err)
	}

	var data struct {
		Clicks int64 `dynamodbav:"clicks"`
	}
	err = attributevalue.UnmarshalMap(out.Attributes, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated data: %v", err)
	}

	return &mainpb.IncrementClickResponse{
		Clicks: data.Clicks,
	}, nil
}

func awsString(v string) *string {
	return &v
}
