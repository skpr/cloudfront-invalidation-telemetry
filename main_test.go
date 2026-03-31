package main

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/skpr/yolog"
	"github.com/stretchr/testify/assert"

	cloudfrontclient "github.com/skpr/cloudfront-invalidation-telemetry/internal/aws/cloudfront"
	cloudwatchclient "github.com/skpr/cloudfront-invalidation-telemetry/internal/aws/cloudwatch"
	"github.com/skpr/cloudfront-invalidation-telemetry/internal/metrics"
)

func TestLogConfigEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   LogConfig
		expected bool
	}{
		{
			name:     "BothSet",
			config:   LogConfig{GroupName: "/aws/cloudfront/test", StreamName: "test-stream"},
			expected: true,
		},
		{
			name:     "MissingGroupName",
			config:   LogConfig{StreamName: "test-stream"},
			expected: false,
		},
		{
			name:     "MissingStreamName",
			config:   LogConfig{GroupName: "/aws/cloudfront/test"},
			expected: true,
		},
		{
			name:     "BothEmpty",
			config:   LogConfig{},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.config.Enabled())
		})
	}
}

func TestGetLogConfig(t *testing.T) {
	tests := []struct {
		name     string
		tags     *cloudfront.ListTagsForResourceOutput
		expected LogConfig
	}{
		{
			name: "BothTagsPresent",
			tags: &cloudfront.ListTagsForResourceOutput{
				Tags: &types.Tags{
					Items: []types.Tag{
						{Key: aws.String(TagLogsGroupName), Value: aws.String("/aws/cloudfront/prod")},
						{Key: aws.String(TagLogsStreamName), Value: aws.String("my-stream")},
					},
				},
			},
			expected: LogConfig{GroupName: "/aws/cloudfront/prod", StreamName: "my-stream"},
		},
		{
			name: "OnlyGroupTag",
			tags: &cloudfront.ListTagsForResourceOutput{
				Tags: &types.Tags{
					Items: []types.Tag{
						{Key: aws.String(TagLogsGroupName), Value: aws.String("/aws/cloudfront/prod")},
					},
				},
			},
			expected: LogConfig{GroupName: "/aws/cloudfront/prod", StreamName: "cloudfront-invalidations"},
		},
		{
			name: "OnlyStreamTag",
			tags: &cloudfront.ListTagsForResourceOutput{
				Tags: &types.Tags{
					Items: []types.Tag{
						{Key: aws.String(TagLogsStreamName), Value: aws.String("my-stream")},
					},
				},
			},
			expected: LogConfig{StreamName: "my-stream"},
		},
		{
			name: "NoRelevantTags",
			tags: &cloudfront.ListTagsForResourceOutput{
				Tags: &types.Tags{
					Items: []types.Tag{
						{Key: aws.String("Environment"), Value: aws.String("production")},
					},
				},
			},
			expected: LogConfig{StreamName: "cloudfront-invalidations"},
		},
		{
			name: "EmptyTags",
			tags: &cloudfront.ListTagsForResourceOutput{
				Tags: &types.Tags{
					Items: []types.Tag{},
				},
			},
			expected: LogConfig{StreamName: "cloudfront-invalidations"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, getLogConfig(tc.tags))
		})
	}
}

// mockCloudFrontClient allows configuring responses per test.
type mockCloudFrontClient struct {
	cloudfrontclient.MockClient
	tags          *cloudfront.ListTagsForResourceOutput
	invalidations []types.InvalidationSummary
	invalidation  *types.Invalidation
}

func (m *mockCloudFrontClient) ListTagsForResource(ctx context.Context, params *cloudfront.ListTagsForResourceInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListTagsForResourceOutput, error) {
	return m.tags, nil
}

func (m *mockCloudFrontClient) ListInvalidations(ctx context.Context, params *cloudfront.ListInvalidationsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListInvalidationsOutput, error) {
	return &cloudfront.ListInvalidationsOutput{
		InvalidationList: &types.InvalidationList{
			Items: m.invalidations,
		},
		ResultMetadata: middleware.Metadata{},
	}, nil
}

func (m *mockCloudFrontClient) GetInvalidation(ctx context.Context, params *cloudfront.GetInvalidationInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetInvalidationOutput, error) {
	return &cloudfront.GetInvalidationOutput{
		Invalidation: m.invalidation,
	}, nil
}

