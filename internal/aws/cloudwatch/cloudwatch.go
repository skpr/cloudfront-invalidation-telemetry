package cloudwatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/smithy-go/middleware"
)

// ClientInterface is a mock cloudwatch interface.
type ClientInterface interface {
	PutMetricData(ctx context.Context, params *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error)
}

// MockClient is a mock cloudwatch client.
type MockClient struct {
	MetricData []types.MetricDatum
}

// PutMetricData mock function.
func (c *MockClient) PutMetricData(ctx context.Context, params *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error) {
	// Store the metrics for later.
	c.MetricData = append(c.MetricData, params.MetricData...)

	return &cloudwatch.PutMetricDataOutput{
		ResultMetadata: middleware.Metadata{},
	}, nil
}
