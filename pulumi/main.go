package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigatewayv2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// Create a DynamoDB table
		table, err := dynamodb.NewTable(ctx, "MyItems", &dynamodb.TableArgs{
			Attributes: dynamodb.TableAttributeArray{
				&dynamodb.TableAttributeArgs{
					Name: pulumi.String("ID"),
					Type: pulumi.String("S"),
				},
			},
			HashKey:     pulumi.String("ID"),
			BillingMode: pulumi.String("PAY_PER_REQUEST"),
		})
		if err != nil {
			return err
		}

		// IAM Role for Lambda
		lambdaRole, err := iam.NewRole(ctx, "lambdaRole", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Action": "sts:AssumeRole",
					"Principal": {
						"Service": "lambda.amazonaws.com"
					},
					"Effect": "Allow",
					"Sid": ""
				}]
			}`),
		})
		if err != nil {
			return err
		}

		// Attach policies to Lambda
		_, err = iam.NewRolePolicyAttachment(ctx, "lambdaBasicExec", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
		})
		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "lambdaDynamoAccess", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess"),
		})
		if err != nil {
			return err
		}

		// Create the Lambda function
		myLambda, err := lambda.NewFunction(ctx, "myApiLambda", &lambda.FunctionArgs{
			Runtime: pulumi.String("provided.al2023"),
			Handler: pulumi.String("bootstrap"),
			Code:    pulumi.NewFileArchive("../lambda/bootstrap.zip"),
			Role:    lambdaRole.Arn,
			Environment: &lambda.FunctionEnvironmentArgs{
				Variables: pulumi.StringMap{
					"TABLE_NAME": table.Name, // dynamic table name
				},
			},
		})
		if err != nil {
			return err
		}

		// API Gateway
		api, err := apigatewayv2.NewApi(ctx, "httpApi", &apigatewayv2.ApiArgs{
			ProtocolType: pulumi.String("HTTP"),
		})
		if err != nil {
			return err
		}

		integration, err := apigatewayv2.NewIntegration(ctx, "apiIntegration", &apigatewayv2.IntegrationArgs{
			ApiId:                api.ID(),
			IntegrationType:      pulumi.String("AWS_PROXY"),
			IntegrationUri:       myLambda.Arn,
			PayloadFormatVersion: pulumi.String("2.0"),
		})
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "apigwPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  myLambda.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("%s/*/*", api.ExecutionArn),
		})
		if err != nil {
			return err
		}

		_, err = apigatewayv2.NewRoute(ctx, "apiRoute", &apigatewayv2.RouteArgs{
			ApiId:    api.ID(),
			RouteKey: pulumi.String("$default"),
			Target:   pulumi.Sprintf("integrations/%s", integration.ID()),
		})
		if err != nil {
			return err
		}

		_, err = apigatewayv2.NewRoute(ctx, "getRoute", &apigatewayv2.RouteArgs{
			ApiId:    api.ID(),
			RouteKey: pulumi.String("GET /"),
			Target:   pulumi.Sprintf("integrations/%s", integration.ID()),
		})
		if err != nil {
			return err
		}

		_, err = apigatewayv2.NewRoute(ctx, "postRoute", &apigatewayv2.RouteArgs{
			ApiId:    api.ID(),
			RouteKey: pulumi.String("POST /"),
			Target:   pulumi.Sprintf("integrations/%s", integration.ID()),
		})
		if err != nil {
			return err
		}

		stage, err := apigatewayv2.NewStage(ctx, "apiStage", &apigatewayv2.StageArgs{
			ApiId:      api.ID(),
			AutoDeploy: pulumi.Bool(true),
			Name:       pulumi.String("$default"),
		})
		if err != nil {
			return err
		}

		ctx.Export("apiUrl", pulumi.Sprintf("%s/%s", api.ApiEndpoint, stage.Name))
		ctx.Export("tableName", table.Name)

		return nil
	})
}
