package test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/moihn/oramodelgen/internal/pkg/codegen"
	"github.com/moihn/oramodelgen/internal/pkg/config"
	"github.com/moihn/oramodelgen/internal/pkg/modelgen/query"
	"github.com/moihn/oramodelgen/internal/pkg/modelgen/table"
)

func TestGenColumns(t *testing.T) {
	tx := GetDbTransaction()
	defer tx.Rollback()
	tdef := config.TableDef{
		Name: "TEST_TABLE_A",
		Populate: []config.TablePopulateDef{
			{
				By: []string{
					"Name",
				},
				Orderby: []config.OrderBy{
					{
						Column:     "Id",
						Descending: false,
					},
				},
			},
		},
	}
	tableModel := table.GenerateTableModel(tx, tdef)
	jenc := json.NewEncoder(os.Stdout)
	jenc.SetIndent("", " ")
	jenc.Encode(tableModel)

	codegenTableModel := codegen.FromDbTableModel(tableModel, tdef)
	jenc.Encode(codegenTableModel)

	os.Stdout.Write(codegen.GenerateTableCode(tableModel, tdef, "table"))
}

func TestGenQuery(t *testing.T) {
	tx := GetDbTransaction()
	defer tx.Rollback()
	qdef := config.QueryDef{
		Name: "getItems",
		Parameters: []config.ParameterDef{
			{
				Name: "id",
				Type: "int",
			},
		},
		Query: `
			select name
			from test_table_a
			where id > :id
		`,
	}
	queryModel := query.GenerateQueryModel(tx, qdef)
	jenc := json.NewEncoder(os.Stdout)
	jenc.SetIndent("", " ")
	jenc.Encode(queryModel)

	codegenTableModel := codegen.FromDbQueryModel(queryModel, qdef)
	jenc.Encode(codegenTableModel)

	os.Stdout.Write(codegen.GenerateQueryCode(queryModel, qdef, "query"))
}
