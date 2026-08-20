package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func queueWorkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "queue:work",
		Short: "Start a queue worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueueWork()
		},
	}
}

func runQueueWork() error {
	if err := mustProjectRoot(); err != nil {
		return err
	}
	if err := ensureJobsRegistry("app/jobs"); err != nil {
		return err
	}
	if err := writeQueueRunner(); err != nil {
		return err
	}
	fmt.Println("gai queue:work")
	return runGo("run", "./app/jobs/cmd")
}

func writeQueueRunner() error {
	module := detectModule()
	src := formatGoSource(fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hlgxz/gai"
	"github.com/Hlgxz/gai/queue"
	"%s/app/jobs"
)

func main() {
	app := gai.New()
	app.LoadConfig("config")
	app.UseServices()

	mgr := gai.Make[*queue.Manager](app.Container, "queue")
	jobs.Register(mgr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("gai queue worker started")
	if err := mgr.Work(ctx); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`, module))
	return writeFile("app/jobs/cmd/main.go", src)
}
