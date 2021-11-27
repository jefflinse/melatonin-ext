# melatonin-ext - Extensions for the [melatonin](https://github.com/jefflinse/melatonin#readme) test framework

These packages extend melatonin to provide additional test contexts for testing various 3rd party services.

- [AWS Lambda](#aws-lambda)

## AWS Lambda

The lambda extension provides a context for testing AWS Lambda functions. It can test Go handler functions directly as unit tests, or it can invoke deployed functions in AWS for performing E2E tests.

### Installation

    go get github.com/jefflinse/melatonin-ext/aws

### Usage

```go
package main

import (
    "github.com/jefflinse/melatonin/mt"
    "github.com/jefflinse/melatonin-ext/aws"
)

func myHandler(ctx context.Context, event interface{}) (interface{}, error) {
    return "Hello, world!", nil
}

func main() {

    mt.RunTests([]mt.TestCase{

        // Test a Go handler function directly
        aws.Handle(myHandler).
            ExpectPayload("Hello, world!"),

        // Test a Lambda function by name...
        aws.Invoke("my-lambda-function").
            ExpectStatus(200).
            ExpectPayload("Hello, world!"),

        // ...or by ARN
        aws.Invoke("arn:aws:lambda:us-west-2:123456789012:function:my-lambda-function").
            ExpectStatus(200).
            ExpectPayload("Hello, world!"),
    })
}
```

### Custom Context

Define a custom context to customize the AWS Lambda service, including the AWS session:

```go
import (
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/lambda"
)

lambdaService := lambda.New(aws.Must(aws.NewSession(
    aws.WithRegion("us-west-2"),
)))

ctx := aws.NewLambdaTextContext(lambdaService)

mt.RunTests([]mt.TestCase{

    ctx.Invoke("my-lambda-function").
        ExpectStatus(200).
        ExpectPayload("Hello, world!"),
})
```

### Use Mock Lambda APIs

A Lambda test context can be created using any type that satisfies the `LambdaAPI` interface, making it simple to substitute your own mock Lambda implementation for testing.

```go
type LambdaAPI interface {
    Invoke(input *lambda.InvokeInput) (*lambda.InvokeOutput, error)
}
```
