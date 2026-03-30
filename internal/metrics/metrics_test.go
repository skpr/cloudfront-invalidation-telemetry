package metrics

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"

	client "github.com/skpr/cloudfront-invalidation-telemetry/internal/aws/cloudwatch"
)

func TestAdd(t *testing.T) {
	client, err := New(&client.MockClient{}, "dev/null")
	assert.NoError(t, err)

	err = client.Add(types.MetricDatum{
		MetricName: aws.String("TestResponse"),
		Value:      aws.Float64(1),
	})
	assert.NoError(t, err)

	// Ensure the client has stored the record.
	assert.Equal(t, len(client.Data), 1)
}

func TestFlush(t *testing.T) {
	cw := &client.MockClient{}

	client, err := New(cw, "dev/null")
	assert.NoError(t, err)

	// Generate 21 data points.
	//   * The flush should be triggered after 20
	//   * There should be 1 left over in the client data.
	for i := 0; i < 21; i++ {
		err = client.Add(types.MetricDatum{
			MetricName: aws.String("TestResponse"),
			Value:      aws.Float64(1),
		})
		assert.NoError(t, err)
	}

	// Test that the CloudWatch client received the data points.
	assert.Equal(t, 20, len(cw.MetricData))

	// Ensure the records were flushed and the remaining records are kept.
	assert.Equal(t, 1, len(client.Data))
}