func (m *mockCloudFrontClient) ListDistributions(ctx context.Context, params *cloudfront.ListDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error) {
	return &cloudfront.ListDistributionsOutput{
		DistributionList: &types.DistributionList{
			Items: []types.DistributionSummary{
				{
					Id:  aws.String("dist-1"),
					ARN: aws.String("arn:aws:cloudfront::123456789012:distribution/dist-1"),
				},
			},
		},
		ResultMetadata: middleware.Metadata{},
	}, nil
}

func TestRunMetricBucketing(t *testing.T) {
	now := time.Now()
	minute1 := now.Truncate(time.Minute)
	minute2 := minute1.Add(-time.Minute)

	cfClient := &mockCloudFrontClient{
		tags: &cloudfront.ListTagsForResourceOutput{
			Tags: &types.Tags{Items: []types.Tag{}},
		},
		invalidations: []types.InvalidationSummary{
			{Id: aws.String("inv-1"), Status: aws.String("Completed"), CreateTime: aws.Time(minute1.Add(10 * time.Second))},
			{Id: aws.String("inv-2"), Status: aws.String("Completed"), CreateTime: aws.Time(minute1.Add(20 * time.Second))},
			{Id: aws.String("inv-3"), Status: aws.String("Completed"), CreateTime: aws.Time(minute2.Add(30 * time.Second))},
		},
		invalidation: &types.Invalidation{
			CreateTime: aws.Time(now),
			Id:         aws.String("inv-detail"),
			InvalidationBatch: &types.InvalidationBatch{
				Paths: &types.Paths{
					Quantity: aws.Int32(2),
					Items:    []string{"/path-a", "/path-b"},
				},
			},
			Status: aws.String("Completed"),
		},
	}

	cwMock := &cloudwatchclient.MockClient{}
	metricsClient, err := metrics.New(cwMock, "test/namespace")
	assert.NoError(t, err)

	logger := yolog.NewLogger("test")

	err = run(ctx(), logger, cfClient, nil, metricsClient, 10*time.Minute)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(cwMock.MetricData))

	bucketCounts := make(map[time.Time]int)
	for _, m := range cwMock.MetricData {
		bucketCounts[*m.Timestamp]++
	}

	assert.Equal(t, 2, bucketCounts[minute1])
	assert.Equal(t, 2, bucketCounts[minute2])
}

func TestRunNoLogsWithoutTags(t *testing.T) {
	now := time.Now()

	cfClient := &mockCloudFrontClient{
		tags: &cloudfront.ListTagsForResourceOutput{
			Tags: &types.Tags{Items: []types.Tag{}},
		},
		invalidations: []types.InvalidationSummary{
			{Id: aws.String("inv-1"), Status: aws.String("Completed"), CreateTime: aws.Time(now)},
		},
		invalidation: &types.Invalidation{
			CreateTime: aws.Time(now),
			Id:         aws.String("inv-1"),
			InvalidationBatch: &types.InvalidationBatch{
				Paths: &types.Paths{
					Quantity: aws.Int32(1),
					Items:    []string{"/index.html"},
				},
			},
			Status: aws.String("Completed"),
		},
	}

	cwMock := &cloudwatchclient.MockClient{}
	metricsClient, err := metrics.New(cwMock, "test/namespace")
	assert.NoError(t, err)

	logger := yolog.NewLogger("test")

	err = run(ctx(), logger, cfClient, nil, metricsClient, 10*time.Minute)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(cwMock.MetricData))
}

func TestRunSkipsOldInvalidations(t *testing.T) {
	old := time.Now().Add(-10 * time.Minute)

	cfClient := &mockCloudFrontClient{
		tags: &cloudfront.ListTagsForResourceOutput{
			Tags: &types.Tags{Items: []types.Tag{}},
		},
		invalidations: []types.InvalidationSummary{
			{Id: aws.String("inv-old"), Status: aws.String("Completed"), CreateTime: aws.Time(old)},
		},
		invalidation: &types.Invalidation{
			CreateTime: aws.Time(old),
			Id:         aws.String("inv-old"),
			InvalidationBatch: &types.InvalidationBatch{
				Paths: &types.Paths{
					Quantity: aws.Int32(1),
					Items:    []string{"/old"},
				},
			},
			Status: aws.String("Completed"),
		},
	}

	cwMock := &cloudwatchclient.MockClient{}
	metricsClient, err := metrics.New(cwMock, "test/namespace")
	assert.NoError(t, err)

	logger := yolog.NewLogger("test")

	err = run(ctx(), logger, cfClient, nil, metricsClient, 5*time.Minute)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(cwMock.MetricData))
}

func ctx() context.Context {
	return context.Background()
}
