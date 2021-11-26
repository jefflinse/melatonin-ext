package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	lambdasvc "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/jefflinse/melatonin/expect"
	"github.com/jefflinse/melatonin/mt"
)

type LambdaAPI interface {
	Invoke(input *lambdasvc.InvokeInput) (*lambdasvc.InvokeOutput, error)
}

type LambdaTestContext struct {
	Session *session.Session

	svc LambdaAPI
}

// DefaultContext returns a LambdaTestContext using a default AWS session.
func DefaultContext() *LambdaTestContext {
	return NewLambdaTestContext(session.Must(session.NewSession()))
}

// NewLambdaFunctionContext creates a new HTTPTestContext for creating tests that target
// AWS Lambda functions using the provided AWS session.
func NewLambdaTestContext(sess *session.Session) *LambdaTestContext {
	return &LambdaTestContext{
		Session: sess,
		svc:     lambdasvc.New(sess),
	}
}

func (c *LambdaTestContext) Invoke(functionID string, description ...string) *LambdaTestCase {
	return newLambdaTestCase(c, functionID, description...)
}

func (c *LambdaTestContext) Handle(handlerFn interface{}, description ...string) *LambdaTestCase {
	return newHandlerTestCase(c, handlerFn, description...)
}

func Invoke(functionID string, description ...string) *LambdaTestCase {
	return DefaultContext().Invoke(functionID, description...)
}

func Handle(handlerFn interface{}, description ...string) *LambdaTestCase {
	return DefaultContext().Handle(handlerFn, description...)
}

type LambdaTestCase struct {
	Desc         string
	FunctionID   string
	HandlerFn    interface{}
	Payload      interface{}
	Expectations LambdaResponseExpectations

	payloadBytes []byte
	tctx         *LambdaTestContext
}

var _ mt.TestCase = &LambdaTestCase{}

func newLambdaTestCase(context *LambdaTestContext, functionID string, description ...string) *LambdaTestCase {
	return &LambdaTestCase{
		Desc:       strings.Join(description, " "),
		FunctionID: functionID,
		tctx:       context,
	}
}

func newHandlerTestCase(c *LambdaTestContext, handlerFn interface{}, description ...string) *LambdaTestCase {
	return &LambdaTestCase{
		Desc:      strings.Join(description, " "),
		HandlerFn: handlerFn,
	}
}

func (tc *LambdaTestCase) Action() string {
	if tc.FunctionID != "" {
		return "INVOKE"
	}

	return "HANDLE"
}

func (tc *LambdaTestCase) Description() string {
	if tc.Desc != "" {
		return tc.Desc
	}

	if tc.FunctionID != "" {
		return fmt.Sprint("Invoke", tc.Target())
	}

	return fmt.Sprint("Handle", tc.Target())
}

func (tc *LambdaTestCase) Target() string {
	if tc.FunctionID != "" {
		return "Lambda: " + tc.FunctionID
	}

	return "Lambda handler func: " + functionName(tc.HandlerFn)
}

func (tc *LambdaTestCase) Execute(t *testing.T) (mt.TestResult, error) {
	if tc.FunctionID != "" && tc.HandlerFn != nil {
		return nil, fmt.Errorf("LambdaTestCase must specify either FunctionID or HandlerFn, not both")
	}

	var result *LambdaTestResult
	var err error
	if tc.FunctionID != "" {
		result, err = tc.invoke()
	} else if tc.HandlerFn != nil {
		result, err = tc.handle()
	}

	if err != nil {
		return nil, err
	}

	result.validateExpectations()

	return result, nil
}

func (tc *LambdaTestCase) Describe(description string) *LambdaTestCase {
	tc.Desc = description
	return tc
}

func (tc *LambdaTestCase) WithPayload(payload interface{}) *LambdaTestCase {
	tc.Payload = payload
	return tc
}

func (tc *LambdaTestCase) ExpectStatus(status int) *LambdaTestCase {
	tc.Expectations.Status = status
	return tc
}

