package handlers

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	mainpb "github.com/aayushxrj/aws-url-shortner/proto/gen"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

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

func awsInt32(v int32) *int32 {
	return &v
}

// ✅ Health Check RPC — verifies DynamoDB connection
func (s *Server) HealthCheck(ctx context.Context, req *mainpb.HealthCheckRequest) (*mainpb.HealthCheckResponse, error) {
	_, err := s.DB.DB.ListTables(ctx, &dynamodb.ListTablesInput{Limit: awsInt32(1)})
	if err != nil {
		return &mainpb.HealthCheckResponse{Status: "unhealthy"}, err
	}
	return &mainpb.HealthCheckResponse{Status: "ok"}, nil
}

// ✅ Get stats for one URL (short_id)
func (s *Server) GetURLStats(ctx context.Context, req *mainpb.GetURLStatsRequest) (*mainpb.GetURLStatsResponse, error) {
	tableName := "Urls"

	out, err := s.DB.DB.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &tableName,
		Key: map[string]types.AttributeValue{
			"short_id": &types.AttributeValueMemberS{Value: req.ShortId},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %v", err)
	}

	if out.Item == nil {
		return nil, fmt.Errorf("short_id %s not found", req.ShortId)
	}

	var item struct {
		ShortID     string `dynamodbav:"short_id"`
		OriginalURL string `dynamodbav:"original_url"`
		CreatedAt   string `dynamodbav:"created_at"`
		ExpireAt    int64  `dynamodbav:"expire_at"`
		Clicks      int64  `dynamodbav:"clicks"`
	}

	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %v", err)
	}

	return &mainpb.GetURLStatsResponse{
		ShortId:     item.ShortID,
		OriginalUrl: item.OriginalURL,
		Clicks:      item.Clicks,
		CreatedAt:   item.CreatedAt,
		ExpireAt:    item.ExpireAt,
	}, nil
}

// ✅ Update existing URL (destination or expiry)
func (s *Server) UpdateURL(ctx context.Context, req *mainpb.UpdateURLRequest) (*mainpb.UpdateURLResponse, error) {
	tableName := "Urls"

	expr := "SET"
	attrs := map[string]types.AttributeValue{}
	exprNames := map[string]string{}

	if req.NewOriginalUrl != "" {
		expr += " #url = :u"
		exprNames["#url"] = "original_url"
		attrs[":u"] = &types.AttributeValueMemberS{Value: req.NewOriginalUrl}
	}

	if req.NewExpireInSeconds > 0 {
		if len(attrs) > 0 {
			expr += ","
		}
		expireAt := time.Now().Add(time.Duration(req.NewExpireInSeconds) * time.Second).Unix()
		expr += " #exp = :e"
		exprNames["#exp"] = "expire_at"
		attrs[":e"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", expireAt)}
	}

	if len(attrs) == 0 {
		return &mainpb.UpdateURLResponse{Success: false, Message: "No update fields provided"}, nil
	}

	_, err := s.DB.DB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 &tableName,
		Key:                       map[string]types.AttributeValue{"short_id": &types.AttributeValueMemberS{Value: req.ShortId}},
		UpdateExpression:          &expr,
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: attrs,
	})
	if err != nil {
		return &mainpb.UpdateURLResponse{Success: false, Message: err.Error()}, err
	}

	return &mainpb.UpdateURLResponse{Success: true, Message: "URL updated successfully"}, nil
}

// ✅ Delete short URL
func (s *Server) DeleteURL(ctx context.Context, req *mainpb.DeleteURLRequest) (*mainpb.DeleteURLResponse, error) {
	tableName := "Urls"

	_, err := s.DB.DB.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &tableName,
		Key: map[string]types.AttributeValue{
			"short_id": &types.AttributeValueMemberS{Value: req.ShortId},
		},
	})
	if err != nil {
		return &mainpb.DeleteURLResponse{Success: false, Message: err.Error()}, err
	}

	return &mainpb.DeleteURLResponse{Success: true, Message: "URL deleted successfully"}, nil
}

// ✅ List all shortened URLs
func (s *Server) ListAllURLs(ctx context.Context, req *mainpb.ListAllURLsRequest) (*mainpb.ListAllURLsResponse, error) {
	tableName := "Urls"

	input := &dynamodb.ScanInput{
		TableName: &tableName,
		Limit:     awsInt32(int32(req.Limit)),
	}

	if req.LastEvaluatedKey != "" {
		input.ExclusiveStartKey = map[string]types.AttributeValue{
			"short_id": &types.AttributeValueMemberS{Value: req.LastEvaluatedKey},
		}
	}

	out, err := s.DB.DB.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan table: %v", err)
	}

	var urls []struct {
		ShortID     string `dynamodbav:"short_id"`
		OriginalURL string `dynamodbav:"original_url"`
		CreatedAt   string `dynamodbav:"created_at"`
		ExpireAt    int64  `dynamodbav:"expire_at"`
		Clicks      int64  `dynamodbav:"clicks"`
	}

	if err := attributevalue.UnmarshalListOfMaps(out.Items, &urls); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %v", err)
	}

	pbUrls := make([]*mainpb.UrlItem, 0, len(urls))
	for _, u := range urls {
		pbUrls = append(pbUrls, &mainpb.UrlItem{
			ShortId:     u.ShortID,
			OriginalUrl: u.OriginalURL,
			CreatedAt:   u.CreatedAt,
			ExpireAt:    u.ExpireAt,
			Clicks:      u.Clicks,
		})
	}

	var lastKey string
	if val, ok := out.LastEvaluatedKey["short_id"]; ok {
		lastKey = val.(*types.AttributeValueMemberS).Value
	}

	return &mainpb.ListAllURLsResponse{
		Urls:              pbUrls,
		LastEvaluatedKey:  lastKey,
	}, nil
}
