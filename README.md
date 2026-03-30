Cloudfront Invalidation Metrics
---

CloudFront Invalidation Metrics offer users with access to CloudWatch
dashboards some interesting new graphs which show the following
custom metrics:

* Invalidation count per CloudFront distribution ID
* Paths selectively invalidated per CloudFront distribution ID.

## How to

The Lambda can be run locally as a Go binary without the Lambda variables
`_LAMBDA_SERVER_PORT` or `AWS_LAMBDA_RUNTIME_API` being set like normal:
```shell
go run main.go
```

It will however need to authenticate to AWS in the standard way, so
set the following variables as needed to authenticate.

|                    | Variable                | Explaination                                                                  |
|--------------------|-------------------------|-------------------------------------------------------------------------------|
| Configuration file | `AWS_CONFIG_FILE`       | Configuration file for AWS to use<br />Defaults to `${HOME}/.aws/credentials. |
| Profile            | `AWS_PROFILE`           | Name of the profile to use in your configuration file.                        |
| Region             | `AWS_REGION`            | The regional identifier - such as `ap-southeast-2`.                           |
| Access Key ID      | `AWS_ACCESS_KEY_ID`     | The access key ID from your IAM credentials.                                  |
| Secret Access Key  | `AWS_SECRET_ACCESS_KEY` | The secret access key from your IAM credentials.                              |

### Examples

1. Providing credentials to the app:
    ```shell
    AWS_ACCESS_KEY_ID=x AWS_SECRET_ACCESS_KEY=y go run main.go
    ```
2. Providing credentials via profile to the app:
    ```shell
    AWS_PROFILE=z go run main.go
    ```

## Licence

This project is licenced under GPLv3
