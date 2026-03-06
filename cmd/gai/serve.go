package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var port int
	var watch bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the development server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if watch {
				return serveWithWatch(port)
			}
			return serveOnce(port)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for file changes and auto-restart")

	return cmd
}

func serveOnce(port int) error {
	fmt.Printf("Building and starting server on :%d...\n", port)

	binary := filepath.Join("tmp", "main")
	if isWindows() {
		binary += ".exe"
	}

	os.MkdirAll("tmp", 0o755)

	buildCmd := exec.Command("go", "build", "-o", binary, ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	runCmd := exec.Command(binary)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	runCmd.Env = append(os.Environ(), fmt.Sprintf("APP_PORT=%d", port))
	return runCmd.Run()
}

func serveWithWatch(port int) error {
	fmt.Printf("Starting development server with hot-reload on :%d...\n", port)
	fmt.Println("Watching for file changes...")

	for {
		binary := filepath.Join("tmp", "main")
		if isWindows() {
			binary += ".exe"
		}
		os.MkdirAll("tmp", 0o755)

		buildCmd := exec.Command("go", "build", "-o", binary, ".")
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			log.Printf("Build error: %v\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		runCmd := exec.Command(binary)
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr
		runCmd.Env = append(os.Environ(), fmt.Sprintf("APP_PORT=%d", port))

		if err := runCmd.Start(); err != nil {
			log.Printf("Start error: %v\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		changed := waitForChanges(".")
		if changed {
			fmt.Println("\nFile changed, restarting...")
			if runCmd.Process != nil {
				runCmd.Process.Kill()
			}
			runCmd.Wait()
		}
	}
}

func waitForChanges(dir string) bool {
	modTimes := make(map[string]time.Time)
	collectModTimes(dir, modTimes)

	for {
		time.Sleep(1 * time.Second)
		newTimes := make(map[string]time.Time)
		collectModTimes(dir, newTimes)

		for path, newTime := range newTimes {
			if oldTime, ok := modTimes[path]; !ok || !newTime.Equal(oldTime) {
				return true
			}
		}
	}
}

func collectModTimes(dir string, times map[string]time.Time) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "tmp" || name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".go" || ext == ".yaml" || ext == ".yml" || ext == ".env" {
			times[path] = info.ModTime()
		}
		return nil
	})
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}
