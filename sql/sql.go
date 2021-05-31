package sql

import (
	"strconv"
	"strings"

	"github.com/dtrehas/vrm"
)

//var queriesInStorage = goyesql.MustParseFile("queries.sql")
var queriesInMemory = make(map[string]string, 25)

const (
	ALL       = "ALL"
	AND       = "AND"
	AS        = "AS"
	ASC       = "ASC"
	AVERAGE   = "AVERAGE"
	COALESCE  = "COALESCE"
	COLUMN    = "COLUMN"
	CUBE      = "CUBE"
	DELETE    = "DELETE"
	DESC      = "DESC"
	EXCEPT    = "EXCEPT \n"
	FROM      = "\nFROM"
	FULL      = "FULL"
	GENERATED = "GENERATED"
	GROUP_BY  = "\nGROUP BY"
	GROUPING  = "\nGROUPING"
	HAVING    = "\nHAVING"
	INDENTITY = "INDENTITY"
	INNER     = "INNER"
	INTO      = "INTO"
	INSERT    = "INSERT"
	JOIN      = "JOIN"
	LEFT      = "LEFT"
	ON        = "ON"
	RENAME    = "RENAME"
	RIGHT     = "RIGHT"
	ROLLUP    = "ROLLUP"
	SELECT    = "\nSELECT"
	ORDER_BY  = "\nORDER BY"
	OUTER     = "OUTER"
	RETURNING = "RETURNING "
	STAR      = "*"
	SUM       = "SUM"
	UNION     = "UNION"
	UPDATE    = "UPDATE"
	USING     = "\nUSING"
	WITH      = "WITH"
	WHERE     = "\nWHERE"

	_                = ""
	EQUAL_TO         = ">"
	NOT_EQUAL        = "<>"
	GREATER_THAN     = ">"
	LESS_THAN        = "<"
	GREATER_OR_EQUAL = ">="
	LESS_OR_EQUAL    = "<="
	_                = ""
	ADD              = "ADD"
	ALTER            = "ALTER"
	ALWAYS           = "ALWAYS"
	BIGINT           = "BIGINT"
	BY_DEFAULT       = "BY DEFAULT"
	CASCADE          = "CASCADE"
	CREATE           = "CREATE"
	DATABASE         = "DATABASE"
	DROP             = "DROP"
	EXISTS           = "EXISTS"
	FOREIGN          = "FOREIGN"
	GRANT            = "GRANT"
	IF               = "IF"
	INT              = "INT"
	KEY              = "KEY"
	NOT              = "NOT"
	NULL             = "NULL"
	OVERRIDING       = "OVERRIDING"
	PRIMARY          = "PRIMARY"
	RENAME_TO        = "RENAME TO "
	REFERENCES       = "REFERENCES"
	ROLE             = "ROLE"
	SEQUENCE         = "SEQUENCE"
	SMALLINT         = "SMALLINT"
	SYSTEM           = "SYSTEM"
	TABLE            = "TABLE"
	TEMPORARY        = "TEMPORARY"
	UNIQUE           = "UNIQUE"
	USER             = "USER"
	VARCHAR          = "VARCHAR"
	VIEW             = "VIEW"
)

func COLUMNS(ofs ...interface{}) string {

	var b strings.Builder
	b.WriteString("(")

	for j, of := range ofs {
		var cs vrm.Columns

		switch of.(type) {

		case vrm.ColumnValues:
			cs = of.(vrm.ColumnValues).Columns
			//case Table:
			//	cs = of.(Table).ColumnSet()

		case vrm.Columns, *vrm.Columns:
			cs = of.(vrm.Columns)

		}

		size := len(cs)
		if size == 0 {
			continue
		}
		//we separate with a ',' all 'ofs' set. except if there is only one element.
		if j != 0 {
			b.WriteRune(',')
		}

		for i, c := range cs {
			b.WriteString(c.Name)
			if i < size-1 {
				b.WriteString(",")
			}
		}
	}
	b.WriteString(")")
	return b.String()
}

func VALUES(ofs ...interface{}) string {

	var b strings.Builder
	b.WriteString("\nVALUES (")

	i := 1
	for j, of := range ofs {
		var cs vrm.Columns
		switch of.(type) {
		//case Table:
		//	cs = of.(Table).ColumnSet()

		case *vrm.ColumnValues:
			cs = of.(*vrm.ColumnValues).Columns
		case vrm.ColumnValues:
			cs = of.(vrm.ColumnValues).Columns

		case vrm.Columns, *vrm.Columns:
			cs = of.(vrm.Columns)
		}

		size := len(cs)
		if size == 0 {
			continue
		}

		//we separate with a ',' all 'ofs' set. except if there is only one element.
		if j != 0 {
			b.WriteRune(',')
		}

		for k := 1; k <= size; k++ {
			b.WriteRune('$')
			b.WriteString(strconv.Itoa(i))
			i++
			if k < size {
				b.WriteString(",")
			}
		}
	}
	b.WriteString(")")
	return b.String()
}

//Sql builds db queries using fluent API
func Sql(args ...interface{}) string {
	var b strings.Builder

	for _, arg := range args {
		var text string
		switch arg.(type) {

		case vrm.Stringer:
			text = arg.(vrm.Stringer).String()

		case string:
			text = arg.(string)

		case vrm.Column:
			text = arg.(vrm.Column).Name

		case vrm.Tabler:
			text = arg.(vrm.Tabler).String()
		}

		b.WriteString(text)
		b.WriteRune(' ')

	}

	b.WriteString(";\n")
	return b.String()

}

//NamedSql caches sql queries for better performance
func NamedSql(args ...interface{}) string {
	if len(args) < 2 {
		return ""
	}

	name := args[0].(string)
	sql, ok := queriesInMemory[name]
	if ok {
		return sql
	}

	sql = Sql(args[1:]...)
	queriesInMemory[name] = sql

	return sql
}
