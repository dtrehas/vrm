package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/dtrehas/vrm"
	"github.com/jackc/pgtype"

	toml "github.com/pelletier/go-toml"
)

type TableInf struct {
	Name        string
	Columns     []*vrm.Column
	Constraints []*ConstraintInf
}

type ConstraintInf struct {
	Name    string
	Type    string
	Columns []*vrm.Column
}

func initArguments() {
	textPtr := flag.String("text", "", "Text to parse. (Required)")
	metricPtr := flag.String("metric", "chars", "Metric {chars|words|lines};. (Required)")
	uniquePtr := flag.Bool("unique", false, "Measure unique values of a metric.")
	flag.Parse()

	if *textPtr == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	fmt.Printf("textPtr: %s, metricPtr: %s, uniquePtr: %t\n", *textPtr, *metricPtr, *uniquePtr)
}

func printArgHelp() {

	_, file := path.Split(os.Args[0])

	fmt.Printf("Usage: %v section\n", file)
	fmt.Println("		section: settings in {section} in this directory's vrm.toml ")
	fmt.Println("")
	fmt.Printf("Usage: %v section [-c path to config %v.toml file]\n", file, file)
}

func genVrmSql(s string) {

	//fields := strings.Fields(s)

}

func main() {

	var conf Config

	//Add congifuration options from extra program arguments like --wipe etc.
	parseArgs(&conf)

	if conf.sql != "" {

	}

	//+Retrieve the configuration profile from {executable-name}.toml
	//+named by the first argument after binary name

	if conf.name == "" {
		log.Fatalln("No config name is given in arguments e.g. clientsDB")
	}

	//Load conf configuration struct with data from the configuration profile
	processConfigFile(&conf)

	//If Wiping option is selected, delete the dir and subdirs of
	//ModelsPath
	if conf.Wipe {
		if err := os.RemoveAll(conf.ModelsPath); err != nil {
			log.Fatalln(err)
		}
	}

	ctx := context.Background()

	//Open the database to retrieve the database structure
	conn := vrm.Open(ctx, &conf.Db)
	if conn == nil {
		log.Fatal("No DB connection, ")
	}
	defer conn.Close(ctx)

	data := &Data{}

	retrieveTablesConfig(ctx, conn, &conf.Db, data)
	retrieveConstraints(ctx, conn, &conf.Db, data)
	createModelsDirectory(&conf)
	createPackageFile(&conf, data)

	fmt.Println("Table, Columns, constraints are retrieved successfuly")

	for _, table := range data.Tables {
		createTableFile(&conf, data, table.Name)
		fmt.Printf("%s table is created\n", table.Name)
	}
}

//Config
type Config struct {
	AutoTimestamps bool
	NoJSONTag      bool
	NoXMLTag       bool
	NoDbTag        bool
	AddBatch       bool
	Db             vrm.DbConfig
	ModelsPath     string
	Version        string
	Wipe           bool
	sql            string
	name           string
	configFilePath string
	execPath       string
}

func (c *Config) ParseToml() { //configName, file string) {

	tml, err := toml.LoadFile(c.configFilePath)
	if err != nil {
		log.Fatalln("No config file name in %v", c.configFilePath)
	}

	if !tml.Has(c.name) {
		log.Fatalf("No %v section found in %v", c.name, c.configFilePath)
	}

	s := tml.Get(c.name).(*toml.Tree)

	c.AutoTimestamps = s.GetDefault("auto-timestamp", true).(bool)
	c.AddBatch = s.GetDefault("add-batch-operations", false).(bool)

	//Output path
	p := s.Get("models-path")
	if p == nil {
		log.Fatalln("No output path for models is given. Exiting.")
	}
	c.ModelsPath = strings.TrimSpace(p.(string))

	c.Db.Host = s.GetDefault("host", "localhost").(string)
	c.Db.Port = int(s.GetDefault("port", "5432").(int64))
	c.Db.Schema = s.GetDefault("schema", "public").(string)

	//Database name
	p = s.Get("dbname")
	if p == nil {
		log.Fatalln("No database name is given. Exiting.")
	}
	c.Db.Database = p.(string)

	//User name
	p = s.Get("user")
	if p == nil {
		log.Fatalln("No user name is given. Exiting.")
	}
	c.Db.User = p.(string)

	c.Db.Password = s.GetDefault("password", "").(string)
	//Password
	p = s.Get("password")
	if p == nil {
		log.Fatalln("No password is given. Exiting.")
	}
	c.Db.Password = p.(string)

	c.Wipe = s.GetDefault("wipe", false).(bool)
}

