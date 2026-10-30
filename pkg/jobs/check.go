package jobs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Checks is a list of Check pointers
type Checks []*Check

// Run is executed to run all Checks in a Checks resource
func (cs *Checks) Run(conns Connections) {
	for _, check := range *cs {
		for _, args := range check.Matrix.Instances() {
			if err := check.Run(conns, args); err != nil {
				log.Fatalf("Check [%s] failed: %e", check.String(), err)
			}
		}
	}
}

// Clone will clone a Checks list
func (cs Checks) Clone() (clone Checks) {
	for _, c := range cs {
		clone = append(clone, c.Clone())
	}
	return clone
}

// Check is a shell script or query that verifies the result of a job.
type Check struct {
	// Home (~) is not resolved
	File       string     `yaml:"file,omitempty"`
	Name       string     `yaml:"name"`
	Role       string     `yaml:"role"`
	Type       string     `yaml:"type"`
	Inline     string     `yaml:"inline,omitempty"`
	BatchMode  bool       `yaml:"batchMode"`
	Rc         int        `yaml:"rc"`
	Expected   string     `yaml:"expected,omitempty"`
	Unexpected string     `yaml:"unexpected,omitempty"`
	Matrix     MatrixArgs `yaml:"matrix,omitempty"`
	tmpFile    string
}

// Clone returns a copy of the check.
func (c Check) Clone() *Check {
	return &Check{
		Name:       c.Name,
		Role:       c.Role,
		Type:       c.Type,
		Inline:     c.Inline,
		File:       c.File,
		Matrix:     c.Matrix,
		BatchMode:  c.BatchMode,
		Expected:   c.Expected,
		Unexpected: c.Unexpected,
	}
}

func (c Check) String() string {
	var chk string
	if c.Inline != "" {
		chk = fmt.Sprintf("inline='%s'", strings.Replace(
			strings.Replace(c.Inline, "\n", "\\n", -1), "'", "''", -1))
	} else {
		chk = fmt.Sprintf("file=%s", c.File)
	}
	return fmt.Sprintf("name='%s', type=%s, %s", strings.Replace(c.Name, "'", "''", -1), c.Type, chk)
}

// VerifyScriptFile checks that the script file of a shell check exists and is executable.
func (c Check) VerifyScriptFile() (err error) {
	if c.Inline != "" {
		return nil
	}
	if c.Type != "shell" {
		return nil
	}
	return verifyScriptFile(c.File)
}

// Verify checks the check against the defined connections and returns the problems found.
func (c Check) Verify(stepName string, conns Connections) (errs []error) {
	switch c.Type {
	case "":
		// This is fine when only one connection is defined. We use that one.
		if len(conns) != 1 {
			errs = append(errs, fmt.Errorf(
				"please reference a specific Type for step check %s.%s, or just define only one Connection",
				stepName, c.Name))
		}
	case "shell":
		// Special type shell for running shell checks instead of db connection
	default:
		if _, exists := conns[c.Type]; !exists {
			errs = append(errs, fmt.Errorf("step check %s.%s references an unknown Type %s", stepName,
				c.Type, c.Name))
		}
	}
	if err := c.VerifyScriptFile(); err != nil {
		errs = append(errs, err)
	}
	return errs
}

// ScriptFile returns a path to the script holding the check.
// This could be the symlink evaluated version of Check.File.
// Or this could be an executable temporary file with Check.Inline as contents.
// This is wat is used to run shell checks
func (c *Check) ScriptFile() (scriptFile string) {
	var err error
	var tmpFile *os.File
	if c.Inline != "" {
		if tmpFile, err = os.CreateTemp("", "pgQuartsInlineCheck"); err != nil {
			log.Panicf("error creating tempfile: %e", err)
		}
		c.tmpFile = tmpFile.Name()
		if _, err = tmpFile.WriteString(c.Inline); err != nil {
			log.Panicf("error writing inline check to tempfile: %e", err)
		} else if err = tmpFile.Close(); err != nil {
			log.Panicf("error closing the tmpfile: %e", err)
			// os.Chmod should also work on Windows
		} else if err = os.Chmod(tmpFile.Name(), tmpFileMode); err != nil {
			log.Panicf("error making inline tempfile script executable: %e", err)
		}
		return c.tmpFile
	}
	if err = c.VerifyScriptFile(); err != nil {
		log.Panicf("Cannot run script %s", c.File)
	}
	if scriptFile, err = filepath.EvalSymlinks(c.File); err != nil {
		log.Panicf("error while evaluating SymLinks: %e", err)
	}
	return scriptFile
}

