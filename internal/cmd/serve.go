package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yumazak/ccdc/internal/server"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start notification HTTP server on the host",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.ListenAndServe(servePort)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 5454, "port to listen on")
	rootCmd.AddCommand(serveCmd)
}
