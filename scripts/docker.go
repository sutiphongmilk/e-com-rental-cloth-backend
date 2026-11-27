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

func sanitizeSlug(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "local"
	}
	return b.String()
}

func runCmd(dir string, env []string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	if runErr := cmd.Run(); runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "[Docker] Failed to execute %s: %v\n", name, runErr)
		os.Exit(1)
	}
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

	tunnelToken := os.Getenv("CLOUDFLARE_TUNNEL_TOKEN")
	if strings.TrimSpace(tunnelToken) == "" {
		tunnelToken = fileEnv["CLOUDFLARE_TUNNEL_TOKEN"]
	}
	tunnelToken = strings.TrimSpace(tunnelToken)

	// "dev", "uat", and "production" run the compiled, low-RAM hardened Distroless container; "local" runs Air hot-reload mode
	isCompiledEnv := rawAppEnv == "production" || rawAppEnv == "uat" || rawAppEnv == "dev"
	overlayFile := "compose.local.yaml"
	modeLabel := fmt.Sprintf("LOCAL DEV (%s)", rawAppEnv)
	if isCompiledEnv {
		overlayFile = "compose.prod.yaml"
		modeLabel = fmt.Sprintf("COMPILED (%s)", rawAppEnv)
	}

	projectName := "ecom-rental-backend-" + sanitizeSlug(rawAppEnv)
	userArgs := os.Args[1:]
	if len(userArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: go run ./scripts/docker.go <up|down|build|logs|config|redeploy> [...args]")
		os.Exit(1)
	}

	cmdEnv := append(os.Environ(), "APP_ENV="+rawAppEnv)

	if userArgs[0] == "redeploy" {
		fmt.Printf("[Docker] Pulling latest Git changes before redeploy (%s)...\n", rawAppEnv)
		runCmd(rootDir, cmdEnv, "git", "pull")
		userArgs = []string{"up", "-d", "--build"}
	}

	composeFiles := []string{"-f", "compose.yaml", "-f", overlayFile}
	if tunnelToken != "" {
		composeFiles = append(composeFiles, "-f", "compose.tunnel.yaml")
	}

	composeArgs := append([]string{"compose", "-p", projectName}, composeFiles...)
	composeArgs = append(composeArgs, userArgs...)

	tunnelSuffix := ""
	if tunnelToken != "" {
		tunnelSuffix = " + compose.tunnel.yaml"
	}
	fmt.Printf("[Docker] APP_ENV=%q | project=%q -> Using %s (%s%s)\n", rawAppEnv, projectName, modeLabel, overlayFile, tunnelSuffix)
	runCmd(rootDir, cmdEnv, "docker", composeArgs...)
}
