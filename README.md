# melatonin-ext - Extensions for the [melatonin](https://github.com/jefflinse/melatonin#readme) test framework

These packages extend melatonin to provide additional test contexts for testing various 3rd party services.

- [Exec](#exec)
- [AWS Lambda](#aws-lambda)

## Exec

The Exec extension provides a context for testing a command line application. It can be used to execute commands
and check the exit code, stdout, and stderr.

### Installation

    go get github.com/jefflinse/melatonin-ext/exec

### Usage

```go
package main

import (
    osexec "os/exec"
    "github.com/jefflinse/melatonin/mt"
    "github.com/jefflinse/melatonin-ext/aws/exec"
)

func main() {

    mt.RunTests([]mt.TestCase{

        // Test a commmand
        exec.Run("echo").
            WithArgs("Hello, world!").
            ExpectExitCode(0).
            ExpectStdout("Hello, world!"),

        // Supply some env vars
        exec.Run("echo").
            WithArgs("Hello, world!").
            WithEnv(map[string]string{
                "FOO": "baz",
                "BAR": "buz",
            }).
            ExpectStatus(200).
            ExpectPayload("Hello, world!"),

        // Supply your own exec.Cmd to run
        exec.Cmd(osexec.Command("echo", "Hello, world!")).
            ExpectStatus(200).
            ExpectPayload("Hello, world!"),
    })
}
```

### Custom Context

Define a custom context to customize the execution context, such as the environment:

```go
ctx := exec.NewTestContext().
    WithInheritedEnvironment(true).
    WithEnvVars(map[string]string{
        "FIRST":  "foo",
        "SECOND": "bar",
    })

mt.RunTests([]mt.TestCase{

    ctx.Run("echo").
        ExpectStatus(200).
        ExpectPayload("Hello, world!"),
        WithEnvVars(map[string]string{
            "SECOND": "new bar",
            "THIRD": "baz",
        }),
})

// environment for test case:
//  <all inherited env vars from runtime>
//  FIRST=foo
//  SECOND=new bar
//  THIRD=baz
```

`WithInheritedEnvironment(true)` instructs the context to inherit the environment variables from the environment that launched the melatonin process. The default is `false`, meaning that exec test contexts (including the default context) begin with an empty environment by default.

`WithEnvVars(map[string]string{})` will overwrite/append environment variables for a context or test case.

## AWS Lambda

The Lambda extension provides a context for testing AWS Lambda functions. It can test Go handler functions directly as unit tests, or it can invoke deployed functions in AWS for performing E2E tests.

### Installation

    go get github.com/jefflinse/melatonin-ext/aws

### Usage

```go
package main

import (
    "github.com/jefflinse/melatonin/mt"
    "github.com/jefflinse/melatonin-ext/aws/lambda"
)

func myHandler(ctx context.Context, event interface{}) (interface{}, error) {
    return "Hello, world!", nil
}

func main() {

    mt.RunTests([]mt.TestCase{

        // Test a Go handler function directly
        lambda.Handle(myHandler).
            ExpectPayload("Hello, world!"),

        // Test a Lambda function by name...
        lambda.Invoke("my-lambda-function").
            ExpectStatus(200).
            ExpectPayload("Hello, world!"),

        // ...or by ARN
        lambda.Invoke("arn:aws:lambda:us-west-2:123456789012:function:my-lambda-function").
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
    lambdasvc "github.com/aws/aws-sdk-go/service/lambda"
)

lambdaService := lambdasvc.New(aws.Must(aws.NewSession(
    aws.WithRegion("us-west-2"),
)))

ctx := lambda.NewTestContext(lambdaService)

mt.RunTests([]mt.TestCase{

    ctx.Invoke("my-lambda-function").
        ExpectStatus(200).
        ExpectPayload("Hello, world!"),
})
```

### Use Mock Lambda APIs

A Lambda test context can be created using any type that satisfies the `LambdaAPI` interface, making it simple to substitute your own mock Lambda implementation for testing.

```go
import "github.com/aws/aws-sdk-go/service/lambda"

type LambdaAPI interface {
    Invoke(input *lambda.InvokeInput) (*lambda.InvokeOutput, error)
}
```
