package main

import (
	"errors"
	"math"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/mig"
)

var (
	minVersion = int64(0)
	maxVersion = int64(math.MaxInt64)
)

var rootCmd = &cobra.Command{
	Use:   "mig",
	Short: "mig is a database migration tool for Postgres and MySQL.",
	Long:  "mig is a database migration tool for Postgres and MySQL.",
	Example: `mig up postgres "user=postgres dbname=postgres sslmode=disable"
mig down mysql "user:password@/dbname"
mig create add_users`,
}

func init() {
	// Set the mig library logger to os.Stdout
	mig.Log = os.Stdout

	rootCmd.Flags().BoolP("version", "", false, "Print the mig tool version")
	viper.BindPFlags(rootCmd.Flags())
}

// getConnArgs takes in args from cobra and returns the 0th and 1st arg
// which should be the driver and connection string
func getConnArgs(args []string) (driver string, conn string, err error) {
	if len(args) < 2 {
		return "", "", errors.New("no connection details provided")
	}

	return args[0], args[1], nil
}
