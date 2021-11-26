package main

import (
	"context"

	"github.com/jefflinse/melatonin-ext/aws"
	"github.com/jefflinse/melatonin/json"
	"github.com/jefflinse/melatonin/mt"
)

func main() {
	myHandlerFn := func(ctx context.Context, event map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{
			"message": "Hello, World!",
		}, nil
	}

	mt.RunTests([]mt.TestCase{
		aws.Handle(myHandlerFn, "testing my handler").
			WithPayload(json.Object{}).
			ExpectStatus(200).
			ExpectPayload(json.Object{
				"message": "Hello, World!",
			}),
	})
}
