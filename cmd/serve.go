package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/yoshino-s/derperer/internal/derperer"
	"github.com/yoshino-s/derperer/internal/handler/http"
	"github.com/yoshino-s/derperer/pkg/speedtest"
	"github.com/yoshino-s/go-app/fofa"
)

func init() {
	httpApp.Configuration().Register(serveCmd.Flags())
	derpererService.Configuration().Register(serveCmd.Flags())
	fofaApp.Configuration().Register(serveCmd.Flags())

	rootCmd.AddCommand(serveCmd)
}

var (
	httpApp         = http.New()
	derpererService = derperer.New()
	fofaApp         = fofa.New()

	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: `Serve runs the HTTP server.`,
		Run: func(cmd *cobra.Command, args []string) {
			app.Append(speedtest.New())
			app.Append(fofaApp)
			app.Append(derpererService)

			app.Append(httpApp)

			app.Go(context.Background())
		},
	}
)