func parseArgs(conf *Config) {
	// vrm-gen called without args
	if len(os.Args) == 1 {
		printArgHelp()
		os.Exit(1)
	}

	//vrm-gen called with args
	args := os.Args[1:]
	conf.execPath = os.Args[0]

	isInputPath := false

	for i, arg := range args {

		if isInputPath {
			conf.configFilePath = arg
			isInputPath = false
			continue
		}

		switch arg {
		case "--wipe":
			conf.Wipe = true
		case "--auto-timestamp":
			conf.AutoTimestamps = true
		case "-c":
			isInputPath = true
		case "-sql":
			if i < len(args)-1 {
				conf.sql = args[i+1]
			}

		default:
			conf.name = arg
		}
	}
}

//processConfigFile expects the vrm.toml configurations file
//in the working directory in order vrm-gen to find it
func processConfigFile(conf *Config) {

	if conf.configFilePath == "" {
		pwd, _ := os.Getwd()

		execPath, _ := os.Executable()
		exec := path.Base(path.Clean(execPath))
		conf.configFilePath = path.Join(pwd, exec+".toml")
	}

	conf.ParseToml()
}

type ConstraintMap map[string]*ConstraintInf

var Constraints = ConstraintMap{}

type TableMap map[string]*TableInf

var Tables = TableMap{}

type Data struct {
	Tables []TableInf
}

var fm = template.FuncMap{
	"escapeIfReservedWord":          EscapeIfReservedWord,
	"camelCase":                     CamelTheSnake,
	"escapedColumns":                EscapedColumns,
	"escapedColumnsAndVariableNums": EscapedColumnsAndVariableNums,
	"primaryKeyOrUniqueConstraints": PrimaryKeyOrUniqueConstraints,
	"noPrimaryKeyNorUniqueColumns":  NoPrimaryKeyNorUniqueColumns,
	"lessThanSizeMinusOne": func(i int, columns []vrm.Column) bool {
		return i < len(columns)-1
	},
	"nextInt": func(i int) int {
		return i + 1
	},
	"dateTimeFound":      DateTimeFound,
	"serialId":           SerialId,
	"nonSerialIdColumns": NonSerialIdColumns,
	"uuidFound":          UuidFound,
	"variableName":       VariableName,
	"variableNames":      VariableNames,
	"variableAndType":    VariableAndType,
	"variablesAndTypes":  VariablesAndTypes,
	"variableDollarNums": VariableDollarNums,
}

