package test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/moihn/oramodelgen/internal/pkg/codegen"
	"github.com/moihn/oramodelgen/internal/pkg/config"
	"github.com/moihn/oramodelgen/internal/pkg/modelgen/table"
)

func TestGenColumns(t *testing.T) {
	tx := GetDbTransaction()
	tdef := config.TableDef{
		Name: "TEST_TABLE_A",
	}
	tableModel := table.GenerateTableModel(tx, tdef)
	jenc := json.NewEncoder(os.Stdout)
	jenc.SetIndent("", " ")
	jenc.Encode(tableModel)

	codegenTableModel := codegen.FromDbModel(tableModel)
	jenc.Encode(codegenTableModel)

	os.Stdout.Write(codegen.Generate(tableModel, "table"))
}
 