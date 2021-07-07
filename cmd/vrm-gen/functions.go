package main

import (
	"log"
	"strconv"
	"strings"

	"github.com/dtrehas/vrm"
)

const Tab = "\t"
const Tab2 = "\t\t"

func CamelTheSnake(str string) string {
	stringz := strings.Split(str, "_")

	for i, z := range stringz {
		stringz[i] = strings.Title(z)
	}
	return strings.Join(stringz, "")
}

func SmallCamelTheCase(str string) string {
	stringz := strings.Split(str, "_")

	if len(stringz) == 0 {

	}
	for i := 1; i < len(stringz); i++ {
		stringz[i] = strings.Title(stringz[i])
	}

	stringz[0] = strings.ToLower(stringz[0])
	return strings.Join(stringz, "")
}

func GoType(udt_name string) string {

	isArray := strings.Index(udt_name, "_") == 0
	if isArray {
		udt_name = udt_name[1:]
	}
	var goType string

	switch udt_name {
	case "int2":
		goType = "int16"
	case "int4":
		goType = "int32"
	case "int8":
		goType = "int64"
	case "bpchar", "text", "varchar":
		goType = "string"
	case "date", "timestamp":
		goType = "time.Time"
	case "decimal":
		goType = "float64" //TODO
	case "float":
		goType = "float32" //TODO
	case "uuid":
		goType = "uuidp.UUID"
	case "bool":
		goType = "bool"
	case "bytea":
		goType = "[]byte"
	default:
		goType = "interface{}"
		log.Printf("unknown pg type: %v\n", udt_name)
	}
	if isArray {
		goType = "[]" + goType
	}
	return goType
}

func UuidFound(columns []*vrm.Column) bool {
	for _, column := range columns {
		switch strings.ToLower(column.Type) {
		case "uuid":
			return true
		}
	}
	return false
}

func SerialId(columns []*vrm.Column) *vrm.Column {
	for _, column := range columns {
		if column.Serial {
			return column
		}
	}
	return nil
}

func NonSerialIdColumns(columns []*vrm.Column) []*vrm.Column {
	var nonSerialColumns []*vrm.Column
	for _, column := range columns {
		if !column.Serial {
			nonSerialColumns = append(nonSerialColumns, column)
		}
	}
	return nonSerialColumns
}

func DateTimeFound(columns []*vrm.Column) bool {
	for _, column := range columns {
		switch strings.ToLower(column.Type) {
		case "date", "timestamp":
			return true
		}
	}
	return false
}

func VariableName(columnName string) string {
	return CamelTheSnake(columnName)
}
func VariableNames(separator string, columns []*vrm.Column, args ...string) string {
	var b strings.Builder

	var modAfter string = ""
	var modBefore string = ""

	switch len(args) {
	case 1:
		modAfter = args[0]

	case 2:
		modBefore = args[0]
		modAfter = args[1]

	}
	for i, col := range columns {
		b.WriteString(modBefore)
		b.WriteString(CamelTheSnake(col.Name))
		b.WriteString(modAfter)
		if i < len(columns)-1 {
			b.WriteString(separator)
		}
	}
	return b.String()
}

func VariableAndType(col *vrm.Column) string {
	variableAndType := CamelTheSnake(col.Name) + " " + col.GoType
	return variableAndType
}

func VariablesTypesAndTags(separator string, columns []*vrm.Column, conf *Config) string {
	var b strings.Builder

	for i, col := range columns {
		b.WriteString(CamelTheSnake(col.Name))
		b.WriteRune(' ')
		b.WriteString((col.GoType))

		//writeTags is true if one of json,xml,dbl tag configuration is true
		//It is used in order to write the start and end tag declaration with `
		writeTags := !conf.NoJSONTag && !conf.NoXMLTag && !conf.NoDbTag

		//open tag if one of the 3 exist
		if writeTags {
			b.WriteString("\t\t`")
		}

		if !conf.NoJSONTag {
			b.WriteString(`json:"`)
			b.WriteString(SmallCamelTheCase(col.Name))
			b.WriteRune('"')

			//if more tags follow, separate them with comma
			if !conf.NoXMLTag || !conf.NoDbTag {
				b.WriteRune(' ')
			}
		}

		if !conf.NoXMLTag {
			b.WriteString(`xml:"`)
			b.WriteString(SmallCamelTheCase(col.Name))
			b.WriteRune('"')

			//if Db Tag is following, separate the current tag with next one with a comma (,)
			if !conf.NoDbTag {
				b.WriteRune(' ')
			}
		}

		if !conf.NoDbTag {
			b.WriteString(`db:"`)
			b.WriteString(col.Name)
			b.WriteRune('"')
		}

		//Close tag if any existed
		if writeTags {
			b.WriteRune('`')
		}

		if i < len(columns)-1 {
			b.WriteString(separator)
		}

	}
	return b.String()
}

func VariablesAndTypes(separator string, columns []*vrm.Column, args ...string) string {
	var b strings.Builder

	var modAfter string = ""
	var modBefore string = ""

	switch len(args) {
	case 1:
		modAfter = args[0]

	case 2:
		modBefore = args[0]
		modAfter = args[1]

	}

	for i, col := range columns {
		b.WriteString(modBefore)
		b.WriteString(CamelTheSnake(col.Name))
		b.WriteString(modAfter)
		b.WriteRune(' ')
		b.WriteString((col.GoType))
		if i < len(columns)-1 {
			b.WriteString(separator)
		}
	}
	return b.String()
}

