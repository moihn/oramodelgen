package modelgen

import (
	"database/sql"

	"github.com/moihn/oramodelgen/internal/pkg/codegen"
	"github.com/moihn/oramodelgen/internal/pkg/config"
	"github.com/moihn/oramodelgen/internal/pkg/modelgen/query"
	"github.com/moihn/oramodelgen/internal/pkg/modelgen/table"
)

func Generate(tx *sql.Tx, modelConfig config.ModelConfig, outputPackage string) map[string][]byte {
	m := map[string][]byte{}

	dbTableModels := table.GenerateTableModels(tx, modelConfig.Tables)
	dbQueryModels := query.GenerateQueryModels(tx, modelConfig.Queries)

	for index, dbTableModel := range dbTableModels {
		m[dbTableModel.Name] = codegen.GenerateTableCode(dbTableModel, modelConfig.Tables[index], outputPackage)
	}

	for index, dbQueryModel := range dbQueryModels {
		m[dbQueryModel.Name] = codegen.GenerateQueryCode(dbQueryModel, modelConfig.Queries[index], outputPackage)
	}

	// return the code
	return m
}
