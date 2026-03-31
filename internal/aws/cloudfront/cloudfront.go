package cloudfront

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/smithy-go/middleware"
)

// ClientInterface is a mock cloudfront client.
type ClientInterface interface {
	GetDistribution(ctx context.Context, params *cloudfront.GetDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionOutput, error)
	GetInvalidation(ctx context.Context, params *cloudfront.GetInvalidationInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetInvalidationOutput, error)
	ListDistributions(ctx context.Context, params *cloudfront.ListDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error)
	ListInvalidations(ctx context.Context, params *cloudfront.ListInvalidationsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListInvalidationsOutput, error)
	ListTagsForResource(ctx context.Context, params *cloudfront.ListTagsForResourceInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListTagsForResourceOutput, error)
}

// MockClient for testing.
type MockClient struct{}

// GetDistribution mock function.
func (c MockClient) GetDistribution(ctx context.Context, params *cloudfront.GetDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionOutput, error) {
	return &cloudfront.GetDistributionOutput{
		Distribution: &types.Distribution{
			Id: aws.String("test-distribution-id"),
		},
	}, nil
}

// GetInvalidation mock function.
func (c MockClient) GetInvalidation(ctx context.Context, params *cloudfront.GetInvalidationInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetInvalidationOutput, error) {
	return &cloudfront.GetInvalidationOutput{
		Invalidation: &types.Invalidation{
			CreateTime: aws.Time(time.Now()),
			Id:         aws.String("test-invalidation-id"),
			InvalidationBatch: &types.InvalidationBatch{
				Paths: &types.Paths{
					Quantity: aws.Int32(3),
					Items: []string{
						"/test-item-one",
						"/test-item-two",
						"/test-item-three",
					},
				},
			},
			Status: aws.String("Completed"),
		},
		ResultMetadata: middleware.Metadata{},
	}, nil
}

// ListDistributions mock function.
func (c MockClient) ListDistributions(ctx context.Context, params *cloudfront.ListDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error) {
	return &cloudfront.ListDistributionsOutput{
		DistributionList: &types.DistributionList{
			Items: []types.DistributionSummary{
				{
					Id: aws.String("test-distribution-id"),
				},
			},
		},
		ResultMetadata: middleware.Metadata{},
	}, nil
}

// ListInvalidations mock function.
func (c MockClient) ListInvalidations(ctx context.Context, params *cloudfront.ListInvalidationsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListInvalidationsOutput, error) {
	return &cloudfront.ListInvalidationsOutput{
		InvalidationList: &types.InvalidationList{
			Items: []types.InvalidationSummary{
				{
					Id:         aws.String("test-invalidation-id"),
					Status:     aws.String("Completed"),
					CreateTime: aws.Time(time.Now()),
				},
			},
		},
		ResultMetadata: middleware.Metadata{},
	}, nil
}

// ListTagsForResource mock function.
func (c MockClient) ListTagsForResource(ctx context.Context, params *cloudfront.ListTagsForResourceInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListTagsForResourceOutput, error) {
	return &cloudfront.ListTagsForResourceOutput{
		Tags: &types.Tags{
			Items: []types.Tag{
				{
					Key:   aws.String("skpr.io/cloudfront-invalidation-telemetry/logs/group"),
					Value: aws.String("/aws/cloudfront/test"),
				},
				{
					Key:   aws.String("skpr.io/cloudfront-invalidation-telemetry/logs/stream"),
					Value: aws.String("test-stream"),
				},
			},
		},
		ResultMetadata: middleware.Metadata{},
	}, nil
}
