package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cloudwatchlogstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/caarlos0/env/v11"
	"github.com/skpr/yolog"

	cloudfrontclient "github.com/skpr/cloudfront-invalidation-telemetry/internal/aws/cloudfront"
	"github.com/skpr/cloudfront-invalidation-telemetry/internal/metrics"
)

const (
	// YoLogStream is the log stream name for yolog logs.
	YoLogStream = "cloudfront-invalidation-telemetry"
	// CloudWatchNamespace is the CloudWatch Namespace to store metrics in.
	CloudWatchNamespace = "Skpr/CloudFront"
	// CloudWatchMetricInvalidationRequest is the CloudWatch Metric name for the count of invalidation requests.
	CloudWatchMetricInvalidationRequest = "InvalidationRequest"
	// CloudWatchMetricInvalidationPathCount is the CloudWatch Metric name for the count of paths in invalidation requests.
	CloudWatchMetricInvalidationPathCount = "InvalidationPathCounter" // @todo, Should be "Count"
	// DimensionDistribution is the CloudWatch Dimension name for the distribution ID.
	DimensionDistribution = "Distribution"
	// TagLogsGroupName is the CloudFront distribution tag key for the CloudWatch Logs group name.
	TagLogsGroupName = "skpr.io/cloudfront-invalidation-telemetry/logs/group"
	// TagLogsStreamName is the CloudFront distribution tag key for the CloudWatch Logs stream name.
	TagLogsStreamName = "skpr.io/cloudfront-invalidation-telemetry/logs/stream"
)

// LogConfig holds the CloudWatch Logs configuration extracted from distribution tags.
type LogConfig struct {
	GroupName  string
	StreamName string
}

// Enabled returns true if both the log group and stream names are set.
func (c LogConfig) Enabled() bool {
	return c.GroupName != ""
}

// getLogConfig extracts the log group and stream names from CloudFront distribution tags.
func getLogConfig(tags *cloudfront.ListTagsForResourceOutput) LogConfig {
	var config LogConfig

	for _, tag := range tags.Tags.Items {
		switch *tag.Key {
		case TagLogsGroupName:
			config.GroupName = *tag.Value
		case TagLogsStreamName:
			config.StreamName = *tag.Value
		}
	}

	if config.StreamName == "" {
		config.StreamName = "cloudfront-invalidations"
	}

	return config
}

type Config struct {
	// Window is a time.Duration to determine how far back the application should look for invalidations.
	Window time.Duration `env:"CLOUDFRONT_INVALIDATION_METRICS_WINDOW" envDefault:"5m"`
}

// InvalidationLogMessage represents a structured log entry for a CloudFront invalidation event.
type InvalidationLogMessage struct {
	InvalidationID string   `json:"invalidation_id"`
	PathCount      int32    `json:"path_count"`
	Paths          []string `json:"paths"`
}

// MetricBucket is a struct to hold the counts for invalidations and paths for a given time bucket.
// This allows us to aggregate the counts for a given time period, and then push the aggregated counts to CloudWatch as a single data point.
type MetricBucket struct {
	Invalidations float64
	Paths         float64
}

func main() {
	lambda.Start(handler)
}

// Start is an exported abstraction so that the application can be
// setup in a way that works for you, opposed to being a tightly
// coupled to provided and assumed Clients.
func handler(ctx context.Context) error {
	logger := yolog.NewLogger(YoLogStream)
	defer logger.Log(os.Stdout)

	var config Config

	err := env.Parse(&config)
	if err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr("window", config.Window.String())

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return logger.WrapError(err)
	}

	client, err := metrics.New(cloudwatch.NewFromConfig(awsConfig), CloudWatchNamespace)
	if err != nil {
		return logger.WrapError(err)
	}

	return run(ctx, logger, cloudfront.NewFromConfig(awsConfig), cloudwatchlogs.NewFromConfig(awsConfig), client, config.Window)
}

