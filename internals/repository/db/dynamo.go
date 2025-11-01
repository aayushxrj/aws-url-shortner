package db

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/joho/godotenv"
)

type DynamoClient struct {
	DB *dynamodb.Client
}

// Load credentials from .env manually
func NewDynamoClient() (*DynamoClient, error) {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Warning: .env file not found (using system environment variables instead)")
	}

	// Get values from environment
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")

	if accessKey == "" || secretKey == "" || region == "" {
		return nil, fmt.Errorf("missing AWS credentials in environment")
	}

	// Manually create AWS config with credentials
	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
			),
		),
		config.WithEndpointResolver(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	db := dynamodb.NewFromConfig(cfg)
	return &DynamoClient{DB: db}, nil
}

// GetLongURL looks up a short key in the Urls table and returns the original URL.
func (c *DynamoClient) GetLongURL(ctx context.Context, shortKey string) (string, error) {
	out, err := c.DB.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("Urls"),
		Key: map[string]types.AttributeValue{
			"short_id": &types.AttributeValueMemberS{Value: shortKey},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get item: %w", err)
	}
	if out.Item == nil {
		return "", fmt.Errorf("short_id not found")
	}

	var data struct {
		OriginalUrl string `dynamodbav:"original_url"`
	}
	if err := attributevalue.UnmarshalMap(out.Item, &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal item: %w", err)
	}
	return data.OriginalUrl, nil
}
