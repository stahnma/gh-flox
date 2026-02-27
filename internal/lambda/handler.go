package lambda

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stahnma/gh-flox/internal/commands"
)

// NewHandler returns a Lambda handler function that exports data and uploads to S3.
func NewHandler(app *commands.App) func(context.Context, interface{}) (string, error) {
	return func(ctx context.Context, event interface{}) (string, error) {
		var buf bytes.Buffer
		if err := app.ExportJSON(ctx, &buf, false); err != nil {
			return "", fmt.Errorf("export: %w", err)
		}

		if buf.Len() == 0 {
			return "", fmt.Errorf("export command produced no output")
		}

		s3Bucket := os.Getenv("S3_BUCKET_NAME")
		s3ObjectKey := os.Getenv("S3_OBJECT_KEY")

		if s3Bucket == "" || s3ObjectKey == "" {
			return "", fmt.Errorf("S3_BUCKET_NAME and S3_OBJECT_KEY environment variables must be set")
		}

		date := time.Now().Format("2006-Jan-02")
		s3ObjectKey = fmt.Sprintf(s3ObjectKey, date)

		cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(os.Getenv("AWS_REGION")))
		if err != nil {
			return "", fmt.Errorf("failed to load AWS config: %w", err)
		}

		svc := s3.NewFromConfig(cfg)

		_, err = svc.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s3Bucket),
			Key:    aws.String(s3ObjectKey),
			Body:   bytes.NewReader(buf.Bytes()),
		})
		if err != nil {
			return "", fmt.Errorf("failed to upload file to S3: %w", err)
		}

		return "Lambda executed successfully and output uploaded to S3", nil
	}
}
