package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/mig"
)

var upCmd = &cobra.Command{
	Use:     "up",
	Short:   "Migrate the database to the most recent version available",
	Long:    "Migrate the database to the most recent version available",
	Example: `mig up mysql "user:password@/dbname"`,
	RunE:    upRunE,
}

var upOneCmd = &cobra.Command{
	Use:     "upone",
	Short:   "Migrate the database by one version",
	Long:    "Migrate the database by one version",
	Example: `mig upone mysql "user:password@/dbname"`,
	RunE:    upOneRunE,
}

func init() {
	upCmd.Flags().StringP("dir", "d", ".", "directory with migration files")
	upOneCmd.Flags().StringP("dir", "d", ".", "directory with migration files")

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(upOneCmd)

	upCmd.PreRun = func(*cobra.Command, []string) {
		viper.BindPFlags(upCmd.Flags())
	}
	upOneCmd.PreRun = func(*cobra.Command, []string) {
		viper.BindPFlags(upOneCmd.Flags())
	}
}

func upRunE(cmd *cobra.Command, args []string) error {
	driver, conn, err := getConnArgs(args)
	if err != nil {
		return err
	}

	count, err := mig.Up(driver, conn, viper.GetString("dir"))
	if err != nil {
		return err
	}

	if count == 0 {
		fmt.Printf("No migrations to run")
	} else {
		fmt.Printf("Success   %d migrations\n", count)
	}

	return nil
}

func upOneRunE(cmd *cobra.Command, args []string) error {
	driver, conn, err := getConnArgs(args)
	if err != nil {
		return err
	}

	name, err := mig.UpOne(driver, conn, viper.GetString("dir"))
	if mig.IsNoMigrationError(err) {
		fmt.Println("No migrations to run")
		return nil
	} else if err != nil {
		return err
	}

	fmt.Printf("Success   %v\n", name)
	return nil
}
