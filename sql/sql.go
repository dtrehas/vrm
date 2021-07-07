package sql

import (
	"strconv"
	"strings"

	"github.com/dtrehas/vrm"
)

//var queriesInStorage = goyesql.MustParseFile("queries.sql")
var queriesInMemory = make(map[string]string, 25)

type Token string

const (
	ALL       Token = "ALL"
	AND       Token = "AND"
	AS        Token = "AS"
	ASC       Token = "ASC"
	AVERAGE   Token = "AVERAGE"
	COALESCE  Token = "COALESCE"
	COLUMN    Token = "COLUMN"
	CUBE      Token = "CUBE"
	DELETE    Token = "DELETE"
	DESC      Token = "DESC"
	EXCEPT    Token = "EXCEPT \n"
	FROM      Token = "\nFROM"
	FULL      Token = "FULL"
	GENERATED Token = "GENERATED"
	GROUP_BY  Token = "\nGROUP BY"
	GROUPING  Token = "\nGROUPING"
	HAVING    Token = "\nHAVING"
	IDENTITY  Token = "IDENTITY"
	INNER     Token = "INNER"
	INTO      Token = "INTO"
	INSERT    Token = "INSERT"
	JOIN      Token = "JOIN"
	LEFT      Token = "LEFT"
	ON        Token = "ON"
	RENAME    Token = "RENAME"
	RIGHT     Token = "RIGHT"
	ROLLUP    Token = "ROLLUP"
	SELECT    Token = "\nSELECT"
	ORDER_BY  Token = "\nORDER BY"
	OUTER     Token = "OUTER"
	RETURNING Token = "RETURNING "
	STAR      Token = "*"
	SUM       Token = "SUM"
	UNION     Token = "UNION"
	UPDATE    Token = "UPDATE"
	USING     Token = "\nUSING"
	WITH      Token = "WITH"
	WHERE     Token = "\nWHERE"

	_                      = ""
	EQUAL_TO         Token = ">"
	NOT_EQUAL        Token = "<>"
	GREATER_THAN     Token = ">"
	LESS_THAN        Token = "<"
	GREATER_OR_EQUAL Token = ">="
	LESS_OR_EQUAL    Token = "<="
	_                Token = ""
	ADD              Token = "ADD"
	ALTER            Token = "ALTER"
	ALWAYS           Token = "ALWAYS"
	BIGINT           Token = "BIGINT"
	BY_DEFAULT       Token = "BY DEFAULT"
	CASCADE          Token = "CASCADE"
	CREATE           Token = "CREATE"
	DATABASE         Token = "DATABASE"
	DROP             Token = "DROP"
	EXISTS           Token = "EXISTS"
	FOREIGN          Token = "FOREIGN"
	GRANT            Token = "GRANT"
	IF               Token = "IF"
	INT              Token = "INT"
	KEY              Token = "KEY"
	NOT              Token = "NOT"
	NULL             Token = "NULL"
	OVERRIDING       Token = "OVERRIDING"
	PRIMARY          Token = "PRIMARY"
	RENAME_TO        Token = "RENAME TO "
	REFERENCES       Token = "REFERENCES"
	ROLE             Token = "ROLE"
	SEQUENCE         Token = "SEQUENCE"
	SMALLINT         Token = "SMALLINT"
	SYSTEM           Token = "SYSTEM"
	TABLE            Token = "TABLE"
	TEMPORARY        Token = "TEMPORARY"
	UNIQUE           Token = "UNIQUE"
	USER             Token = "USER"
	VARCHAR          Token = "VARCHAR"
	VIEW             Token = "VIEW"
)

var TokenMap = map[string]Token{
	"ALL": ALL, "AND": AND, "AS": AS, "ASC": ASC, "AVERAGE": AVERAGE, "COALESCE": COALESCE, "COLUMN": COLUMN, "CUBE": CUBE,
	"DELETE": DELETE, "DESC": DESC, "EXCEPT": EXCEPT, "FROM": FROM, "FULL": FULL, "GENERATED": GENERATED, "GROUP BY": GROUP_BY, "GROUPING": GROUPING,
	"HAVING": HAVING, "IDENTITY": IDENTITY, "INNER": INNER, "INTO": INTO, "INSERT": INSERT, "JOIN": JOIN, "LEFT": LEFT, "ON": ON, "RENAME": RENAME,
	"RIGHT": RIGHT, "ROLLUP": ROLLUP, "SELECT": SELECT, "ORDER BY": ORDER_BY, "OUTER": OUTER, "RETURNING": RETURNING, "*": "STAR", "SUM": SUM,
	"UNION": UNION, "UPDATE": UPDATE, "USING": USING, "WITH": WITH, "WHERE": WHERE, "=": EQUAL_TO, "<>": NOT_EQUAL,
	">": GREATER_THAN, "<": LESS_THAN, ">=": GREATER_OR_EQUAL, "<=": LESS_OR_EQUAL, "ADD": ADD,
	"ALTER": ALTER, "ALWAYS": ALWAYS, "BIGINT": BIGINT, "BY DEFAULT": BY_DEFAULT, "CASCADE": CASCADE, "CREATE": CREATE, "DATABASE": DATABASE, "DROP": DROP,
	"EXISTS": EXISTS, "FOREIGN": FOREIGN, "GRANT": GRANT, "IF": IF, "INT": INT, "KEY": KEY, "NOT": NOT, "NULL": NULL, "OVERRIDING": OVERRIDING,
	"PRIMARY": PRIMARY, "RENAME TO": RENAME_TO, "REFERENCES": REFERENCES, "ROLE": ROLE, "SEQUENCE": SEQUENCE, "SMALLINT": SMALLINT, "SYSTEM": SYSTEM,
	"TABLE": TABLE, "TEMPORARY": TEMPORARY, "UNIQUE": UNIQUE, "USER": USER, "VARCHAR": VARCHAR, "VIEW": VIEW,
}

func COLUMNS(ofs ...interface{}) string {

	var b strings.Builder
	b.WriteString("(")

	for j, of := range ofs {
		var cs vrm.Columns

		switch of.(type) {

		case vrm.ColumnValues:
			cs = of.(vrm.ColumnValues).Columns
		case vrm.Table, *vrm.Table:
			cs = vrm.ColumnsOf(of.(vrm.Table))
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
		case vrm.Table, *vrm.Table: //TODO add Table
			cs = vrm.ColumnsOf(of.(vrm.Table))

		case *vrm.ColumnValues, vrm.ColumnValues:
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

		case string:
			text = arg.(string)

		case vrm.Column, *vrm.Column:

			text = arg.(vrm.Column).Name
			col := arg.(vrm.Column)
			if **col.IsWriteTable {
				b.WriteRune('"')
				b.WriteString(col.Table.Name_())
				b.WriteString(`"."`)
				b.WriteString(text)
				b.WriteRune('"')
			} else {
				b.WriteRune('"')
				b.WriteString(text)
				b.WriteRune('"')
			}

		case vrm.Table, *vrm.Table:
			table := arg.(vrm.Table)

			if table.WriteSchema_() {
				b.WriteRune('"')
				b.WriteString(table.Schema_())
				b.WriteString(`"."`)
				b.WriteString(table.Name_())
				b.WriteRune('"')
			} else {
				b.WriteRune('"')
				b.WriteString(table.Name_())
				b.WriteRune('"')
			}
		case vrm.Stringer:
			text = arg.(vrm.Stringer).String()
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
