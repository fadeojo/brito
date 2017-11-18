package controllers

import (
	"fmt"
	"os"
	"testing"

	"github.com/fadeojo/brito/app"
	"github.com/fadeojo/brito/db"
	"github.com/fadeojo/brito/rendering"
	"github.com/volatiletech/abcweb/abcdatabase"
	"github.com/volatiletech/abcweb/abcsessions"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newRootMock returns a Root struct with all members initialized to
// test compatible mock values
func newRootMock(templatesDir string) Root {

	// Set up zap logger. Use zap.NewNop() if you wish to disable logging.
	zapCfg := zap.NewDevelopmentConfig()
	zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapCfg.OutputPaths = []string{"stdout"}
	log, err := zapCfg.Build()
	if err != nil {
		panic(err)
	}

	// Set up the template renderer
	a := app.NewApp()
	a.Config.Server.RenderRecompile = true

	return Root{

		Log:    log,
		Render: rendering.New(a, templatesDir, nil),
	}
}

func TestMain(m *testing.M) {
	var err error
	var count int
	db.DB, count, err = abcdatabase.SetupTestSuite(db.GoTestdata)
	if err != nil {
		fmt.Printf("TestMain setup failed: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("TestMain Setup ran %d migrations.\n", count)

	r := m.Run()
	os.Exit(r)
}
