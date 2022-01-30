package cmd

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// closeCmd represents the close command
var closeCmd = &cobra.Command{
	Use:   "close",
	Short: "Close an SSHuttle tunnel opened in the background.",
	Long:  `Close an SSHuttle tunnel opened in the background.`,
	Run: func(cmd *cobra.Command, args []string) {
		pidFile := "/tmp/tunnel.pid"
		if _, err := os.Stat(pidFile); errors.Is(err, os.ErrNotExist) {
			log.Fatal("error : no tunnel opened")
		}

		pidString, err := os.ReadFile(pidFile)
		if err != nil {
			log.Fatalf("error opening pidfile %s : %v", pidFile, err)
		}

		pid, err := strconv.Atoi(strings.TrimSuffix(string(pidString), "\n"))
		if err != nil {
			log.Fatalf("error converting pid : %v", err)
		}

		syscall.Kill(pid, syscall.SIGTERM)

		log.Print("tunnel closed")
	},
}

func init() {
	rootCmd.AddCommand(closeCmd)
}
