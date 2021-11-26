package main

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/jefflinse/melatonin-ext/aws"
	"github.com/jefflinse/melatonin/json"
	"github.com/jefflinse/melatonin/mt"
)

func main() {
	sess := aws.NewLambdaTestContext(session.Must(session.NewSession()))

	mt.RunTests([]mt.TestCase{
		aws.Handle(sampleHandler, "testing my handler").
			WithPayload(json.Object{}).
			ExpectStatus(200).
			ExpectPayload(json.Object{
				"message": "Hello, World!",
			}),

		sess.Invoke("testFunction", "test a lambda by specifying a function name").
			WithPayload(json.Object{
				"name": "Bob",
			}).
			ExpectStatus(200).
			ExpectPayload("Hello Bob!"),

		sess.Invoke("arn:aws:lambda:us-west-2:933760355198:function:testFunction", "test a lambda by specifying an ARN").
			WithPayload(json.Object{
				"name": "Bob",
			}).
			ExpectStatus(200).
			ExpectPayload("Hello Bob!"),

		aws.Invoke("testFunction", "test a lambda using the default context"),
	})
}

func sampleHandler(ctx context.Context, event map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"message": "Hello, World!",
	}, nil
}
