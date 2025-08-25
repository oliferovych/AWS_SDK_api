package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var db *dynamodb.Client

func init() {
	// Load AWS config (uses Lambda execution role by default)
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load AWS SDK config, %v", err))
	}
	db = dynamodb.NewFromConfig(cfg)
}


type Car struct {
	ID		string `json:"id"`
	Make	string `json:"make"`
	Model	string `json:"model"`
	Year	int    `json:"year"`
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	fmt.Println("Received request:", req.RequestContext.HTTP.Method, req.RequestContext.HTTP.Path)
	fmt.Printf("Raw request body: %s\n", req.Body)
	switch req.RequestContext.HTTP.Method {
	case "GET":
		return handleGet(ctx, req)
	case "POST":
		return handlePost(ctx, req)
	default:
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusMethodNotAllowed,
			Body:       "method not allowed",
		}, nil
	}
}



func handleGet(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	id := req.QueryStringParameters["id"]
	TableNameEnv := os.Getenv("TABLE_NAME")
	if TableNameEnv == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "TABLE_NAME environment variable is not set",
		}, nil
	}

	if id == "" {
		// No id provided, scan the whole table
		out, err := db.Scan(ctx, &dynamodb.ScanInput{
			TableName: &TableNameEnv,
		})
		if err != nil {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       err.Error(),
			}, nil
		}
		cars := []Car{}
		for _, item := range out.Items {
			year := 0
			if y, ok := item["Year"].(*types.AttributeValueMemberN); ok {
				year, _ = strconv.Atoi(y.Value)
			}
			cars = append(cars, Car{
				ID:    item["ID"].(*types.AttributeValueMemberS).Value,
				Make:  item["Make"].(*types.AttributeValueMemberS).Value,
				Model: item["Model"].(*types.AttributeValueMemberS).Value,
				Year:  year,
			})
		}
		body, _ := json.Marshal(cars)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusOK,
			Body:       string(body),
			Headers:    map[string]string{"Content-Type": "application/json"},
		}, nil
	}

	// id provided, get single item
	out, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &TableNameEnv,
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, nil
	}
	if out.Item == nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusNotFound,
			Body:       "item not found",
		}, nil
	}
	year := 0
	if y, ok := out.Item["Year"].(*types.AttributeValueMemberN); ok {
		year, _ = strconv.Atoi(y.Value)
	} else {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "invalid year field",
		}, nil
	}
	item := Car{
		ID:    out.Item["ID"].(*types.AttributeValueMemberS).Value,
		Make:  out.Item["Make"].(*types.AttributeValueMemberS).Value,
		Model: out.Item["Model"].(*types.AttributeValueMemberS).Value,
		Year:  year,
	}
	body, _ := json.Marshal(item)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusOK,
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func handlePost(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var item Car
	if err := json.Unmarshal([]byte(req.Body), &item); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "invalid request body",
		}, nil
	}

	TableNameEnv := os.Getenv("TABLE_NAME")
	if TableNameEnv == "" {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "TABLE_NAME environment variable is not set",
		}, nil
	}

	_, err := db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &TableNameEnv,
		Item: map[string]types.AttributeValue{
			"ID":   &types.AttributeValueMemberS{Value: item.ID},
			"Make": &types.AttributeValueMemberS{Value: item.Make},
			"Model": &types.AttributeValueMemberS{Value: item.Model},
			"Year":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", item.Year)},
		},
	})
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusCreated,
		Body:       fmt.Sprintf("item %s created", item.ID),
		Headers:    map[string]string{"Content-Type": "text/plain"},
	}, nil
}

// Helpers
func serverError(err error) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       err.Error(),
	}, nil
}

func clientError(status int, msg string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       msg,
	}, nil
}

func notFound() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusNotFound,
		Body:       "item not found",
	}, nil
}

func main() {
	lambda.Start(handler)
}
