package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "docker-doctor",
	Short: "A lightweight CLI tool to scan Docker host health",
	Long: `Docker Doctor is a tool to diagnose Docker host issues like disk usage,
log bloat, restart loops, OOM kills, daemon/config issues, and networking problems.
It generates reports in JSON, HTML, or Markdown formats.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

type exitCoder interface {
	ExitCode() int
}

// ExitError allows commands to exit with a specific exit code.
// If Err is nil, no error message is printed.
type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) ExitCode() int { return e.Code }
func (e ExitError) Unwrap() error { return e.Err }
func (e ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		if ee, ok := err.(exitCoder); ok {
			if msg := strings.TrimSpace(err.Error()); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
			}
			os.Exit(ee.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
}

var configFile string

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "doctor.yml", "config file")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}