// ScriptBody does the exact opposite of ScriptFile.
// For Check.File it reads the contents.
// In other situations it just returns Check.Inline.
// This is wat is used to run checks on database connections.
func (c Check) ScriptBody() (string, error) {
	if c.Inline != "" {
		return c.Inline, nil
	}
	scriptBodyBytes, err := os.ReadFile(c.File)
	if err != nil {
		return "", nil
	}
	return string(scriptBodyBytes), nil
}

// Run runs the check and returns an error if the output or return code is not as expected.
func (c *Check) Run(conns Connections, args InstanceArguments) error {
	log.Infof("Running check: %s, with arguments %s", c.String(), args.String())
	if c.Type == "" || c.Type == "shell" {
		return c.RunOsCheck(args)
	}
	if body, err := c.ScriptBody(); err != nil {
		return err
	} else if stdOut, err := conns.Execute(c.Type, c.Role, body, c.BatchMode, args); err != nil && c.Rc == 0 {
		return fmt.Errorf("%s unexpectedly generated an error: %e", c.String(), err)
	} else if err == nil && c.Rc != 0 {
		return fmt.Errorf("%s unexpectedly ran without error", c.String())
	} else if expErr := CheckOutput(stdOut, c.Expected, c.Unexpected); expErr != nil {
		return fmt.Errorf("%s in stdout", expErr.Error())
	}
	return nil
}

// CleanTempFile removes the temporary file created for an inline script.
func (c *Check) CleanTempFile() {
	if c.tmpFile != "" {
		log.Debugf("removing tmp file %s", c.tmpFile)
		if err := os.Remove(c.tmpFile); err != nil {
			log.Errorf("error removing file %s: %e", c.tmpFile, err)
		}
		c.tmpFile = ""
	}
}

// CheckOutput verifies that the output contains expected and does not contain unexpected.
// Empty strings are not checked.
func CheckOutput(stdOut Result, expected string, unexpected string) error {
	stdout := NewResultFromString(stdOut.String())
	if expected != "" && !stdout.Contains(expected) {
		return fmt.Errorf("expected string (%s) not found", expected)
	} else if unexpected != "" && stdout.Contains(unexpected) {
		return fmt.Errorf("unexpected string (%s) found", unexpected)
	}
	return nil
}

// RunOsCheck runs the check as a bash script and verifies its return code and output.
func (c *Check) RunOsCheck(args InstanceArguments) (err error) {
	exCheck := exec.Command("/bin/bash", c.ScriptFile()) // #nosec
	exCheck.Env = args.AsEnv()
	var stdOut, stdErr bytes.Buffer
	exCheck.Stdout = io.MultiWriter(&stdOut)
	exCheck.Stderr = io.MultiWriter(&stdErr)
	defer c.CleanTempFile()
	if err = exCheck.Run(); err != nil {
		switch typedErr := err.(type) {
		case *exec.ExitError:
			rc := typedErr.ExitCode()
			if rc != c.Rc {
				return fmt.Errorf("unexpected returncode (expected=%d, actual = %d)", c.Rc, rc)
			}
		default:
		}
	}
	if expErr := CheckOutput(NewResultFromString(stdOut.String()), c.Expected, c.Unexpected); expErr != nil {
		return fmt.Errorf("%s in stdout", expErr.Error())
	} else if expErr = CheckOutput(NewResultFromString(stdErr.String()), c.Expected, c.Unexpected); expErr != nil {
		return fmt.Errorf("%s in stderr", expErr.Error())
	}
	log.Debugf("check %s successfully executed", c.String())
	return nil
}
