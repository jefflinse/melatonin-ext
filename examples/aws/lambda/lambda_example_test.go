package main_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	lambdasvc "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/jefflinse/melatonin-ext/aws/lambda"
	"github.com/jefflinse/melatonin/json"
	"github.com/jefflinse/melatonin/mt"
)

func TestLambda(t *testing.T) {
	svc := lambdasvc.New(session.Must(session.NewSession()))
	lda := lambda.NewTestContext(svc)

	runner := mt.NewTestRunner().WithContinueOnFailure(true)
	runner.RunTestsT(t, []mt.TestCase{
		lambda.Handle(sampleHandler, "testing my handler").
			WithPayload(json.Object{}).
			ExpectStatus(200).
			ExpectPayload(json.Object{
				"message": "Hello, World!",
			}),

		lambda.Handle(sampleHandler).
			WithPayload(json.Object{}).
			ExpectStatus(200).
			ExpectPayload(json.Object{
				"message": "Hello, World!",
			}),

		lda.Invoke("testFunction", "test a lambda by specifying a function name").
			WithPayload(json.Object{
				"name": "Bob",
			}).
			ExpectStatus(200).
			ExpectPayload("Hello Bob!").
			ExpectVersion(lambda.VersionLatest),

		lda.Invoke("testFunction", "test a lambda expecting a specific version").
			WithPayload(json.Object{
				"name": "Bob",
			}).
			ExpectStatus(200).
			ExpectPayload("Hello Bob!").
			ExpectVersion("2"),

		lda.Invoke("testFunction").
			WithPayload(json.Object{
				"name": "Bob",
			}).
			ExpectPayload("Hello Bob!"),

		lda.Invoke("arn:aws:lambda:us-west-2:933760355198:function:testFunction", "test a lambda by specifying an ARN").
			WithPayload(json.Object{
				"name": "Bob",
			}).
			ExpectStatus(200).
			ExpectPayload("Hello Bob!"),

		lda.Invoke("arn:aws:lambda:us-west-2:933760355198:function:testFunction").
			WithPayload(json.Object{
				"name": "Bob",
			}).
			ExpectPayload("Hello Bob!"),

		lda.Invoke("testFunction", "test a lambda by performing a dry run").
			AsDryRun(),

		lda.Invoke("testFailingFunction", "test a function that returns an expected error").
			WithPayload(json.Object{
				"name": "Bob",
			}).
			ExpectStatus(200).
			ExpectFunctionError("my function error"),

		lda.Invoke("doesNotExist", "attempt to test a function that doesn't exist").
			WithPayload(json.Object{}),

		lda.Invoke("doesNotExist", "attempt to test a function that doesn't exist").
			WithPayload(json.Object{}).ExpectStatus(200),

		lda.Invoke("doesNotExist", "attempt to test a function that doesn't exist").
			WithPayload(json.Object{}).ExpectStatus(404),

		lambda.Invoke("testFunction", "test a lambda using the default context"),
	})
}

func sampleHandler(ctx context.Context, event map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"message": "Hello, World!",
	}, nil
}
