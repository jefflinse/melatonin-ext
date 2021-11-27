package exec

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	osexec "os/exec"
	"strings"
	"testing"

	"github.com/jefflinse/melatonin/mt"
)

type TestContext struct {
	Environment []string
}

func DefaultContext() *TestContext {
	return NewTestContext()
}

func NewTestContext() *TestContext {
	return &TestContext{
		Environment: []string{},
	}
}

func (c *TestContext) WithEnvVars(env map[string]string) *TestContext {
	for k, v := range env {
		c.Environment = append(c.Environment, k+"="+v)
	}

	return c
}

func (c *TestContext) WithInheritedEnvironment(inherit bool) *TestContext {
	if inherit {
		c.Environment = os.Environ()
	} else {
		c.Environment = []string{}
	}

	return c
}

type TestCase struct {
	Desc         string
	Expectations Expectations

	cmd  *osexec.Cmd
	tctx *TestContext
}

var _ mt.TestCase = &TestCase{}

func (tc *TestCase) Action() string {
	return "EXEC"
}

func (tc *TestCase) Description() string {
	if tc.Desc != "" {
		return tc.Desc
	}

	return tc.cmd.String()
}

func (tc *TestCase) Execute(t *testing.T) (mt.TestResult, error) {
	result := &TestResult{
		testCase: tc,
	}

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	tc.cmd.Stdout = stdout
	tc.cmd.Stderr = stderr

	if err := tc.cmd.Run(); err != nil {
		switch e := err.(type) {
		case *fs.PathError:
			result.errors = append(result.errors, fmt.Errorf("%s: %s", e.Path, e.Err))
		default:
			result.errors = append(result.errors, err)
		}
	}

	result.ExitCode = tc.cmd.ProcessState.ExitCode()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.validateExpectations()

	return result, nil
}

func (tc *TestCase) Target() string {
	return strings.Join(tc.cmd.Args, " ")
}

func (c *TestContext) newTestCase(cmd *osexec.Cmd, description ...string) *TestCase {
	cmd.Env = append(c.Environment, cmd.Env...)
	return &TestCase{
		Desc:         strings.Join(description, ", "),
		Expectations: Expectations{},

		cmd:  cmd,
		tctx: c,
	}
}

func Run(command string, description ...string) *TestCase {
	return DefaultContext().Run(command, description...)
}

func Cmd(command *osexec.Cmd, description ...string) *TestCase {
	return DefaultContext().Cmd(command, description...)
}

func (tc *TestContext) Run(command string, description ...string) *TestCase {
	t := tc.newTestCase(osexec.Command(command), description...)
	return t
}

func (tc *TestContext) Cmd(cmd *osexec.Cmd, description ...string) *TestCase {
	t := tc.newTestCase(cmd, description...)
	return t
}

func (tc *TestCase) Describe(description string) *TestCase {
	tc.Desc = description
	return tc
}

func (tc *TestCase) WithArgs(args ...string) *TestCase {
	tc.cmd.Args = append(tc.cmd.Args, args...)
	return tc
}

func (tc *TestCase) WithEnvVars(env map[string]string) *TestCase {
	for k, v := range env {
		tc.cmd.Env = append(tc.cmd.Env, k+"="+v)
	}

	return tc
}

func (tc *TestCase) WithStdin(stdin io.Reader) *TestCase {
	tc.cmd.Stdin = stdin
	return tc
}

func (tc *TestCase) WithWorkingDir(dir string) *TestCase {
	tc.cmd.Dir = dir
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