func (tc *LambdaTestCase) ExpectPayload(payload interface{}) *LambdaTestCase {
	tc.Expectations.Payload = payload
	return tc
}

func (tc *LambdaTestCase) ExpectExactPayload(payload interface{}) *LambdaTestCase {
	tc.Expectations.Payload = payload
	tc.Expectations.WantExactJSONPayload = true
	return tc
}

func (tc *LambdaTestCase) invoke() (*LambdaTestResult, error) {
	if tc.tctx.svc == nil {
		tc.tctx.svc = lambdasvc.New(tc.tctx.Session)
	}

	payload, err := tc.requestPayloadBytes()
	if err != nil {
		return nil, err
	}

	resp, err := tc.tctx.svc.Invoke(&lambdasvc.InvokeInput{
		FunctionName: &tc.FunctionID,
		Payload:      payload,
	})

	if err != nil {
		return nil, fmt.Errorf("invoke: %w", err)
	}

	return &LambdaTestResult{
		testCase: tc,
		Status:   int(*resp.StatusCode),
		Payload:  resp.Payload,
	}, nil
}

func (tc *LambdaTestCase) handle() (*LambdaTestResult, error) {
	handler := lambda.NewHandler(tc.HandlerFn)
	payload, err := tc.requestPayloadBytes()
	if err != nil {
		return nil, err
	}

	resp, err := handler.Invoke(context.TODO(), payload)
	if err != nil {
		return nil, err
	}

	return &LambdaTestResult{
		testCase: tc,
		Status:   200,
		Payload:  resp,
	}, nil
}

func (tc *LambdaTestCase) requestPayloadBytes() ([]byte, error) {
	if tc.payloadBytes != nil {
		return tc.payloadBytes, nil
	}

	var payload []byte
	if tc.Payload != nil {
		var err error
		switch v := tc.Payload.(type) {
		case []byte:
			payload = v
		case string:
			payload = []byte(v)
		case func() []byte:
			payload = v()
		case func() ([]byte, error):
			payload, err = v()
		default:
			payload, err = json.Marshal(tc.Payload)
		}

		if err != nil {
			return nil, fmt.Errorf("request body: %w", err)
		}
	}

	tc.payloadBytes = payload
	return tc.payloadBytes, nil
}

type LambdaResponseExpectations struct {
	Status               int
	Payload              interface{}
	WantExactJSONPayload bool
}

type LambdaTestResult struct {
	Status  int
	Payload []byte

	testCase *LambdaTestCase
	errors   []error
}

func (r *LambdaTestResult) Errors() []error {
	return r.errors
}

func (r *LambdaTestResult) TestCase() mt.TestCase {
	return r.testCase
}

func functionName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func parseResponsePayload(body []byte) interface{} {
	if len(body) > 0 {
		var bodyMap map[string]interface{}
		if err := json.Unmarshal(body, &bodyMap); err == nil {
			return bodyMap
		}

		var bodyArray []interface{}
		if err := json.Unmarshal(body, &bodyArray); err == nil {
			return bodyArray
		}

		b := strings.TrimPrefix(string(body), `"`)
		return strings.TrimSuffix(b, `"`)
	}

	return nil
}

func (r *LambdaTestResult) validateExpectations() {
	tc := r.TestCase().(*LambdaTestCase)
	if tc.Expectations.Status != 0 {
		if r.Status != tc.Expectations.Status {
			r.errors = append(r.errors, fmt.Errorf("expected status code %d, got %d", tc.Expectations.Status, r.Status))
		}
	}

	if tc.Expectations.Payload != nil {
		body := parseResponsePayload(r.Payload)
		if errs := expect.Value("body", tc.Expectations.Payload, body, tc.Expectations.WantExactJSONPayload); len(errs) > 0 {
			r.errors = append(r.errors, errs...)
		}
	}
}

var _ mt.TestResult = &LambdaTestResult{}
