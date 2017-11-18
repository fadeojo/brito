package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kat-co/vala"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/volatiletech/abcweb/config"
)

// migrateCmd represents the "migrate" command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run migration tasks (up, down, redo, status, version)",
	Long: `Run migration tasks on the migrations in your migrations directory.
These tasks also regenerate your models automatically, which can be disabled
using the --no-models flag.

Migrations can be generated by using the "abcweb gen migration" command.
`,
	Example: "abcweb migrate up\nabcweb migrate down",
}

var errNoMigrations = errors.New("No migrations to run")

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Migrate the database to the most recent version",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := migrateExec(cmd, args, "up")
		if err != nil && err != errNoMigrations {
			return err
		}

		if !cnf.ModeViper.GetBool("no-models") && err != errNoMigrations {
			return modelsExec(cmd, args)
		}

		return nil
	},
}

var upOneCmd = &cobra.Command{
	Use:   "upone",
	Short: "Migrate the database by one version",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := migrateExec(cmd, args, "upone")
		if err != nil && err != errNoMigrations {
			return err
		}

		if !cnf.ModeViper.GetBool("no-models") && err != errNoMigrations {
			return modelsExec(cmd, args)
		}

		return nil
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Roll back the version by one",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := migrateExec(cmd, args, "down")
		if err != nil && err != errNoMigrations {
			return err
		}

		if !cnf.ModeViper.GetBool("no-models") && err != errNoMigrations {
			return modelsExec(cmd, args)
		}

		return nil
	},
}

var downAllCmd = &cobra.Command{
	Use:   "downall",
	Short: "Roll back all migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := migrateExec(cmd, args, "downall")
		if err != nil && err != errNoMigrations {
			return err
		}

		if !cnf.ModeViper.GetBool("no-models") && err != errNoMigrations {
			return modelsExec(cmd, args)
		}

		return nil
	},
}

var redoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Down then up the latest migration",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := migrateExec(cmd, args, "redo")
		if err != nil && err != errNoMigrations {
			return err
		}

		if !cnf.ModeViper.GetBool("no-models") && err != errNoMigrations {
			return modelsExec(cmd, args)
		}

		return nil
	},
}

var redoAllCmd = &cobra.Command{
	Use:   "redoall",
	Short: "Down then up all migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := migrateExec(cmd, args, "redoall")
		if err != nil && err != errNoMigrations {
			return err
		}

		if !cnf.ModeViper.GetBool("no-models") && err != errNoMigrations {
			return modelsExec(cmd, args)
		}

		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Dump the migration status for the current database",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrateExec(cmd, args, "status")
	},
}

var dbVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current version of the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		return migrateExec(cmd, args, "version")
	},
}

func init() {
	basepath, err := config.GetBasePath()
	if err != nil {
		panic(fmt.Sprintf("unable to get abcweb base path: %s", err))
	}

	replaceArgs := make([]string, len(replaceFiles))

	// Prefix the replaceWith file with the basepath
	for i := 0; i < len(replaceFiles); i++ {
		replaceArgs[i] = fmt.Sprintf("%s:%s", replaceFiles[i][0], filepath.Join(basepath, replaceFiles[i][1]))
	}

	// migrate flags
	migrateCmd.PersistentFlags().StringP("db", "", "", `Valid options: postgres|mysql (default: "config.toml db field")`)
	migrateCmd.PersistentFlags().StringP("env", "e", "dev", "The config.toml file environment to load")

	// Up/Down/Redo flags
	modelsFlags := &pflag.FlagSet{}
	modelsFlags.StringP("schema", "s", "public", "The name of your database schema, for databases that support real schemas")
	modelsFlags.StringP("pkgname", "p", "models", "The name you wish to assign to your generated package")
	modelsFlags.StringP("output", "o", filepath.FromSlash("db/models"), "The name of the folder to output to. Automatically created relative to webapp root dir")
	modelsFlags.StringP("basedir", "", "", "The base directory has the templates and templates_test folders")
	modelsFlags.StringSliceP("blacklist", "b", nil, "Do not include these tables in your generated package")
	modelsFlags.StringSliceP("whitelist", "w", nil, "Only include these tables in your generated package")
	modelsFlags.StringSliceP("tag", "t", nil, "Struct tags to be included on your models in addition to json, yaml, toml")
	modelsFlags.StringSliceP("replace", "", replaceArgs, "Replace templates by directory: relpath/to_file.tpl:relpath/to_replacement.tpl")
	modelsFlags.MarkHidden("replace")
	modelsFlags.BoolP("debug", "d", false, "Debug mode prints stack traces on error")
	modelsFlags.BoolP("no-tests", "", false, "Disable generated go test files")
	modelsFlags.BoolP("no-hooks", "", false, "Disable hooks feature for your models")
	modelsFlags.BoolP("no-auto-timestamps", "", false, "Disable automatic timestamps for created_at/updated_at")
	modelsFlags.BoolP("tinyint-not-bool", "", false, "Map MySQL tinyint(1) in Go to int8 instead of bool")
	modelsFlags.BoolP("wipe", "", true, "Delete the output folder (rm -rf) before generation to ensure sanity")
	// no models flag
	modelsFlags.BoolP("no-models", "", false, "Disable model generation after migration command")
	// hide flags not recommended for use

	upCmd.Flags().AddFlagSet(modelsFlags)
	upOneCmd.Flags().AddFlagSet(modelsFlags)
	downCmd.Flags().AddFlagSet(modelsFlags)
	downAllCmd.Flags().AddFlagSet(modelsFlags)
	redoCmd.Flags().AddFlagSet(modelsFlags)
	redoAllCmd.Flags().AddFlagSet(modelsFlags)

	RootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(upCmd)
	migrateCmd.AddCommand(upOneCmd)
	migrateCmd.AddCommand(downCmd)
	migrateCmd.AddCommand(downAllCmd)
	migrateCmd.AddCommand(redoCmd)
	migrateCmd.AddCommand(redoAllCmd)
	migrateCmd.AddCommand(statusCmd)
	migrateCmd.AddCommand(dbVersionCmd)

	migrateCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Usually the RootCmd persistent pre-run does this init for us,
		// but since we have to override the persistent pre-run here
		// to provide configuration to all children commands, we have to
		// do the init ourselves.
		var err error
		cnf, err = config.Initialize(cmd.Flags().Lookup("env"))
		if err != nil {
			return err
		}

		cnf.ModeViper.BindPFlags(migrateCmd.PersistentFlags())
		cnf.ModeViper.BindPFlags(cmd.Flags())

		return nil
	}
}

