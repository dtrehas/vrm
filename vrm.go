package vrm

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ErrNoRows              = errors.New("No rows")
	ErrMoreRowsNotExpected = errors.New("More rows not expected")
)
var (
	_ Quexecer = &pgxpool.Pool{}
	_ Quexecer = &pgx.Conn{}
	_ Quexecer = *new(pgx.Tx)
)

type Tabler interface {
	String() string
	//	ColumnSet() Columns
}

// Quexecer is an interface that pgxscan can query and get the pgx.Rows from.
// For example, it can be: *pgxpool.Pool, *pgx.Conn or pgx.Tx.
type Quexecer interface {
	Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

type Batch interface {
	Queue(query string, args ...interface{})
}

// Stringer is implemented by any value that has a String method,
// which defines the "native" format for that value.
// The String method is used to print values passed as an operand
// to any format that accepts a string or to an unformatted printer
// such as Print.
type Stringer interface {
	String() string
}

var (
	_ Quexecer = &pgxpool.Pool{}
	_ Quexecer = &pgx.Conn{}
	_ Quexecer = *new(pgx.Tx)
)

//type TableNameType string

type Table struct {
	Name        string
	Schema      string
	WriteSchema bool
}

//Column contains information for database column.
type Column struct {
	Table         *Table
	WriteTable    **bool
	Array         bool
	Key           bool
	GoType        string
	Name          string
	NotInsertable bool
	NotUpdatable  bool
	Nullable      bool
	PartialKey    bool
	Position      int
	Serial        bool
	Type          string
	Unique        bool
	//Db2Go         Converter
	//Go2Db         Converter
}

//String returns the column name or the table name + . + the column name
func (c *Column) String() string {

	if **c.WriteTable {
		return c.Table.Name + "." + c.Name
	}
	return c.Name
}

//Columns represents an array of Column.
type Columns []Column

//Filter method filters an
func (c *Columns) Filter(filters ...ColumnFilter) Columns {

	allColumns := c
	var columns = make(Columns, 0, len(*allColumns))

	for _, col := range *allColumns {

		ok := true
		for _, filter := range filters {

			if filter == nil {
				continue
			}

			if !filter(&col) {
				ok = false
			}
		}
		if ok {
			columns = append(columns, col)
		}

	}
	return columns
}

type ColumnFilter func(col *Column) bool

var NoKey ColumnFilter = func(col *Column) bool {
	return !col.Key
}

var Insertable ColumnFilter = func(col *Column) bool {
	return !col.NotInsertable
}
var Updatable ColumnFilter = func(col *Column) bool {
	return !col.NotUpdatable
}
var Keys ColumnFilter = func(col *Column) bool {
	return col.Key || col.PartialKey
}
var NoKeys ColumnFilter = func(col *Column) bool {
	return !col.Key && !col.PartialKey
}

func (cs Columns) String() string {

	var b strings.Builder

	size := len(cs)
	if size == 0 {
		return ""
	}

	for i, c := range cs {
		b.WriteString(c.Name)
		if i < size-1 {
			b.WriteString(",")
		}
	}

	return b.String()
}

type Converter func(interface{}) interface{}

type Values []interface{}

type Valuer interface {
	Values() Values
}

type ColumnValues struct {
	Columns Columns
	Values  Values
}

//ColumnsOf extracts from a Table struct, fields of a Column type.
func ColumnsOf(table interface{}) Columns {

	//Indirect returns always content even it is the pointed content of a pointer.
	tab := reflect.Indirect(reflect.ValueOf(table))

	var columns = make(Columns, 0, tab.NumField())

	for i := 0; i < tab.NumField(); i++ {

		//tab.Field(i).Type().Name() finds the type name e.g. Column
		if tab.Field(i).Type().Name() == "Column" {
			col := tab.Field(i).Interface().(Column)
			columns = append(columns, col)
		}
	}
	return columns

}
