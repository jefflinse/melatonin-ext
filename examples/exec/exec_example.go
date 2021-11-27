package main

import (
	"github.com/jefflinse/melatonin-ext/exec"
	"github.com/jefflinse/melatonin/mt"
)

func main() {
	runner := mt.NewTestRunner().WithContinueOnFailure(true)
	runner.RunTests([]mt.TestCase{

		exec.Exec("echo", "test a local command").
			WithArgs("Hello, World!").
			ExpectExitCode(0).
			ExpectStdout("Hello, World!\n").
			ExpectStderr(""),

		exec.Exec("echo").
			WithArgs("Hello, World!").
			ExpectExitCode(0),

		exec.Exec("/bin/notfound", "attempt to execute something nonexistent"),
	})
}