func migrateExec(cmd *cobra.Command, args []string, subCmd string) error {
	checkDep("mig")

	migrateCmdConfig.DB = cnf.ModeViper.GetString("db")
	migrateCmdConfig.DBName = cnf.ModeViper.GetString("dbname")
	migrateCmdConfig.User = cnf.ModeViper.GetString("user")
	migrateCmdConfig.Pass = cnf.ModeViper.GetString("pass")
	migrateCmdConfig.Host = cnf.ModeViper.GetString("host")
	migrateCmdConfig.Port = cnf.ModeViper.GetInt("port")
	migrateCmdConfig.SSLMode = cnf.ModeViper.GetString("sslmode")

	var connStr string
	if migrateCmdConfig.DB == "postgres" {
		if migrateCmdConfig.SSLMode == "" {
			migrateCmdConfig.SSLMode = "require"
			cnf.ModeViper.Set("sslmode", migrateCmdConfig.SSLMode)
		}

		if migrateCmdConfig.Port == 0 {
			migrateCmdConfig.Port = 5432
			cnf.ModeViper.Set("port", migrateCmdConfig.Port)
		}
		connStr = postgresConnStr(migrateCmdConfig)
	} else if migrateCmdConfig.DB == "mysql" {
		if migrateCmdConfig.SSLMode == "" {
			migrateCmdConfig.SSLMode = "true"
			cnf.ModeViper.Set("sslmode", migrateCmdConfig.SSLMode)
		}

		if migrateCmdConfig.Port == 0 {
			migrateCmdConfig.Port = 3306
			cnf.ModeViper.Set("port", migrateCmdConfig.Port)
		}
		connStr = mysqlConnStr(migrateCmdConfig)
	}

	err := vala.BeginValidation().Validate(
		vala.StringNotEmpty(migrateCmdConfig.DB, "db"),
		vala.StringNotEmpty(migrateCmdConfig.User, "user"),
		vala.StringNotEmpty(migrateCmdConfig.Host, "host"),
		vala.Not(vala.Equals(migrateCmdConfig.Port, 0, "port")),
		vala.StringNotEmpty(migrateCmdConfig.DBName, "dbname"),
		vala.StringNotEmpty(migrateCmdConfig.SSLMode, "sslmode"),
	).Check()

	if err != nil {
		return err
	}

	excArgs := []string{
		subCmd,
		migrateCmdConfig.DB,
		connStr,
	}

	exc := exec.Command("mig", excArgs...)
	exc.Dir = filepath.Join(cnf.AppPath, migrationsDirectory)

	out, err := exc.CombinedOutput()

	fmt.Print(string(out))
	if strings.HasPrefix(string(out), "No migrations to run") {
		return errNoMigrations
	}

	return err
}

const errNoTables = "unable to initialize tables: no tables found in database"

func modelsExec(cmd *cobra.Command, args []string) error {
	if err := modelsCmdSetup(cmd, args); err != nil {
		if err.Error() == errNoTables {
			fmt.Printf("\nNo tables found, skipping models generation.")
			return nil
		}

		fmt.Printf("\nFail   Generating models")
		return err
	}

	if err := modelsCmdState.Run(true); err != nil {
		fmt.Printf("\nFail   Generating models")
		return err
	}

	fmt.Printf("\nSuccess   Generating models")
	return nil
}

// mysqlConnStr returns a mysql connection string compatible with the
// Go mysql driver package, in the format:
// user:pass@tcp(host:port)/dbname?tls=true
func mysqlConnStr(cfg migrateConfig) string {
	var out bytes.Buffer

	out.WriteString(cfg.User)
	if len(cfg.Pass) > 0 {
		out.WriteByte(':')
		out.WriteString(cfg.Pass)
	}
	out.WriteString(fmt.Sprintf("@tcp(%s:%d)/%s", cfg.Host, cfg.Port, cfg.DBName))
	if len(cfg.SSLMode) > 0 {
		out.WriteString("?tls=")
		out.WriteString(cfg.SSLMode)
	}

	return out.String()
}

// postgressConnStr returns a postgres connection string compatible with the
// Go pq driver package, in the format:
// user=bob password=secret host=1.2.3.4 port=5432 dbname=mydb sslmode=verify-full
func postgresConnStr(cfg migrateConfig) string {
	connStrs := []string{
		fmt.Sprintf("user=%s", cfg.User),
	}

	if len(cfg.Pass) > 0 {
		connStrs = append(connStrs, fmt.Sprintf("password=%s", cfg.Pass))
	}

	connStrs = append(connStrs, []string{
		fmt.Sprintf("host=%s", cfg.Host),
		fmt.Sprintf("port=%d", cfg.Port),
		fmt.Sprintf("dbname=%s", cfg.DBName),
	}...)

	if len(cfg.SSLMode) > 0 {
		connStrs = append(connStrs, fmt.Sprintf("sslmode=%s", cfg.SSLMode))
	}

	return strings.Join(connStrs, " ")
}
