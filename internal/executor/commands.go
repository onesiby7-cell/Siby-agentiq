package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type CommandExecutor struct {
	mu         sync.Mutex
	workingDir string
	env        map[string]string
	timeout    time.Duration
}

type CommandResult struct {
	Command   string
	Output    string
	Error     string
	ExitCode  int
	Duration  time.Duration
	Success   bool
}

func NewCommandExecutor(workingDir string) *CommandExecutor {
	if workingDir == "" {
		workingDir, _ = os.Getwd()
	}
	return &CommandExecutor{
		workingDir: workingDir,
		env:        make(map[string]string),
		timeout:    5 * time.Minute,
	}
}

func (ce *CommandExecutor) SetWorkingDir(dir string) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.workingDir = dir
}

func (ce *CommandExecutor) SetEnv(key, value string) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.env[key] = value
}

func (ce *CommandExecutor) SetTimeout(d time.Duration) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.timeout = d
}

func (ce *CommandExecutor) Execute(ctx context.Context, command string) (*CommandResult, error) {
	ce.mu.Lock()
	wd := ce.workingDir
	env := ce.env
	timeout := ce.timeout
	ce.mu.Unlock()

	start := time.Now()
	
	shell := "/bin/sh"
	shellFlag := "-c"
	if strings.Contains(runtime.GOOS, "windows") {
		shell = "cmd.exe"
		shellFlag = "/c"
	}

	cmd := exec.CommandContext(ctx, shell, shellFlag, command)
	cmd.Dir = wd

	envCopy := os.Environ()
	for k, v := range env {
		envCopy = append(envCopy, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = envCopy

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-ctx.Done():
		cmd.ProcessKill()
		return &CommandResult{
			Command:  command,
			Error:    "timeout",
			Duration: time.Since(start),
			Success:  false,
		}, ctx.Err()

	case err := <-done:
		result := &CommandResult{
			Command:  command,
			Output:   stdout.String(),
			Error:    stderr.String(),
			Duration: time.Since(start),
			Success:  err == nil,
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}

		return result, nil
	}
}

func (ce *CommandExecutor) ExecuteMultiple(ctx context.Context, commands []string) ([]*CommandResult, error) {
	var results []*CommandResult
	
	for _, cmd := range commands {
		result, err := ce.Execute(ctx, cmd)
		results = append(results, result)
		if err != nil && result.ExitCode != 0 {
			return results, fmt.Errorf("command failed: %s", cmd)
		}
	}
	
	return results, nil
}

func (ce *CommandExecutor) ParseCommandsFromOutput(output string) []string {
	var commands []string
	scanner := strings.NewScanner(output)
	
	var inCommandBlock bool
	var currentCmd strings.Builder
	
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(strings.ToUpper(line), "```bash") ||
		   strings.HasPrefix(strings.ToUpper(line), "```sh") ||
		   strings.HasPrefix(strings.ToUpper(line), "```shell") {
			if inCommandBlock {
				if cmd := strings.TrimSpace(currentCmd.String()); cmd != "" {
					commands = append(commands, cmd)
				}
				currentCmd.Reset()
			}
			inCommandBlock = !inCommandBlock
			continue
		}
		
		if inCommandBlock {
			if currentCmd.Len() > 0 {
				currentCmd.WriteString(" ")
			}
			currentCmd.WriteString(line)
		}
	}
	
	if cmd := strings.TrimSpace(currentCmd.String()); cmd != "" {
		commands = append(commands, cmd)
	}
	
	return commands
}

func ExtractAndExecute(ctx context.Context, llmOutput, workingDir string) ([]*CommandResult, error) {
	ce := NewCommandExecutor(workingDir)
	commands := ce.ParseCommandsFromOutput(llmOutput)
	
	if len(commands) == 0 {
		return nil, nil
	}
	
	return ce.ExecuteMultiple(ctx, commands)
}

func CommonCommands() map[string]string {
	return map[string]string{
		"go mod tidy":  "go mod tidy",
		"go build":     "go build ./...",
		"go test":      "go test ./...",
		"go vet":       "go vet ./...",
		"npm install":  "npm install",
		"npm build":    "npm run build",
		"npm test":     "npm test",
		"pip install":  "pip install -r requirements.txt",
		"cargo build":  "cargo build",
		"cargo test":   "cargo test",
		"rustfmt":      "rustfmt src/**/*.rs",
		"make build":   "make build",
		"make test":    "make test",
		"docker build": "docker build .",
	}
}

func IsCommonCommand(cmd string) bool {
	common := CommonCommands()
	normalized := strings.TrimSpace(cmd)
	for _, c := range common {
		if normalized == c || normalized == strings.Split(c, " ")[0] {
			return true
		}
	}
	return false
}
