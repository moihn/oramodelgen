package table

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"strings"

	"github.com/moihn/oramodelgen/internal/pkg/config"
	"github.com/moihn/oramodelgen/internal/pkg/dbmodel"
	"github.com/sirupsen/logrus"
)

func getCurrentDbUser(tx *sql.Tx) string {
	sqlQuery := `SELECT USER FROM DUAL`
	rows, err := tx.Query(sqlQuery)
	if err != nil {
		logrus.Fatalf("failed to query current db user: %v", err)
	}
	defer rows.Close()
	var user string
	if rows.Next() {
		err = rows.Scan(&user)
		if err != nil {
			logrus.Fatalf("failed to extract db user from query result: %v", err)
		}

	} else {
		log.Fatalf("failed to find current db user, is the db connection valid?")
	}
	return strings.ToUpper(user)
}

func resolveSynonym(tx *sql.Tx, table *config.TableDef) {
	if len(table.Owner) > 0 {
		return
	}
	dbUser := getCurrentDbUser(tx)

	// check synonyms
	sqlQuery := `
		SELECT table_owner, table_name
		FROM all_synonyms
		WHERE owner = :owner
		  AND synonym_name = :synonymName`
	synRows, err := tx.Query(sqlQuery,
		sql.Named("owner", dbUser),
		sql.Named("synonymName", table.Name),
	)
	if err != nil {
		logrus.Fatalf("failed to query synonyms for owner=%v, synonym_name=%v: %v", dbUser, table.Name, err)
	}
	defer synRows.Close()
	if synRows.Next() {
		err = synRows.Scan(&table.Owner, &table.Name)
		if err != nil {
			logrus.Fatalf("failed to extract synonym information from query result: %v", err)
		}
		return
	}
	// check current user table names
	sqlQuery = `SELECT table_name FROM user_tables WHERE table_name = :tableName`
	tblRows, err := tx.Query(sqlQuery,
		sql.Named("tableName", table.Name),
	)
	if err != nil {
		logrus.Fatalf("failed to query synonyms for owner=%v, synonym_name=%v: %v", dbUser, table.Name, err)
	}
	defer tblRows.Close()
	if tblRows.Next() {
		table.Owner = dbUser
		return
	}
	logrus.Fatalf("Name %v is not found as synonym name or table name for user %v", table.Name, dbUser)
}

func ResolveDataType(name string, dataScale *int) dbmodel.DataType {
	switch {
	case name == "BLOB":
		return dbmodel.Blob_t
	case name == "CHAR":
		return dbmodel.Char_t
	case name == "CLOB":
		return dbmodel.Clob_t
	case name == "DATE":
		return dbmodel.Date_t
	case strings.HasPrefix(name, "TIMESTAMP"):
		return dbmodel.TimestampTz_t
	case name == "FLOAT":
		return dbmodel.Float_t
	case name == "LONG":
		return dbmodel.Long_t
	case name == "NUMBER":
		if dataScale != nil && *dataScale > 0 {
			return dbmodel.Float_t
		}
		return dbmodel.Number_t
	case name == "VARCHAR2" || name == "NVARCHAR2":
		return dbmodel.Varchar_t
	default:
		logrus.Fatalf("encountered unsupported database datatype: %v", name)
		return dbmodel.Varchar_t // this is for syntax check, we terminate before
	}
}

func resolveBoolValue(value string) bool {
	value = strings.ToUpper(value)
	switch {
	case value == "T" || value == "Y" || value == "YES":
		return true
	default:
		return false
	}
}

func getColumns(db *sql.Tx, table config.TableDef) []*dbmodel.DbColumnModel {
	sqlQuery := `
	select
		upper(column_name) as column_name,
		upper(data_type) as data_type,
		nullable,
		data_length,
		data_precision,
		data_scale,
		identity_column
	from
		all_tab_columns
	where upper(owner) = upper(:owner)
	and upper(table_name) = upper(:tableName)
	order by
		column_id asc
	`
	rows, err := db.Query(sqlQuery,
		sql.Named("owner", table.Owner),
		sql.Named("tableName", table.Name),
	)
	if err != nil {
		logrus.Fatalf("failed to run query to get columns of table %v.%v. Query: %v \nError: %v",
			table.Owner, table.Name, sqlQuery, err)
	}
	defer rows.Close()

	columns := []*dbmodel.DbColumnModel{}
	for rows.Next() {
		var colName, dataType, nullable, identity string
		var dataLength int
		var dataPrecision, dataScale *int
		err := rows.Scan(&colName, &dataType, &nullable, &dataLength, &dataPrecision, &dataScale, &identity)
		if err != nil {
			logrus.Fatalf("failed to extract column information for table %v.%v: %v", table.Owner, table.Name, err)
		}
		columns = append(columns, &dbmodel.DbColumnModel{
			Name:     strings.ToUpper(colName),
			Type:     ResolveDataType(dataType, dataScale),
			Nullable: resolveBoolValue(nullable),
			// Length:        dataLength,
			// Scale:         dataScale,
			// Precision:     dataPrecision,
			AutoGenerated: resolveBoolValue(identity),
		})
	}
	return columns
}

