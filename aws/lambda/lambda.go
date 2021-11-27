package lambda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	lambdasvc "github.com/aws/aws-sdk-go/service/lambda"
	"github.com/jefflinse/melatonin/expect"
	"github.com/jefflinse/melatonin/mt"
)

const (
	VersionLatest = "$LATEST"
)

type LambdaAPI interface {
	Invoke(input *lambdasvc.InvokeInput) (*lambdasvc.InvokeOutput, error)
}

type TestContext struct {
	svc LambdaAPI
}

// DefaultContext returns a LambdaTestContext using a default AWS Lambda service.
func DefaultContext() *TestContext {
	svc := lambdasvc.New(session.Must(session.NewSession()))
	return NewTestContext(svc)
}

// EmptyContext returns a LambdaTestContext with no configured AWS service.
// It is suitable only for testing local function handlers.
func EmptyContext() *TestContext {
	return &TestContext{}
}

// NewLambdaFunctionContext creates a new HTTPTestContext for creating tests that target
// AWS Lambda functions using the provided AWS service.
func NewTestContext(svc LambdaAPI) *TestContext {
	return &TestContext{
		svc: svc,
	}
}

func (c *TestContext) Invoke(functionID string, description ...string) *TestCase {
	return newLambdaTestCase(c, functionID, description...)
}

func (c *TestContext) Handle(handlerFn interface{}, description ...string) *TestCase {
	return newHandlerTestCase(c, handlerFn, description...)
}

func Invoke(functionID string, description ...string) *TestCase {
	return DefaultContext().Invoke(functionID, description...)
}

func Handle(handlerFn interface{}, description ...string) *TestCase {
	return EmptyContext().Handle(handlerFn, description...)
}

type TestCase struct {
	Desc         string
	FunctionID   string
	HandlerFn    interface{}
	Payload      interface{}
	Expectations ResponseExpectations

	payloadBytes []byte
	tctx         *TestContext
}

var _ mt.TestCase = &TestCase{}

func newLambdaTestCase(context *TestContext, functionID string, description ...string) *TestCase {
	return &TestCase{
		Desc:       strings.Join(description, " "),
		FunctionID: functionID,
		tctx:       context,
	}
}

func newHandlerTestCase(c *TestContext, handlerFn interface{}, description ...string) *TestCase {
	return &TestCase{
		Desc:      strings.Join(description, " "),
		HandlerFn: handlerFn,
	}
}

func (tc *TestCase) Action() string {
	if tc.FunctionID != "" {
		return "INVOKE"
	}

	return "HANDLE"
}

func (tc *TestCase) Description() string {
	if tc.Desc != "" {
		return tc.Desc
	}

	if tc.FunctionID != "" {
		return "Invoke " + tc.Target()
	}

	return "Call " + tc.Target()
}

func (tc *TestCase) Target() string {
	if tc.FunctionID != "" {
		target := strings.Replace(tc.FunctionID, "arn:aws:lambda:", ":::", 1)
		target = strings.Replace(target, ":function:", "::", 1)
		return fmt.Sprintf("AWS Lambda (%s)", target)
	}

	return "AWS Lambda handler (" + functionName(tc.HandlerFn) + ")"
}

func (tc *TestCase) Execute(t *testing.T) (mt.TestResult, error) {
	if tc.FunctionID != "" && tc.HandlerFn != nil {
		return nil, fmt.Errorf("LambdaTestCase must specify either FunctionID or HandlerFn, not both")
	}

	var result *TestResult
	var err error
	if tc.FunctionID != "" {
		result, err = tc.invoke()
	} else if tc.HandlerFn != nil {
		result, err = tc.handle()
	}

	if err != nil {
		return nil, err
	}

	if len(result.Errors()) == 0 {
		result.validateExpectations()
	}

	return result, nil
}

func (tc *TestCase) Describe(description string) *TestCase {
	tc.Desc = description
	return tc
}

func (tc *TestCase) WithPayload(payload interface{}) *TestCase {
	tc.Payload = payload
	return tc
}

func (tc *TestCase) ExpectFunctionError(err string) *TestCase {
	tc.Expectations.FunctionError = err
	return tc
}

func (tc *TestCase) ExpectPayload(payload interface{}) *TestCase {
	tc.Expectations.Payload = payload
	return tc
}

func (tc *TestCase) ExpectExactPayload(payload interface{}) *TestCase {
	tc.Expectations.Payload = payload
	tc.Expectations.WantExactJSONPayload = true
	return tc
}