func createModelsDirectory(conf *Config) error {
	modelsPath := strings.TrimSpace(conf.ModelsPath)
	if err := os.Mkdir(modelsPath, 0777); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func createPackageFile(conf *Config, data *Data) {

	modelsPath := strings.TrimSpace(conf.ModelsPath)
	packageName := path.Base(modelsPath)
	filePath := path.Join(modelsPath, packageName+"-package.go")

	file, err := os.Create(path.Clean(filePath))
	if err != nil {
		log.Println(err)
	}

	tpl, err := template.New("package_template").Funcs(fm).Parse(Package_Tpl_String)
	if err != nil {
		log.Fatalln(err)
	}

	tpl.Execute(file, map[string]interface{}{
		"Package": packageName,
		"Tables":  data.Tables,
	})

	file.Close()

}

func genPackage(variables Variables, conf *Config, file *os.File) {
	file.WriteString(`package ` + variables["Package"].(string) + "\n")
}

func genImports(variables Variables, conf *Config, file *os.File) {

	columns := variables["Columns"].([]*vrm.Column)

	file.WriteString(
		`import (
	"context"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/dtrehas/vrm"` + "\n")

	// if conf.AddBatch {
	// 	file.WriteString(`		"github.com/jackc/pgx/v4"` + "\n")
	// }

	// if DateTimeFound(columns) {
	// 	file.WriteString(`	"time"` + "\n")
	// }

	file.WriteString(`	"time"` + "\n")

	if UuidFound(columns) {
		file.WriteString(`	uuidp "github.com/gofrs/uuid"` + "\n")
	}

	file.WriteString(")\n")
}

type Variables map[string]interface{}

//genStruct writes variable/type definition to the file, for a specific table
//
//
//
func genStruct(variables Variables, conf *Config, file *os.File) {
	d := &Identer{}

	d.S("type " + variables["CamelCaseTable"].(string) + " struct {").Nl()

	//every variable/type combination is seperatate by newline and some indent
	structFields := VariablesTypesAndTags("\n\t\t", variables["Columns"].([]*vrm.Column), conf)
	d.S("\t\t")
	d.S(structFields)
	d.S("\n}\n")

	file.WriteString(d.String())

}

func genInsertStatements(variables Variables, conf *Config, file *os.File) {
	d := &Identer{}

	columns := variables["Columns"].([]*vrm.Column)
	nonSerialIdColumns := NonSerialIdColumns(columns)
	variablesAndTypes := VariablesAndTypes(",\n\t\t", nonSerialIdColumns)
	camelCaseTable := variables["CamelCaseTable"].(string)
	table := variables["Table"].(string)
	serialId := SerialId(columns)

	d.S("func Insert").S(camelCaseTable).S(" (ctx context.Context, conn vrm.Quexecer, ").S(variablesAndTypes).R(')')

	if serialId != nil {
		d.R('(').S(VariableAndType(serialId)).S(", err error) {").MoreIdent(Tab).Nl()
		d.S("sql := `INSERT INTO ").
			S(table).R(' ').S(EscapedColumns(",", nonSerialIdColumns)).
			S(" VALUES (").S(VariableDollarNums(",", nonSerialIdColumns)).
			S(") RETURNING ").S(serialId.Name).S(";`").Nl(2)

		d.S("row := conn.QueryRow(ctx, sql, " + VariableNames(",\n\t\t", nonSerialIdColumns)).S(")").Nl()

		d.S("err = row.Scan(&" + VariableName(serialId.Name)).S(")").Nl(2)
		d.S("if err!=nil {").MoreIdent(Tab).Nl()
		d.S("return -1, err").LessIdent().Nl()
		d.S("}").LessIdent().Nl(2)
	} else {
		d.S("(tag pgconn.CommandTag, err error) {").MoreIdent(Tab).Nl().
			S("sql := `INSERT INTO ").S(table).S(" ").S(EscapedColumns(",", columns)).S(" VALUES (").S(VariableDollarNums(",", columns)).S(");`").Nl(2).
			S("tag, err = conn.Exec(ctx, sql, ").S(VariableNames(",\n\t\t", columns)).S(")").Nl(2)
	}
	d.S("return").LessIdent().Nl().
		S("}").
		Nl()

	file.WriteString(d.String())
}

func genBatchInsertStatements(variables Variables, conf *Config, file *os.File) {
	d := &Identer{}

	columns := variables["Columns"].([]*vrm.Column)
	nonSerialIdColumns := NonSerialIdColumns(columns)
	variablesAndTypes := VariablesAndTypes(",\n\t\t", nonSerialIdColumns)
	camelCaseTable := variables["CamelCaseTable"].(string)
	table := variables["Table"].(string)
	serialId := SerialId(columns)

	d.S("func BatchInsert").S(camelCaseTable).S(" (b* pgx.Batch, ").S(variablesAndTypes).R(')').
		S("{").MoreIdent(Tab).Nl()

	if serialId != nil {

		d.S("sql := `INSERT INTO ").S(table).S(" ").S(EscapedColumns(",", nonSerialIdColumns)).
			S(" VALUES (").S(VariableDollarNums(",", nonSerialIdColumns)).S(") RETURNING ").S(serialId.Name).S(";`").Nl(2)

		d.S("b.Queue (sql, ").S(VariableNames(", ", nonSerialIdColumns)).R(')')

	} else {
		d.S("sql := `INSERT INTO ").S(table).R(' ').S(EscapedColumns(",", columns)).S(" VALUES (").S(VariableDollarNums(",", columns)).S(");`").Nl(2).
			S("b.Queue (sql, ").S(VariableNames(", ", columns)).R(')')
	}

	d.LessIdent().Nl()
	d.S("}").LessIdent().Nl(2)
	file.WriteString(d.String())
}

func genStructBatchInsertStatements(variables Variables, conf *Config, file *os.File) {
	d := &Identer{}

	columns := variables["Columns"].([]*vrm.Column)
	nonSerialIdColumns := NonSerialIdColumns(columns)

	camelCaseTable := variables["CamelCaseTable"].(string)
	//serialId := SerialId(columns)

	d.S("func (t *").S(camelCaseTable).S(") BatchInsert(b *pgx.Batch) {").MoreIdent(Tab).Nl()
	d.S("BatchInsert").S(camelCaseTable).S("(b, "). //t.DocumentId, t.SubjectCodeId).LessIndent().Nl()
							S(VariableNames(",", nonSerialIdColumns, "t.", "")). //BatchInsertDocumentSubjectCode(b, t.DocumentId, t.SubjectCodeId)
							R(')').LessIdent().Nl()
	d.R('}').
		Nl()

	file.WriteString(d.String())

}

func genStructInsertStatements(variables Variables, conf *Config, file *os.File) {
	d := &Identer{}

	columns := variables["Columns"].([]*vrm.Column)
	nonSerialIdColumns := NonSerialIdColumns(columns)

	camelCaseTable := variables["CamelCaseTable"].(string)
	serialId := SerialId(columns)

	d.S("func (t *").S(camelCaseTable).S(") Insert(ctx context.Context, conn vrm.Quexecer) (").MoreIdent(Tab).Nl()
	if serialId == nil {
		d.S("tag pgconn.CommandTag")
	} else {
		d.S(VariableAndType(serialId))
	}
	d.S(" ,err error) {").MoreIdent(Tab).Nl()

	d.S("return Insert").S(camelCaseTable).S(" (ctx, conn, ").S(VariableNames(",", nonSerialIdColumns, "t.", "")).R(')').LessIdent().Nl()
	d.R('}').
		Nl()

	file.WriteString(d.String())
}

func genUpdateStatements(variables Variables, conf *Config, file *os.File) {
	d := &Identer{}

	columns := variables["Columns"].([]*vrm.Column)
	constraints := variables["Constraints"].([]*ConstraintInf)
	//nonSerialIdColumns := NonSerialIdColumns(columns)

	camelCaseTable := variables["CamelCaseTable"].(string)
	table := variables["Table"].(string)
	//serialId := SerialId(columns)

	// {{ if $PrimaryKeyOrUniqueConstraints -}}
	primaryKeyOrUniqueConstraints := PrimaryKeyOrUniqueConstraints(constraints)

	if primaryKeyOrUniqueConstraints != nil {
		for _, constraint := range primaryKeyOrUniqueConstraints {

			// func Update{{$.CamelCaseTable }}By{{ range .Columns}}{{ variableName .Name}}{{ end }} (ctx context.Context, conn vrm.Quexecer,
			// 	{{- variablesAndTypes ",\n\t\t" .Columns}}, {{ variablesAndTypes ",\n\t\t" $.Columns "New"}}) (pgconn.CommandTag, error){

			funcArgs := VariableNames(",", constraint.Columns)
			funcName := "func Update" + camelCaseTable + "By" + VariableNames("", constraint.Columns)

			d.S("//").S(funcName).S(" updates ").S(table).S(" using ").S(funcArgs).Nl()
			d.S(funcName).S(" (ctx context.Context, conn vrm.Quexecer,").
				S(VariablesAndTypes(",\n\t\t", constraint.Columns)).S(", ").S(VariablesAndTypes(",\n\t\t", columns, "New")).S(") (pgconn.CommandTag, error){").
				MoreIdent(Tab).Nl()

			// 	sql:= {{ $tick }}UPDATE {{ $.Table }} SET
			d.S("sql:= `UPDATE ").S(table).S(" SET").MoreIdent(Tab).Nl().
				// 	{{ escapedColumnsAndVariableNums ", " $.Columns}}
				S(EscapedColumnsAndVariableNums(", ", columns)).LessIdent().Nl().
				// 	WHERE
				S("WHERE").MoreIdent(Tab).Nl().
				// 		{{ escapedColumnsAndVariableNums " AND \n\t\t" .Columns}}); `
				S(EscapedColumnsAndVariableNums(" AND \n\t\t", constraint.Columns)).S("); `").LessIdent().Nl(2).

				// 	return conn.Exec(ctx, sql, {{ variableNames ", " .Columns}},{{ variableNames ", " $.Columns "New"}})
				S("return conn.Exec(ctx, sql, ").S(VariableNames(", ", constraint.Columns)).S(", ").S(VariableNames(", ", columns, "New")).R(')').LessIdent().Nl().
				R('}')
			// }

		}

	} else {

		startWithId := len(columns)

		// {{- $StartWithId:= len $.Columns -}}

		// func Update{{ .CamelCaseTable }} (ctx context.Context, conn vrm.Quexecer,
		// 		{{ variablesAndTypes ",\n\t\t" .Columns }}, {{ variablesAndTypes ",\n\t\t" .Columns "New"}}) (pgconn.CommandTag, error){

		funcName := "func Update" + camelCaseTable
		funcArgs := VariablesAndTypes(", ", columns)
		d.S("//").S(funcName).S(" updates ").S(table).S(" using ").S(funcArgs).Nl()

		d.S(funcName).S("(ctx context.Context, conn vrm.Quexecer,").Nl().
			S(VariablesAndTypes(",\n\t\t", columns)).S(", ").S(VariablesAndTypes(",\n\t\t", columns, "New")).S(") (pgconn.CommandTag, error){").
			MoreIdent(Tab).Nl().
			S("sql:= `UPDATE ").S(table).S(" SET").MoreIdent(Tab).Nl().
			S(EscapedColumnsAndVariableNums(", ", columns)).LessIdent().Nl().
			S("WHERE").MoreIdent(Tab).Nl().
			S(EscapedColumnsAndVariableNums(" AND \n\t\t", columns, startWithId)).S("); `").LessIdent().Nl(2).
			S("return conn.Exec(ctx, sql, ").S(VariableNames(", ", columns)).S(", ").S(VariableNames(", ", columns, "New")).R(')').
			LessIdent().Nl().
			R('}')
	}

	file.WriteString(d.String())
}

//createTableFile writes a file, with all necessary function to access/update/insert functions
// a database table, defined by conf config
func createTableFile(conf *Config, data *Data, tableName string) error {

	tablePath := path.Join(conf.ModelsPath, tableName+".go")
	file, err := os.Create(tablePath)

	Package := path.Base(conf.ModelsPath)

	variables := map[string]interface{}{
		"CamelCaseTable": CamelTheSnake(tableName),
		"Columns":        Tables[tableName].Columns,
		"Constraints":    Tables[tableName].Constraints,
		"Package":        Package,
		"Table":          tableName,
		"UpperCaseTable": strings.ToUpper(tableName),
		"Version":        conf.Version,
		"Conf":           conf,
	}

	genPackage(variables, conf, file)
	genImports(variables, conf, file)
	genStruct(variables, conf, file)
	genInsertStatements(variables, conf, file)
	genBatchInsertStatements(variables, conf, file)
	genStructInsertStatements(variables, conf, file)
	genStructBatchInsertStatements(variables, conf, file)
	genUpdateStatements(variables, conf, file)

	tpl, err := template.New("table_template").Funcs(fm).Parse(Table_tpl_string)
	if err != nil {
		log.Fatal(err)
	}

	if err := tpl.Execute(file, variables); err != nil {
		log.Fatal(err)
	}
	return file.Close()
}

func GetTableNames(ctx context.Context, conn vrm.Quexecer, conf *vrm.DbConfig) []string {
	sql := `select table_name from information_schema.tables where table_catalog='` + conf.Database + `' and table_schema='` + conf.Schema + `';`
	var table string
	var tables []string
	rows, err := conn.Query(ctx, sql)
	defer rows.Close()

	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		rows.Scan(&table)
		//k,_:= rows.Values()
		//log.Println(k)
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return tables
}

func retrieveConstraints(ctx context.Context, conn vrm.Quexecer, conf *vrm.DbConfig, data *Data) {
	//retrieve PRIMARY KEY, FOREIGN KEY, UNIQUE constraint names and types for all GetTableNames
	sql := `SELECT table_name, constraint_name, constraint_type from information_schema.table_constraints where table_catalog='` + conf.Database + `' AND table_schema='` + conf.Schema + `' and constraint_type IN('PRIMARY KEY', 'FOREIGN KEY', 'UNIQUE') order by table_name;`

	rows, err := conn.Query(ctx, sql)

	if err != nil {
		log.Fatalln(err)
	}
	defer rows.Close()

	for rows.Next() {

		var table_name, constraint_name, constraint_type string

		err := rows.Scan(&table_name,
			&constraint_name,
			&constraint_type)

		if err != nil {
			log.Fatal(err)
		}

		for i, _ := range data.Tables {
			table := &data.Tables[i]
			if table.Name == table_name {

				constraint := ConstraintInf{
					Name:    constraint_name,
					Type:    constraint_type,
					Columns: []*vrm.Column{},
				}

				table.Constraints = append(table.Constraints, &constraint)
				Constraints[constraint_name] = &constraint
			}
		}
	}

	//retrieve the column names of constraints and match with the existing constraints
	sql = `select table_name, constraint_name, column_name, ordinal_position from information_schema.key_column_usage where table_catalog='` + conf.Database + `' AND table_schema='` + conf.Schema + `' order by table_name, constraint_name, ordinal_position`
	rows, err = conn.Query(ctx, sql)

	if err != nil {
		log.Fatalln(err)
	}
	defer rows.Close()

	for rows.Next() {

		var table_name, constraint_name, column_name string
		var ordinal_position int

		err := rows.Scan(
			&table_name,
			&constraint_name,
			&column_name,
			&ordinal_position)

		if err != nil {
			log.Fatal(err)
		}

		table := Tables[table_name]
		constraint := Constraints[constraint_name]

		for i, _ := range table.Columns {
			column := table.Columns[i]
			if column.Name == column_name {
				constraint.Columns = append(constraint.Columns, column)
				switch constraint.Type {

				case "PRIMARY KEY":
					column.Key = true
					column.PartialKey = true
				case "UNIQUE":
					column.Unique = true

				case "FOREIGN KEY":
				}

			}
		}
	}
}

func retrieveTablesConfig(ctx context.Context, conn vrm.Quexecer, conf *vrm.DbConfig, data *Data) {
	tableNames := GetTableNames(ctx, conn, conf)
	data.Tables = make([]TableInf, len(tableNames))

	for i, tableName := range tableNames {

		data.Tables[i].Name = tableName
		Tables[tableName] = &data.Tables[i]
		_retrieveTableConfiguration(ctx, conn, conf, data, i)
	}
}

func _retrieveTableConfiguration(ctx context.Context, conn vrm.Quexecer, conf *vrm.DbConfig, data *Data, index int) {

	table := &data.Tables[index]

	sql := `select column_name, ordinal_position, column_default, is_nullable, data_type,udt_name from information_schema.columns where table_name='` + table.Name + `' and table_catalog='` + conf.Database + `' and table_schema='` + conf.Schema + `';`
	rows, err := conn.Query(ctx, sql)

	if err != nil {
		log.Fatalln(err)
	}
	defer rows.Close()

	for rows.Next() {
		var column_name string
		var ordinal_position int
		var column_default pgtype.Text
		var is_nullable string
		var data_type string
		var udt_name string

		err := rows.Scan(&column_name,
			&ordinal_position,
			&column_default,
			&is_nullable,
			&data_type,
			&udt_name)

		if err != nil {
			log.Fatal(err)
		}

		col := &vrm.Column{
			Array:         strings.Index(udt_name, "_") == 0,
			Name:          column_name,
			GoType:        GoType(udt_name),
			Position:      ordinal_position,
			Key:           false,
			PartialKey:    false,
			NotInsertable: false,
			Nullable:      is_nullable == "YES",
			Serial: (column_default.Status == pgtype.Present) &&
				(strings.Index(column_default.String, "_seq") != -1),
			Type:   udt_name,
			Unique: false,
		}
		table.Columns = append(table.Columns, col)
	}
}
