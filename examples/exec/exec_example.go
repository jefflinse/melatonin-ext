package main

import (
	"github.com/jefflinse/melatonin-ext/exec"
	"github.com/jefflinse/melatonin/mt"
)

func main() {
	runner := mt.NewTestRunner().WithContinueOnFailure(true)
	runner.RunTests([]mt.TestCase{

		exec.Run("echo", "test a local command").
			WithArgs("Hello, World!").
			ExpectExitCode(0).
			ExpectStdout("Hello, World!\n").
			ExpectStderr(""),

		exec.Run("echo").
			WithArgs("Hello, World!").
			ExpectExitCode(0),

		exec.Run("/bin/notfound", "attempt to execute something nonexistent"),
	})
}
