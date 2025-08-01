package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	CharmLog "github.com/charmbracelet/log"
	"gopkg.in/yaml.v3"
)

var logger = CharmLog.NewWithOptions(os.Stderr, CharmLog.Options{
	ReportTimestamp: true,
	TimeFormat:      time.Kitchen,
	Prefix:          "Orchestrator Service 🎻",
})

type serviceEntry struct {
	Name       string   `yaml:"name"`
	Path       string   `yaml:"path"`
	MaxRetries int      `yaml:"max_retries"`
	Env        []string `yaml:"env,omitempty"`
}

type manifest struct {
	Services []serviceEntry `yaml:"services"`
}

type service struct {
	name       string
	path       string
	maxRetries int
	env        []string
}

func launch(ctx context.Context, svc service, wg *sync.WaitGroup) {
	defer wg.Done()
	retries := 0
	logger.Info("Launching service", "name", svc.name)

	for {
		if svc.maxRetries > 0 && retries >= svc.maxRetries {
			logger.Error("max retries reached", "service", svc.name, "retries", retries)
			return
		}

		select {
		case <-ctx.Done():
			logger.Info("received stop signal", "name", svc.name)
			return
		default:
			retries++
			logger.Info("starting service", "service", svc.name, "attempt", retries)

			cmd := exec.CommandContext(ctx, svc.path)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = append(os.Environ(), svc.env...)

			if err := cmd.Start(); err != nil {
				logger.Error("failed to start service", "name", svc.name, "err", err)
				time.Sleep(2 * time.Second)
				continue
			}

			pid := cmd.Process.Pid
			logger.Info("service started", "service", svc.name, "pid", pid)

			err := cmd.Wait()
			logger.Warn("service exited, attempting restart", "service", svc.name, "err", err)
			time.Sleep(2 * time.Second) // wait before restarting
		}
	}
}

func main() {
	logger.Info("Starting orchestrator...")

	data, err := os.ReadFile("services.yaml")
	if err != nil {
		logger.Fatal("could not read services.yaml", "err", err)
	}

	var m manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		logger.Fatal("could not parse services.yaml", "err", err)
	}

	var services []service
	for _, entry := range m.Services {
		services = append(services, service{
			name:       entry.Name,
			path:       entry.Path,
			maxRetries: entry.MaxRetries,
			env:        entry.Env,
		})
	}
	if len(services) == 0 {
		logger.Fatal("no services found in manifest")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		signal := <-sigs
		logger.Info("caught signal; shutting down", "signal", signal)
		cancel()
	}()

	var wg sync.WaitGroup
	for _, service := range services {
		wg.Add(1)
		go launch(ctx, service, &wg)
	}

	wg.Wait()
	logger.Info("Orchestrator shutdown complete")
}
