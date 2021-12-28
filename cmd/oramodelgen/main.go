package main

import (
	"database/sql"
	"flag"
	"os"
	"path"

	_ "github.com/godror/godror"
	"github.com/moihn/oramodelgen/internal/pkg/config"
	"github.com/moihn/oramodelgen/internal/pkg/modelgen"
	"github.com/sirupsen/logrus"
)

func LogInit(logLevelName string) {
	// Only log the warning severity or above.
	if logLevel, err := logrus.ParseLevel(logLevelName); err != nil {
		logrus.Fatal("Fail to parse logging level: ", logLevel)
	} else {
		logrus.SetLevel(logLevel)
		logrus.SetOutput(os.Stderr)
	}
}

func main() {
	var configFile, dbConnString, logLevelName, modelFile, outputPackage, outputDir string
	var printVersion bool
	flag.StringVar(&configFile, "config", "", "Configuration YAML file")
	flag.StringVar(&dbConnString, "dbConnectString", "", "Oracle database connection string.")
	flag.StringVar(&modelFile, "model", "m", "Model YAML file")
	flag.StringVar(&outputDir, "outdir", "P", "Output package code under given directory")
	flag.StringVar(&outputPackage, "outputPackage", "P", "Output code under given package")
	flag.StringVar(&logLevelName, "log-level", "warn", "Log level")
	flag.BoolVar(&printVersion, "version", false, "Show version of the executable")
	flag.Parse()

	LogInit(logLevelName)

	if len(configFile) > 0 {
		configString, err := os.ReadFile(configFile)
		if err != nil {
			logrus.Fatalf("failed to read file %v: %v", configFile, err)
		}
		config := config.LoadConfig(configString)
		if len(dbConnString) == 0 && config.DbConnectString != nil {
			dbConnString = *config.DbConnectString
		}
	}

	// get db transaction
	logrus.Debugf("DbConnectionString: %v", dbConnString)
	db, err := sql.Open("godror", dbConnString)
	if err != nil {
		logrus.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		logrus.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		logrus.Fatal(err)
	}
	// we don't change database
	defer tx.Rollback()

	modelDefString, err := os.ReadFile(modelFile)
	if err != nil {
		logrus.Fatalf("failed to read file %v: %v", modelFile, err)
	}
	modelDef := config.LoadModelConfig(modelDefString)
	codes := modelgen.Generate(tx, modelDef, outputPackage)
	packageDir := path.Join(outputDir, outputPackage)
	err = os.MkdirAll(packageDir, 0755)
	if err != nil {
		logrus.Fatalf("failed to create output package directory %v: %v", packageDir, err)
	}
	for codeName, code := range codes {
		fileName := codeName + ".go"
		filePath := path.Join(packageDir, fileName)
		if err = os.WriteFile(filePath, code, 0644); err != nil {
			logrus.Fatalf("failed to write code for %v: %v", codeName, err)
		}
	}
}
