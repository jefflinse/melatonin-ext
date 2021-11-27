package exec

import (
	"fmt"
	"io"
	"io/fs"
	osexec "os/exec"
	"strings"
	"testing"

	"github.com/jefflinse/melatonin/mt"
)

type TestContext struct {
	Environment map[string]string
}

func DefaultContext() *TestContext {
	return NewTestContext()
}

func NewTestContext() *TestContext {
	return &TestContext{
		Environment: map[string]string{},
	}
}

func (c *TestContext) WithEnvironment(env map[string]string) *TestContext {
	for k, v := range env {
		c.Environment[k] = v
	}

	return c
}

type TestCase struct {
	Desc         string
	Expectations Expectations

	command *osexec.Cmd
	tctx    *TestContext
}

var _ mt.TestCase = &TestCase{}

func (tc *TestCase) Action() string {
	return "EXEC"
}

func (tc *TestCase) Description() string {
	if tc.Desc != "" {
		return tc.Desc
	}

	return tc.command.String()
}

func (tc *TestCase) Execute(t *testing.T) (mt.TestResult, error) {
	result := &TestResult{
		testCase: tc,
	}

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	tc.command.Stdout = stdout
	tc.command.Stderr = stderr

	if err := tc.command.Run(); err != nil {
		switch e := err.(type) {
		case *fs.PathError:
			result.errors = append(result.errors, fmt.Errorf("%s: %s", e.Path, e.Err))
		default:
			result.errors = append(result.errors, err)
		}
	}

	result.ExitCode = tc.command.ProcessState.ExitCode()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.validateExpectations()

	return result, nil
}

func (tc *TestCase) Target() string {
	return strings.Join(tc.command.Args, " ")
}

func (c *TestContext) newTestCase(command string, description ...string) *TestCase {
	return &TestCase{
		Desc:         strings.Join(description, ", "),
		Expectations: Expectations{},

		command: osexec.Command(command),
		tctx:    c,
	}
}

func Run(command string, description ...string) *TestCase {
	return DefaultContext().Run(command, description...)
}

func (tc *TestContext) Run(command string, description ...string) *TestCase {
	return tc.newTestCase(command, description...)
}

func (tc *TestCase) WithArgs(args ...string) *TestCase {
	tc.command.Args = append(tc.command.Args, args...)
	return tc
}

func (tc *TestCase) WithEnv(env map[string]string) *TestCase {
	for k, v := range env {
		tc.command.Env = append(tc.command.Env, k+"="+v)
	}

	return tc
}

func (tc *TestCase) WithStdin(stdin io.Reader) *TestCase {
	tc.command.Stdin = stdin
	return tc
}

func (tc *TestCase) WithWorkingDir(dir string) *TestCase {
	tc.command.Dir = dir
	return tc
}

func (tc *TestCase) ExpectExitCode(code int) *TestCase {
	tc.Expectations.ExitCode = &code
	return tc
}

func (tc *TestCase) ExpectStdout(stdout string) *TestCase {
	tc.Expectations.Stdout = &stdout
	return tc
}

func (tc *TestCase) ExpectStderr(stderr string) *TestCase {
	tc.Expectations.Stderr = &stderr
	return tc
}

type Expectations struct {
	ExitCode *int
	Stdout   *string
	Stderr   *string
}

type TestResult struct {
	ExitCode int
	Stdout   string
	Stderr   string

	errors   []error
	testCase *TestCase
}

var _ mt.TestResult = &TestResult{}

func (r *TestResult) Errors() []error {
	return r.errors
}

func (r *TestResult) TestCase() mt.TestCase {
	return r.testCase
}

func (r *TestResult) validateExpectations() {
	tc := r.TestCase().(*TestCase)

	if tc.Expectations.ExitCode != nil && r.ExitCode != *tc.Expectations.ExitCode {
		r.errors = append(r.errors, fmt.Errorf("expected exit code %d, got %d", *tc.Expectations.ExitCode, r.ExitCode))
	}

	if tc.Expectations.Stdout != nil && r.Stdout != *tc.Expectations.Stdout {
		r.errors = append(r.errors, fmt.Errorf("expected stdout %q, got %q", *tc.Expectations.Stdout, r.Stdout))
	}

	if tc.Expectations.Stderr != nil && r.Stderr != *tc.Expectations.Stderr {
		r.errors = append(r.errors, fmt.Errorf("expected stderr %q, got %q", *tc.Expectations.Stderr, r.Stderr))
	}
}
