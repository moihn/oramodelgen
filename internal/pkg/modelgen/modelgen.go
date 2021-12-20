package modelgen

import (
	"database/sql"

	"github.com/moihn/oramodelgen/internal/pkg/config"
	"github.com/moihn/oramodelgen/internal/pkg/modelgen/table"
)

func Generate(tx *sql.Tx, modelConfig config.ModelConfig) map[string]string {
	m := map[string]string{}

	table.GenerateTableModels(tx, modelConfig.Tables)
	// query.GenerateQueryModel(db, model.Queries)
	return m
}