func getPrimaryKeyConstraint(
	tx *sql.Tx, table config.TableDef,
	columns map[string]*dbmodel.DbColumnModel,
	indexes map[string]*dbmodel.DbIndexModel) *dbmodel.DbConstraintModel {
	sqlQuery := `
		select 
			upper(cc.constraint_name),
			upper(cc.column_name),
			cc.position,
			upper(c.index_name)
		from 
			all_constraints c, all_cons_columns cc
		where upper(c.owner) = upper(:owner)
		  and upper(c.table_name) = upper(:tableName)
		  and c.constraint_type = 'P'
		  and c.constraint_name = cc.constraint_name
		  and c.owner = cc.owner
		  and c.table_name = cc.table_name
		order by cc.position asc
	`
	rows, err := tx.Query(sqlQuery,
		sql.Named("owner", table.Owner),
		sql.Named("tableName", table.Name),
	)
	if err != nil {
		logrus.Fatalf("failed to run query to get columns of table %v.%v. Query: %v \nError: %v",
			table.Owner, table.Name, sqlQuery, err)
	}
	defer rows.Close()

	var constraint *dbmodel.DbConstraintModel
	for rows.Next() {
		var constraintName, colName, indexName string
		var position int
		err := rows.Scan(&constraintName, &colName, &position, &indexName)
		if err != nil {
			logrus.Fatalf("failed to extract primary key constraint name for table %v.%v: %v",
				table.Owner, table.Name, err)
		}
		if constraint == nil {
			constraint = &dbmodel.DbConstraintModel{
				Name:  constraintName,
				Type:  dbmodel.PrimaryKey,
				Index: indexes[indexName],
			}
		}
		column, exists := columns[colName]
		if !exists {
			logrus.Fatalf("PK column %v for table %v.%v is not found", colName, table.Owner, table.Name)
		}
		column.IsPrimaryKey = true
		constraint.Columns = append(constraint.Columns, column)
	}
	return constraint
}

func getForeignKeyConstraint(tx *sql.Tx, table config.TableDef, columns map[string]*dbmodel.DbColumnModel, indexes map[string]*dbmodel.DbIndexModel) map[string]*dbmodel.DbConstraintModel {
	sqlQuery := `
		select upper(cc.column_name), cc.position, upper(cr.constraint_name), upper(cr.index_name), upper(cp.owner), upper(cp.table_name)
		from all_cons_columns cc, all_constraints cr, all_constraints cp
		where cc.owner = upper(:owner)
			and cc.table_name = upper(:tableName)
			and cc.owner = cr.owner
			and cc.constraint_name = cr.constraint_name
			and cc.table_name = cr.table_name
			and cr.constraint_type = 'R'
			and cr.r_owner = cp.owner
			and cr.r_constraint_name = cp.constraint_name
			and cp.constraint_type = 'P'
		order by cc.position asc
	`
	rows, err := tx.Query(sqlQuery,
		sql.Named("owner", table.Owner),
		sql.Named("tableName", table.Name),
	)
	if err != nil {
		logrus.Fatalf("failed to run query to get columns of table %v.%v. Query: %v \nError: %v",
			table.Owner, table.Name, sqlQuery, err)
	}
	defer rows.Close()

	constraints := map[string]*dbmodel.DbConstraintModel{}
	for rows.Next() {
		var constraintName, colName, indexName, pOwner, pTableName string
		var position int
		err := rows.Scan(&colName, &position, &constraintName, &indexName, &pOwner, &pTableName)
		if err != nil {
			logrus.Fatalf("failed to extract primary key constraint name for table %v.%v: %v",
				table.Owner, table.Name, err)
		}
		constraint, exists := constraints[constraintName]
		if !exists {
			referencedTable := fmt.Sprintf("%v.%v", pOwner, pTableName)
			constraint = &dbmodel.DbConstraintModel{
				Name:            constraintName,
				Type:            dbmodel.ForeignKey,
				Index:           indexes[indexName],
				ReferencedTable: &referencedTable,
			}
			constraints[constraintName] = constraint
		}
		column, exists := columns[colName]
		if !exists {
			logrus.Fatalf("PK column %v for table %v.%v is not found", colName, table.Owner, table.Name)
		}
		constraint.Columns = append(constraint.Columns, column)
	}
	return constraints
}

