package codegen

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/moihn/oramodelgen/internal/pkg/dbmodel"
	"github.com/sirupsen/logrus"
)

//go:embed table.go.tmpl
var tableTemplate string
var tableCodegen *template.Template

var fns = template.FuncMap{
    "incr": func(x int) int {
        return x+1
    },
}

func getTableCodegen() *template.Template {
	if tableCodegen == nil {
		var err error
		tableCodegen, err = template.New("table.go.tmpl").Funcs(fns).Parse(tableTemplate)
		if err != nil {
			logrus.Fatalf("failed to compile table template: %v", err)
		}
	}
	return tableCodegen
}

func Generate(tableModel dbmodel.DbTableModel, packageName string) []byte {
	codegenModel := FromDbModel(tableModel)
	codegenModel.Package = packageName
	tableCodegen := getTableCodegen()
	var buf bytes.Buffer
	err := tableCodegen.Execute(&buf, codegenModel)
	if err != nil {
		logrus.Fatalf("failed to build code for table %v: %v", tableModel.Name, err)
	}
	return buf.Bytes()
}
