package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/mig"
)

var redoCmd = &cobra.Command{
	Use:     "redo",
	Short:   "Down then up the latest migration",
	Long:    "Down then up the latest migration",
	Example: `mig redo postgres "user=postgres dbname=postgres sslmode=disable"`,
	RunE:    redoRunE,
}

var redoAllCmd = &cobra.Command{
	Use:     "redo",
	Short:   "Down then up all migrations",
	Long:    "Down then up all migrations",
	Example: `mig redoall postgres "user=postgres dbname=postgres sslmode=disable"`,
	RunE:    redoAllRunE,
}

func init() {
	redoCmd.Flags().StringP("dir", "d", ".", "directory with migration files")

	rootCmd.AddCommand(redoCmd)
	redoCmd.PreRun = func(*cobra.Command, []string) {
		viper.BindPFlags(redoCmd.Flags())
	}
}

func redoRunE(cmd *cobra.Command, args []string) error {
	driver, conn, err := getConnArgs(args)
	if err != nil {
		return err
	}

	name, err := mig.Redo(driver, conn, viper.GetString("dir"))
	if mig.IsNoMigrationError(err) {
		fmt.Println("No migrations to run")
		return nil
	} else if err != nil {
		return err
	}

	fmt.Printf("Success   %v\n", name)
	return nil
}

func redoAllRunE(cmd *cobra.Command, args []string) error {
	driver, conn, err := getConnArgs(args)
	if err != nil {
		return err
	}

	_, err = mig.DownAll(driver, conn, viper.GetString("dir"))
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