// Execute will execute the given API calls against the input Clients.
func run(ctx context.Context, logger *yolog.Logger, cloudfrontClient cloudfrontclient.ClientInterface, logsClient *cloudwatchlogs.Client, metricsClient metrics.ClientInterface, window time.Duration) error {
	distributions, err := cloudfrontClient.ListDistributions(ctx, &cloudfront.ListDistributionsInput{})
	if err != nil {
		return logger.WrapError(err)
	}

	timeAgo := time.Now().Add(-window)

	for _, distribution := range distributions.DistributionList.Items {
		tags, err := cloudfrontClient.ListTagsForResource(ctx, &cloudfront.ListTagsForResourceInput{
			Resource: distribution.ARN,
		})
		if err != nil {
			return logger.WrapError(err)
		}

		logConfig := getLogConfig(tags)

		invalidations, err := cloudfrontClient.ListInvalidations(ctx, &cloudfront.ListInvalidationsInput{
			DistributionId: distribution.Id,
		})
		if err != nil {
			return logger.WrapError(err)
		}

		var (
			buckets = make(map[time.Time]*MetricBucket)
			logs    []cloudwatchlogstypes.InputLogEvent
		)

		for t := timeAgo.Truncate(time.Minute); !t.After(time.Now()); t = t.Add(time.Minute) {
			buckets[t] = &MetricBucket{}
		}

		for _, invalidation := range invalidations.InvalidationList.Items {
			if invalidation.CreateTime.Before(timeAgo) {
				continue
			}

			invalidationDetail, err := cloudfrontClient.GetInvalidation(ctx, &cloudfront.GetInvalidationInput{
				DistributionId: distribution.Id,
				Id:             invalidation.Id,
			})
			if err != nil {
				return logger.WrapError(err)
			}

			if invalidationDetail != nil {
				bucket := invalidation.CreateTime.Truncate(time.Minute)

				buckets[bucket].Invalidations++
				buckets[bucket].Paths += float64(*invalidationDetail.Invalidation.InvalidationBatch.Paths.Quantity)

				if logConfig.Enabled() {
					message := InvalidationLogMessage{
						InvalidationID: *invalidation.Id,
						PathCount:      *invalidationDetail.Invalidation.InvalidationBatch.Paths.Quantity,
					}

					message.Paths = append(message.Paths, invalidationDetail.Invalidation.InvalidationBatch.Paths.Items...)

					logMessage, err := json.Marshal(message)
					if err != nil {
						return logger.WrapError(err)
					}

					logs = append(logs, cloudwatchlogstypes.InputLogEvent{
						Message:   aws.String(string(logMessage)),
						Timestamp: aws.Int64(invalidation.CreateTime.UnixMilli()),
					})
				}
			}
		}

		logger.SetAttr(*distribution.Id, buckets)

		for bucket, counts := range buckets {
			err = metricsClient.Add(cloudwatchtypes.MetricDatum{
				MetricName: aws.String(CloudWatchMetricInvalidationRequest),
				Unit:       cloudwatchtypes.StandardUnitCount,
				Value:      aws.Float64(counts.Invalidations),
				Timestamp:  aws.Time(bucket),
				Dimensions: []cloudwatchtypes.Dimension{
					{
						Name:  aws.String(DimensionDistribution),
						Value: aws.String(*distribution.Id),
					},
				},
			})
			if err != nil {
				return logger.WrapError(err)
			}

			err = metricsClient.Add(cloudwatchtypes.MetricDatum{
				MetricName: aws.String(CloudWatchMetricInvalidationPathCount),
				Unit:       cloudwatchtypes.StandardUnitCount,
				Value:      aws.Float64(counts.Paths),
				Timestamp:  aws.Time(bucket),
				Dimensions: []cloudwatchtypes.Dimension{
					{
						Name:  aws.String(DimensionDistribution),
						Value: aws.String(*distribution.Id),
					},
				},
			})
			if err != nil {
				return logger.WrapError(err)
			}
		}

		if len(logs) > 0 {
			_, err := logsClient.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
				LogGroupName:  aws.String(logConfig.GroupName),
				LogStreamName: aws.String(logConfig.StreamName),
			})
			if err != nil {
				var existsError *cloudwatchlogstypes.ResourceAlreadyExistsException

				if !errors.As(err, &existsError) {
					return logger.WrapError(err)
				}
			}

			sort.Slice(logs, func(i, j int) bool {
				return *logs[i].Timestamp < *logs[j].Timestamp
			})

			_, err = logsClient.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
				LogGroupName:  aws.String(logConfig.GroupName),
				LogStreamName: aws.String(logConfig.StreamName),
				LogEvents:     logs,
			})
			if err != nil {
				return logger.WrapError(err)
			}
		}

	}

	err = metricsClient.Flush()
	if err != nil {
		return logger.WrapError(err)
	}

	return nil
}
