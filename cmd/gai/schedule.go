package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func scheduleRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schedule:run",
		Short: "Run the application scheduler",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSchedule()
		},
	}
}

func runSchedule() error {
	if err := mustProjectRoot(); err != nil {
		return err
	}
	if err := ensureScheduleRegistry("app/console"); err != nil {
		return err
	}
	if err := writeScheduleRunner(); err != nil {
		return err
	}
	fmt.Println("gai schedule:run")
	return runGo("run", "./app/console/cmd")
}

func writeScheduleRunner() error {
	module := detectModule()
	src := formatGoSource(fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hlgxz/gai"
	"github.com/Hlgxz/gai/schedule"
	"%s/app/console"
)

func main() {
	app := gai.New()
	app.LoadConfig("config")
	app.UseServices()

	s := gai.Make[*schedule.Scheduler](app.Container, "schedule")
	console.Register(s)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("gai scheduler started")
	s.Run(ctx)
}
`, module))
	return writeFile("app/console/cmd/main.go", src)
}

func ensureScheduleRegistry(dir string) error {
	path := dir + "/register.go"
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	src := `package console

import (
	"github.com/Hlgxz/gai/schedule"
)

// Register attaches scheduled tasks.
func Register(s *schedule.Scheduler) {
}
`
	return writeFile(path, src)
}
