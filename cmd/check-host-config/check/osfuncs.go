package check

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
)

// ExecCommand is mockable version of os/exec.Command
var ExecCommand = exec.Command

// Exec is mockable version of os/exec.Command.Run
var Exec = func(name string, arg ...string) ([]byte, []byte, int, error) {
	cmdStr := name
	if len(arg) > 0 {
		cmdStr += " " + strings.Join(arg, " ")
	}

	cmd := exec.Command(name, arg...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
		log.Printf("Exec: %s (%s)\n%s\n%s", cmdStr, err, stdoutBuf.String(), stderrBuf.String())
	} else {
		log.Printf("Exec: %s\n", cmdStr)
	}

	return stdoutBuf.Bytes(), stderrBuf.Bytes(), exitCode, err
}

// ExecString is a convenience function that returns the stdout and stderr as strings
// and trims the whitespace. It uses mockable Exec.
func ExecString(name string, arg ...string) (string, string, int, error) {
	stdout, stderr, exitCode, err := Exec(name, arg...)
	return strings.TrimSpace(string(stdout)), strings.TrimSpace(string(stderr)), exitCode, err
}

// Exists is mockable version of os.Stat
var Exists = func(name string) bool {
	log.Printf("Exists: %s\n", name)
	_, err := os.Stat(name)
	exists := !os.IsNotExist(err)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Exists failed: %s (error: %v)\n", name, err)
	}
	return exists
}

// ExistsDir is mockable version that checks if a path exists and is a directory
var ExistsDir = func(name string) bool {
	log.Printf("ExistsDir: %s\n", name)
	info, err := os.Stat(name)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("ExistsDir failed: %s (error: %v)\n", name, err)
		}
		return false
	}
	return info.IsDir()
}

// Stat is mockable version of os.Stat
var Stat = func(name string) (os.FileInfo, error) {
	log.Printf("Stat: %s\n", name)
	return os.Stat(name)
}

// Grep is mockable version of os.ReadFile with grep capabilities
var Grep = func(pattern, filename string) (bool, error) {
	log.Printf("Grep: %s %s\n", pattern, filename)
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Grep failed: %s %s (error: %v)\n", pattern, filename, err)
		return false, err
	}
	return strings.Contains(string(content), pattern), nil
}

// ReadFile is mockable version of os.ReadFile
var ReadFile = func(filename string) ([]byte, error) {
	log.Printf("ReadFile: %s\n", filename)
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("ReadFile failed: %s (error: %v)\n", filename, err)
	}
	return data, err
}