func (tc *TestCase) ExpectStatus(status int) *TestCase {
	tc.Expectations.Status = status
	return tc
}

func (tc *TestCase) ExpectVersion(version string) *TestCase {
	tc.Expectations.Version = version
	return tc
}

func (tc *TestCase) invoke() (*TestResult, error) {
	if tc.tctx.svc == nil {
		return nil, errors.New("no AWS Lambda service provided")
	}

	payload, err := tc.requestPayloadBytes()
	if err != nil {
		return nil, err
	}

	resp, err := tc.tctx.svc.Invoke(&lambdasvc.InvokeInput{
		FunctionName: &tc.FunctionID,
		Payload:      payload,
	})

	result := &TestResult{
		testCase:        tc,
		InvocationError: err,
		Payload:         resp.Payload,
	}

	if resp.StatusCode != nil {
		result.Status = int(*resp.StatusCode)
	}

	if resp.FunctionError != nil {
		result.FunctionError = *resp.FunctionError
	}

	if resp.ExecutedVersion != nil {
		result.Version = *resp.ExecutedVersion
	}

	return result, nil
}

func (tc *TestCase) handle() (*TestResult, error) {
	handler := lambda.NewHandler(tc.HandlerFn)
	payload, err := tc.requestPayloadBytes()
	if err != nil {
		return nil, err
	}

	resp, err := handler.Invoke(context.TODO(), payload)
	if err != nil {
		return nil, err
	}

	return &TestResult{
		testCase: tc,
		Status:   200,
		Payload:  resp,
	}, nil
}

func (tc *TestCase) requestPayloadBytes() ([]byte, error) {
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

type ResponseExpectations struct {
	FunctionError        string
	Payload              interface{}
	Status               int
	Version              string
	WantExactJSONPayload bool
}

type TestResult struct {
	FunctionError   string
	InvocationError error
	Payload         []byte
	Status          int
	Version         string

	testCase *TestCase
	errors   []error
}

func (r *TestResult) Errors() []error {
	return r.errors
}

func (r *TestResult) TestCase() mt.TestCase {
	return r.testCase
}

func functionName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func parseResponsePayload(body []byte) interface{} {
	if len(body) > 0 {
		// see if it's a function error response
		d := json.NewDecoder(bytes.NewReader(body))
		d.DisallowUnknownFields()
		funcErrResp := &functionError{}
		if err := d.Decode(funcErrResp); err == nil && funcErrResp.Message != nil && funcErrResp.Type != nil {
			return funcErrResp
		}

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

type functionError struct {
	Message *string `json:"errorMessage"`
	Type    *string `json:"errorType"`
}

func (r *TestResult) validateExpectations() {
	tc := r.TestCase().(*TestCase)

	// Check for an unexpected invocation error.
	if r.InvocationError != nil && !(tc.Expectations.Status != 0 && r.Status == tc.Expectations.Status) {
		awsErr := r.InvocationError.(awserr.Error)
		r.errors = append(r.errors, errors.New(awsErr.Message()))
	}

	// If the wrong version was executed, that should probably be the next error,
	// since it will affect the interpretation of the other errors.
	if tc.Expectations.Version != "" && r.Version != tc.Expectations.Version {
		r.errors = append(r.errors, fmt.Errorf("expected to execute version %q, but executed version %q", tc.Expectations.Version, r.Version))
	}

	if tc.Expectations.Status != 0 && r.Status != tc.Expectations.Status {
		r.errors = append(r.errors, fmt.Errorf("expected status %d, got %d", tc.Expectations.Status, r.Status))
	}

	if tc.Expectations.FunctionError != "" && r.FunctionError == "Unhandled" {
		errPayload := parseResponsePayload(r.Payload).(*functionError)
		if *errPayload.Message != tc.Expectations.FunctionError {
			r.errors = append(r.errors, fmt.Errorf("expected function error %q, got %q", tc.Expectations.FunctionError, *errPayload.Message))
		}
	} else {
		if r.FunctionError != "" {
			r.errors = append(r.errors, fmt.Errorf("expected no function error, got %q", r.FunctionError))
		}
	}

	if tc.Expectations.Payload != nil {
		body := parseResponsePayload(r.Payload)
		if errs := expect.Value("body", tc.Expectations.Payload, body, tc.Expectations.WantExactJSONPayload); len(errs) > 0 {
			r.errors = append(r.errors, errs...)
		}
	}
}

var _ mt.TestResult = &TestResult{}