func getConstraints(tx *sql.Tx, table config.TableDef, columns map[string]*dbmodel.DbColumnModel, indexes map[string]*dbmodel.DbIndexModel) []*dbmodel.DbConstraintModel {
	// get primary key constraints
	pkConstraint := getPrimaryKeyConstraint(tx, table, columns, indexes)
	fkConstraints := getForeignKeyConstraint(tx, table, columns, indexes)
	constraints := []*dbmodel.DbConstraintModel{}

	if pkConstraint != nil {
		constraints = append(constraints, pkConstraint)
	}
	if len(fkConstraints) > 0 {
		for _, con := range fkConstraints {
			constraints = append(constraints, con)
		}
	}
	return constraints
}

func getIndexes(tx *sql.Tx, table config.TableDef, columns map[string]*dbmodel.DbColumnModel) map[string]*dbmodel.DbIndexModel {
	sqlQuery := `
		SELECT 
			upper(i.index_name), 
			upper(ic.column_name), 
			i.index_type, 
			DECODE(c.constraint_type, 'U', 'UNIQUE', 
									i.uniqueness) as uniqueness 
		FROM
			all_indexes     i, 
			all_ind_columns ic, 
			all_constraints c 
		WHERE 
			upper(i.owner) = upper(:owner) 
		AND 
			i.table_owner = i.owner 
		AND 
			i.table_owner = ic.table_owner
		AND 
			i.table_name = ic.table_name 
		AND 
			i.index_name = ic.index_name 
		AND 
			c.owner (+) = i.table_owner 
		AND 
			i.index_name = c.constraint_name (+)
		AND upper(i.table_name) = upper(:tableName)
		ORDER BY 
			i.index_name,  
			ic.column_position
	`
	rows, err := tx.Query(sqlQuery,
		sql.Named("owner", table.Owner),
		sql.Named("tableName", table.Name),
	)
	if err != nil {
		logrus.Fatalf("failed to run query to get indexes of table %v.%v. Query: %v \nError: %v",
			table.Owner, table.Name, sqlQuery, err)
	}
	defer rows.Close()
	indexes := map[string]*dbmodel.DbIndexModel{}
	for rows.Next() {
		var colName, indexName, indexType, unique string
		err := rows.Scan(&indexName, &colName, &indexType, &unique)
		if err != nil {
			logrus.Fatalf("failed to extract column information for table %v.%v: %v", table.Owner, table.Name, err)
		}
		column, exists := columns[colName]
		if !exists {
			logrus.Fatalf("Index column %v is not found in table %v.%v", colName, table.Owner, table.Name)
		}
		index, exists := indexes[indexName]
		if !exists {
			index = &dbmodel.DbIndexModel{
				Name:   indexName,
				Unique: unique == "UNIQUE",
			}
			indexes[indexName] = index
		}
		index.Columns = append(index.Columns, column)
	}
	return indexes
}

func GenerateTableModel(tx *sql.Tx, table config.TableDef) dbmodel.DbTableModel {
	table.Name = strings.ToUpper(table.Name) // fix table name to upper case
	resolveSynonym(tx, &table)
	tableModel := dbmodel.DbTableModel{
		Name: strings.ToUpper(table.Name),
	}
	tableModel.Columns = getColumns(tx, table)

	columnGenModelMap := map[string]*dbmodel.DbColumnModel{}
	for _, column := range tableModel.Columns {
		columnGenModelMap[column.Name] = column
	}
	indexes := getIndexes(tx, table, columnGenModelMap)
	for _, index := range indexes {
		tableModel.Indexes = append(tableModel.Indexes, index)
	}
	tableModel.Constraints = getConstraints(tx, table, columnGenModelMap, indexes)

	return tableModel
}

func GenerateTableModels(tx *sql.Tx, tables []config.TableDef) []dbmodel.DbTableModel {
	tableModels := []dbmodel.DbTableModel{}
	for index := range tables {
		table := tables[index]
		tableModel := GenerateTableModel(tx, table)
		tableModels = append(tableModels, tableModel)
	}
	return tableModels
}
