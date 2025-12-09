package check

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
)

// ExecCommand is mockable version of os/exec.Command
var ExecCommand func(name string, arg ...string) *exec.Cmd = exec.Command

// Exec is mockable version of os/exec.Command.Run
var Exec func(name string, arg ...string) ([]byte, []byte, error) = func(name string, arg ...string) ([]byte, []byte, error) {
	cmdStr := name
	if len(arg) > 0 {
		cmdStr += " " + strings.Join(arg, " ")
	}
	log.Printf("Exec: %s\n", cmdStr)
	cmd := exec.Command(name, arg...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	if err != nil {
		log.Printf("Exec failed: %s (error: %v)\n%s\n%s", cmdStr, err, stdoutBuf.String(), stderrBuf.String())
	}
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}

// ExecString is a convenience function that returns the stdout and stderr as strings
// and trims the whitespace. It uses mockable Exec.
func ExecString(name string, arg ...string) (string, string, error) {
	stdout, stderr, err := Exec(name, arg...)
	return strings.TrimSpace(string(stdout)), strings.TrimSpace(string(stderr)), err
}

// Exists is mockable version of os.Stat
var Exists func(name string) bool = func(name string) bool {
	log.Printf("Exists: %s\n", name)
	_, err := os.Stat(name)
	exists := !os.IsNotExist(err)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Exists failed: %s (error: %v)\n", name, err)
	}
	return exists
}

// Grep is mockable version of os.ReadFile with grep capabilities
var Grep func(pattern, filename string) (bool, error) = func(pattern, filename string) (bool, error) {
	log.Printf("Grep: %s %s\n", pattern, filename)
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Grep failed: %s %s (error: %v)\n", pattern, filename, err)
		return false, err
	}
	return strings.Contains(string(content), pattern), nil
}

// ReadFile is mockable version of os.ReadFile
var ReadFile func(filename string) ([]byte, error) = func(filename string) ([]byte, error) {
	log.Printf("ReadFile: %s\n", filename)
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("ReadFile failed: %s (error: %v)\n", filename, err)
	}
	return data, err
}