func VariableDollarNums(separator string, columns []*vrm.Column) string {
	var b strings.Builder

	size := len(columns)
	for i := 1; i <= size; i++ {
		b.WriteRune('$')
		b.WriteString(strconv.Itoa(i))
		if i < size {
			b.WriteString(separator)
		}
	}
	return b.String()
}

func EscapedColumns(separator string, columns []*vrm.Column) string {
	var b strings.Builder
	for i, col := range columns {
		b.WriteString(EscapeIfReservedWord(col.Name))
		if i < len(columns)-1 {
			b.WriteString(separator)
		}
	}
	return b.String()
}

func EscapedColumnsAndVariableNums(separator string, columns []*vrm.Column, args ...interface{}) string {
	var b strings.Builder

	var mod string = ""
	startNum := 1

	switch len(args) {

	case 1, 2:
		for _, arg := range args {
			switch arg.(type) {
			case string:
				mod = arg.(string)
			case int:
				startNum = arg.(int)
			}
		}
	}

	for i, col := range columns {
		b.WriteString(EscapeIfReservedWord(col.Name))
		b.WriteString(mod)
		b.WriteString("=")
		b.WriteRune('$')
		b.WriteString(strconv.Itoa(i + startNum))

		if i < len(columns)-1 {
			b.WriteString(separator)
		}
	}
	return b.String()
}

func PrimaryKeyOrUniqueConstraints(allConstraints []*ConstraintInf) []*ConstraintInf {
	if allConstraints == nil {
		return nil
	}

	var filtered []*ConstraintInf

	for _, constraint := range allConstraints {

		switch strings.ToUpper(constraint.Type) {
		case "PRIMARY KEY", "UNIQUE":
			filtered = append(filtered, constraint)
		}
	}
	return filtered
}

func NoPrimaryKeyNorUniqueConstraints(allConstraints []*ConstraintInf) []*ConstraintInf {
	if allConstraints == nil {
		return nil
	}

	var filtered []*ConstraintInf

	for _, constraint := range allConstraints {

		switch strings.ToUpper(constraint.Type) {
		case "PRIMARY KEY", "UNIQUE":
		default:
			filtered = append(filtered, constraint)
		}
	}
	return filtered
}

func NoPrimaryKeyNorUniqueColumns(allColumns []*vrm.Column, pkOrUqColumns []*vrm.Column) []*vrm.Column {
	if allColumns == nil {
		return nil
	}

	var filtered []*vrm.Column

outer:
	for _, col := range allColumns {

		for _, col2 := range pkOrUqColumns {
			//if a primary key or unique column found then continue to next given
			if col.Name == col2.Name {
				continue outer
			}
		}
		//otherwise this column is not a primary key or unique key, so append it to NoPrimaryKeyNorUniqueColumns
		filtered = append(filtered, col)

	}
	return filtered
}

type Identer struct {
	ident            string
	accumulatedIdent []string
	builder          strings.Builder
}

func (d *Identer) Nl(times ...int) *Identer {

	if times == nil || len(times) == 0 {
		d.builder.WriteString("\n")
	} else {
		for i := 0; i < times[0]; i++ {
			d.builder.WriteString("\n")
		}
	}
	d.builder.WriteString(d.ident)
	return d
}

func (d *Identer) S(str string) *Identer {
	d.builder.WriteString(str)
	return d
}
func (d *Identer) Int(i int) *Identer {
	d.builder.WriteString(strconv.Itoa(i))
	return d
}

func (d *Identer) R(r rune) *Identer {
	d.builder.WriteRune(r)
	return d
}
func (d *Identer) Buf(buf []byte) *Identer {
	d.builder.Write(buf)
	return d
}

func (d *Identer) MoreIdent(ident string, times ...int) *Identer {
	if times == nil || len(times) == 0 {
		d.accumulatedIdent = append(d.accumulatedIdent, ident)
	} else {
		newIdent := &strings.Builder{}
		for i := 0; i < times[0]; i++ {
			newIdent.WriteString(ident)
		}
		d.accumulatedIdent = append(d.accumulatedIdent, newIdent.String())
	}
	d.ident = strings.Join(d.accumulatedIdent, "")
	return d
}

func (d *Identer) LessIdent() *Identer {
	if len(d.accumulatedIdent) == 0 {
		d.ident = ""
		return d
	}
	d.accumulatedIdent = d.accumulatedIdent[0 : len(d.accumulatedIdent)-1]
	d.ident = strings.Join(d.accumulatedIdent, "")
	return d
}

func (d *Identer) SetIdent(ident string) *Identer {
	d.ident = ident
	d.accumulatedIdent = []string{d.ident}
	return d
}
func (d *Identer) Ident() *Identer {
	d.builder.WriteString(d.ident)
	return d
}

func (d *Identer) Copy(other *Identer) *Identer {
	copy(d.accumulatedIdent, other.accumulatedIdent)
	d.ident = other.ident
	d.builder = strings.Builder{}
	d.builder.WriteString(other.builder.String())
	return d
}

func (d *Identer) String() string {
	return d.builder.String()
}
