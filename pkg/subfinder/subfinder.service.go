package subfinder

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"
)

type Service interface {
	FindSubdomains(ctx context.Context, domain string) ([]string, error)
}

type SubfinderService struct {
	SubfinderPath string
	ExecutionTimeout time.Duration
}

func NewService(subfinderPath string, timeout time.Duration) *SubfinderService {
	return &SubfinderService{
		SubfinderPath:    subfinderPath,
		ExecutionTimeout: timeout,
	}
}

func (s *SubfinderService) FindSubdomains(ctx context.Context, domain string) ([]string, error) {
	var subdomains []string

	cmdCtx, cancel := context.WithTimeout(ctx, s.ExecutionTimeout)
	defer cancel()

	cmdArgs := []string{"-d", domain, "-silent"}

	cmd := exec.CommandContext(cmdCtx, s.getExecutablePath(), cmdArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe for subfinder: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe for subfinder: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start subfinder command: %w", err)
	}

	go func() {
		errScanner := bufio.NewScanner(stderr)
		for errScanner.Scan() {
			log.Printf("Subfinder STDERR for %s: %s", domain, errScanner.Text())
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		subdomain := strings.TrimSpace(scanner.Text())
		if subdomain != "" {
			subdomains = append(subdomains, subdomain)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading subfinder stdout: %w", err)
	}

	err = cmd.Wait()
	if cmdCtx.Err() != nil {
		return subdomains, cmdCtx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("subfinder command failed for domain %s: %w", domain, err)
	}

	return subdomains, nil
}

func (s *SubfinderService) getExecutablePath() string {
	if s.SubfinderPath != "" {
		return s.SubfinderPath
	}
	return "subfinder"
}