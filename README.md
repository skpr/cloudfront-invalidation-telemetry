Cloudfront Invalidation Telemetry
---

A Lambda for extracting invalidation metrics and logs from a CloudFront distribution.

## What this provides?

**Metrics**

* Invalidation count per CloudFront distribution ID
* Paths selectively invalidated per CloudFront distribution ID.

**Logs**

Logging if configured by adding the following tags to the CloudFront distribution.

| Name                                                    | Description                                                                      |
|---------------------------------------------------------|----------------------------------------------------------------------------------|
| `skpr.io/cloudfront-invalidation-telemetry/logs/group`  | The CloudWatch Logs group which will receive logged invalidation requests        |
| `skpr.io/cloudfront-invalidation-telemetry/logs/stream` | The CloudWatch Logs group stream which will receive logged invalidation requests |
