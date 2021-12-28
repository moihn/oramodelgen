package test

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/godror/godror"
	"github.com/moihn/oramodelgen/internal/pkg/config"
	"github.com/sirupsen/logrus"
)

var db *sql.DB

func GetDbTransaction() *sql.Tx {
	if db == nil {
		configFile, found := os.LookupEnv("TEST_CONFIG_PATH")
		if !found {
			configFile = "testConfig.yml"
		}
		configAbsFile, err := filepath.Abs(configFile)
		if err != nil {
			logrus.Fatalf("failed to read from file %v: %v", configFile, err)
		}
		configString, err := os.ReadFile(configAbsFile)
		if err != nil {
			logrus.Fatalf("failed to read from file %v: %v", configAbsFile, err)
		}
		dbConnString := config.LoadConfig(configString).DbConnectString
		if dbConnString == nil {
			logrus.Fatalf("DbConnectString field is not found in test configuration file %v", configAbsFile)
		}
		logrus.Debugf("DbConnectionString: %v", *dbConnString)
		db, err = sql.Open("godror", *dbConnString)
		if err != nil {
			logrus.Fatalf("failed to open connection: %v", err)
		}
	}
	if err := db.Ping(); err != nil {
		logrus.Fatalf("failed to ping remote database server: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		logrus.Fatalf("failed to start transaction: %v", err)
	}
	return tx
}
