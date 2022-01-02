package codegen

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/moihn/oramodelgen/internal/pkg/config"
	"github.com/moihn/oramodelgen/internal/pkg/dbmodel"
	"github.com/sirupsen/logrus"
)

//go:embed table.go.tmpl
var tableTemplate string
var tableCodegen *template.Template

//go:embed query.go.tmpl
var queryTemplate string
var queryCodegen *template.Template

var fns = template.FuncMap{
	"incr": func(x int) int {
		return x + 1
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

func getQueryCodegen() *template.Template {
	if queryCodegen == nil {
		var err error
		queryCodegen, err = template.New("query.go.tmpl").Funcs(fns).Parse(queryTemplate)
		if err != nil {
			logrus.Fatalf("failed to compile query template: %v", err)
		}
	}
	return queryCodegen
}

func GenerateTableCode(tableModel dbmodel.DbTableModel, tableDef config.TableDef, packageName string) (string, []byte) {
	codegenModel := FromDbTableModel(tableModel, tableDef)
	codegenModel.Package = packageName
	tableCodegen := getTableCodegen()
	var buf bytes.Buffer
	err := tableCodegen.Execute(&buf, codegenModel)
	if err != nil {
		logrus.Fatalf("failed to build code for table %v: %v", tableModel.Name, err)
	}
	return codegenModel.TableCamelName, buf.Bytes()
}

func GenerateQueryCode(queryModel dbmodel.DbQueryModel, queryDef config.QueryDef, packageName string) (string, []byte) {
	codegenModel := FromDbQueryModel(queryModel, queryDef)
	codegenModel.Package = packageName
	queryCodegen := getQueryCodegen()
	var buf bytes.Buffer
	err := queryCodegen.Execute(&buf, codegenModel)
	if err != nil {
		logrus.Fatalf("failed to build code for query %v: %v", queryModel.Name, err)
	}
	return codegenModel.QueryMethodName + "Query", buf.Bytes()
}
