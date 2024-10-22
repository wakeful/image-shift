package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/wakeful/image-shift/cmd/sub"
)

func main() {
	rootCmd := sub.NewShiftCmd(slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: slog.LevelInfo},
	)))

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
