package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func parseDotEnv(filePath string) map[string]string {
	env := make(map[string]string)
	f, err := os.Open(filePath)
	if err != nil {
		return env
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		env[key] = val
	}
	return env
}

func main() {
	rootDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Docker] Failed to determine working directory: %v\n", err)
		os.Exit(1)
	}

	fileEnv := parseDotEnv(filepath.Join(rootDir, ".env"))
	rawAppEnv := os.Getenv("APP_ENV")
	if strings.TrimSpace(rawAppEnv) == "" {
		rawAppEnv = fileEnv["APP_ENV"]
	}
	if strings.TrimSpace(rawAppEnv) == "" {
		rawAppEnv = "local"
	}
	rawAppEnv = strings.ToLower(strings.TrimSpace(rawAppEnv))

	isProduction := rawAppEnv == "production"
	overlayFile := "compose.local.yaml"
	modeLabel := fmt.Sprintf("LOCAL (%s)", rawAppEnv)
	if isProduction {
		overlayFile = "compose.prod.yaml"
		modeLabel = "PRODUCTION (prd)"
	}

	userArgs := os.Args[1:]
	if len(userArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: go run ./scripts/docker.go <up|down|build|logs|config> [...args]")
		os.Exit(1)
	}

	composeArgs := append([]string{"compose", "-f", "compose.yaml", "-f", overlayFile}, userArgs...)
	fmt.Printf("[Docker] APP_ENV=%q -> Using %s configuration (%s)\n", rawAppEnv, modeLabel, overlayFile)

	cmd := exec.Command("docker", composeArgs...)
	cmd.Dir = rootDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "APP_ENV="+rawAppEnv)

	if runErr := cmd.Run(); runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "[Docker] Failed to execute docker: %v\n", runErr)
		os.Exit(1)
	}
}
