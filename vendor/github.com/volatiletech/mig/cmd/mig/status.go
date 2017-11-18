package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/mig"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Dump the migration status for the database",
	Long:    "Dump the migration status for the database",
	Example: `mig status postgres "user=postgres dbname=postgres sslmode=disable"`,
	RunE:    statusRunE,
}

func init() {
	statusCmd.Flags().StringP("dir", "d", ".", "directory with migration files")

	rootCmd.AddCommand(statusCmd)

	statusCmd.PreRun = func(*cobra.Command, []string) {
		viper.BindPFlags(statusCmd.Flags())
	}
}

func statusRunE(cmd *cobra.Command, args []string) error {
	driver, conn, err := getConnArgs(args)
	if err != nil {
		return err
	}

	status, err := mig.Status(driver, conn, viper.GetString("dir"))
	if err != nil {
		return err
	}

	if len(status) == 0 {
		fmt.Printf("No migrations applied")
		return nil
	}

	fmt.Println("Applied At                  Migration")
	fmt.Println("===================================================")
	for _, s := range status {
		fmt.Printf("%-24s -- %v\n", s.Applied, s.Name)
	}

	return nil
}
