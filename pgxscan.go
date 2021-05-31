package vrm

import (
	"context"
	"reflect"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

// Select is a high-level function that queries rows from Querier and calls the ScanAll function.
// See ScanAll for details.
func Select(ctx context.Context, db Quexecer, dst interface{}, query string, args ...interface{}) error {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "vrm: query multiple result rows")
	}
	err = ScanAll(dst, rows)
	return errors.WithStack(err)
}

// Get is a high-level function that queries rows from Querier and calls the ScanOne function.
// See ScanOne for details.
func Get(ctx context.Context, db Quexecer, dst interface{}, query string, args ...interface{}) error {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "vrm: query one result row")
	}
	err = ScanOne(dst, rows)
	return errors.WithStack(err)
}

// ScanAll iterates all rows to the end. After iterating it closes the rows,
// and propagates any errors that could pop up.
// It expects that destination should be a slice. For each row it scans data and appends it to the destination slice.
// ScanAll supports both types of slices: slice of structs by a pointer and slice of structs by value,
// for example:
//
//     type User struct {
//         ID    string
//         Name  string
//         Email string
//         Age   int
//     }
//
//     var usersByPtr []*User
//     var usersByValue []User
//
// Both usersByPtr and usersByValue are valid destinations for ScanAll function.
//
// Before starting, ScanAll resets the destination slice,
// so if it's not empty it will overwrite all existing elements.
func ScanAll(dst interface{}, rows pgx.Rows) error {
	err := processRows(dst, rows, true /* multipleRows */)
	return errors.WithStack(err)
}

// ScanOne iterates all rows to the end and makes sure that there was exactly one row
// otherwise it returns an error. Use NotFound function to check if there were no rows.
// After iterating ScanOne closes the rows,
// and propagates any errors that could pop up.
// It scans data from that single row into the destination.
func ScanOne(dst interface{}, rows pgx.Rows) error {
	err := processRows(dst, rows, false /* multipleRows */)
	return errors.WithStack(err)
}

// NotFound returns true if err is a not found error.
// This error is returned by ScanOne if there were no rows.
func NotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

var errNotFound = errors.New("vrm: no row was found")

type sliceDestinationMeta struct {
	val             reflect.Value
	elementBaseType reflect.Type
	elementByPtr    bool
}

func processRows(dst interface{}, rows pgx.Rows, multipleRows bool) error {
	defer rows.Close() // nolint: errcheck
	var sliceMeta *sliceDestinationMeta
	if multipleRows {
		var err error
		sliceMeta, err = parseSliceDestination(dst)
		if err != nil {
			return errors.WithStack(err)
		}
		// Make sure slice is empty.
		sliceMeta.val.Set(sliceMeta.val.Slice(0, 0))
	}
	rs := NewRowScanner(rows)
	var rowsAffected int
	for rows.Next() {
		var err error
		if multipleRows {
			err = scanSliceElement(rs, sliceMeta)
		} else {
			err = rs.Scan(dst)
		}
		if err != nil {
			return errors.WithStack(err)
		}
		rowsAffected++
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "vrm: rows final error")
	}

	rows.Close()

	exactlyOneRow := !multipleRows
	if exactlyOneRow {
		if rowsAffected == 0 {
			return errors.WithStack(errNotFound)
		} else if rowsAffected > 1 {
			return errors.Errorf("vrm: expected 1 row, got: %d", rowsAffected)
		}
	}
	return nil
}

func parseSliceDestination(dst interface{}) (*sliceDestinationMeta, error) {
	dstValue, err := parseDestination(dst)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dstType := dstValue.Type()

	if dstValue.Kind() != reflect.Slice {
		return nil, errors.Errorf(
			"vrm: destination must be a slice, got: %v", dstType,
		)
	}

	elementBaseType := dstType.Elem()
	var elementByPtr bool
	// If it's a slice of pointers to structs,
	// we handle it the same way as it would be slice of struct by value
	// and dereference pointers to values,
	// because eventually we work with fields.
	// But if it's a slice of primitive type e.g. or []string or []*string,
	// we must leave and pass elements as is to Rows.Scan().
	if elementBaseType.Kind() == reflect.Ptr {
		elementBaseTypeElem := elementBaseType.Elem()
		if elementBaseTypeElem.Kind() == reflect.Struct {
			elementBaseType = elementBaseTypeElem
			elementByPtr = true
		}
	}

	meta := &sliceDestinationMeta{
		val:             dstValue,
		elementBaseType: elementBaseType,
		elementByPtr:    elementByPtr,
	}
	return meta, nil
}

func scanSliceElement(rs *RowScanner, sliceMeta *sliceDestinationMeta) error {
	dstValPtr := reflect.New(sliceMeta.elementBaseType)
	if err := rs.Scan(dstValPtr.Interface()); err != nil {
		return errors.WithStack(err)
	}
	var elemVal reflect.Value
	if sliceMeta.elementByPtr {
		elemVal = dstValPtr
	} else {
		elemVal = dstValPtr.Elem()
	}

	sliceMeta.val.Set(reflect.Append(sliceMeta.val, elemVal))
	return nil
}

type startScannerFunc func(rs *RowScanner, dstValue reflect.Value) error

//go:generate mockery --name startScannerFunc --inpackage

// RowScanner embraces Rows and exposes the Scan method
// that allows scanning data from the current row into the destination.
// The first time the Scan method is called
// it parses the destination type via reflection and caches all required information for further scans.
// Due to this caching mechanism, it's not allowed to call Scan for destinations of different types,
// the behavior is unknown in that case.
// RowScanner doesn't proceed to the next row nor close them, it should be done by the client code.
//
// The main benefit of using this type directly
// is that you can instantiate a RowScanner and manually iterate over the rows
// and control how data is scanned from each row.
// This can be beneficial if the result set is large
// and you don't want to allocate a slice for all rows at once
// as it would be done in ScanAll.
//
// ScanOne and ScanAll both use RowScanner type internally.
type RowScanner struct {
	rows               pgx.Rows
	columns            []string
	columnToFieldIndex map[string][]int
	mapElementType     reflect.Type
	started            bool
	start              startScannerFunc
}

// NewRowScanner returns a new instance of the RowScanner.
func NewRowScanner(rows pgx.Rows) *RowScanner {
	return &RowScanner{rows: rows, start: startScanner}
}

// Scan scans data from the current row into the destination.
// On the first call it caches expensive reflection work and uses it the future calls.
// See RowScanner for details.
func (rs *RowScanner) Scan(dst interface{}) error {
	dstVal, err := parseDestination(dst)
	if err != nil {
		return errors.WithStack(err)
	}
	err = rs.doScan(dstVal)
	return errors.WithStack(err)
}

// ScanRow creates a new RowScanner and calls RowScanner.Scan
// that scans current row data into the destination.
// It's just a helper function if you don't bother with efficiency
// and don't want to instantiate a new RowScanner before iterating the rows,
// so it could cache the reflection work between Scan calls.
// See RowScanner for details.
func ScanRow(dst interface{}, rows pgx.Rows) error {
	rs := NewRowScanner(rows)
	err := rs.Scan(dst)
	return errors.WithStack(err)
}

func parseDestination(dst interface{}) (reflect.Value, error) {
	dstVal := reflect.ValueOf(dst)

	if !dstVal.IsValid() || (dstVal.Kind() == reflect.Ptr && dstVal.IsNil()) {
		return reflect.Value{}, errors.Errorf("vrm: destination must be a non nil pointer")
	}
	if dstVal.Kind() != reflect.Ptr {
		return reflect.Value{}, errors.Errorf("vrm: destination must be a pointer, got: %v", dstVal.Type())
	}

	dstVal = dstVal.Elem()
	return dstVal, nil
}

func (rs *RowScanner) doScan(dstValue reflect.Value) error {
	if !rs.started {
		if err := rs.start(rs, dstValue); err != nil {
			return errors.WithStack(err)
		}
		rs.started = true
	}
	var err error
	switch dstValue.Kind() {
	case reflect.Struct:
		err = rs.scanStruct(dstValue)
	case reflect.Map:
		err = rs.scanMap(dstValue)
	default:
		err = rs.scanPrimitive(dstValue)
	}
	return errors.WithStack(err)
}

func startScanner(rs *RowScanner, dstValue reflect.Value) error {

	columns := make([]string, len(rs.rows.FieldDescriptions()))
	for i, fd := range rs.rows.FieldDescriptions() {
		columns[i] = string(fd.Name)
	}

	if err := rs.ensureDistinctColumns(); err != nil {
		return errors.WithStack(err)
	}
	if dstValue.Kind() == reflect.Struct {
		rs.columnToFieldIndex = getColumnToFieldIndexMap(dstValue.Type())
		return nil
	}
	if dstValue.Kind() == reflect.Map {
		mapType := dstValue.Type()
		if mapType.Key().Kind() != reflect.String {
			return errors.Errorf(
				"vrm: invalid type %v: map must have string key, got: %v",
				mapType, mapType.Key(),
			)
		}
		rs.mapElementType = mapType.Elem()
		return nil
	}
	// It's the primitive type case.
	columnsNumber := len(rs.columns)
	if columnsNumber != 1 {
		return errors.Errorf(
			"vrm: to scan into a primitive type, columns number must be exactly 1, got: %d",
			columnsNumber,
		)
	}
	return nil
}

func (rs *RowScanner) scanStruct(structValue reflect.Value) error {
	scans := make([]interface{}, len(rs.columns))
	for i, column := range rs.columns {
		fieldIndex, ok := rs.columnToFieldIndex[column]
		if !ok {
			return errors.Errorf(
				"vrm: column: '%s': no corresponding field found, or it's unexported in %v",
				column, structValue.Type(),
			)
		}
		// Struct may contain embedded structs by ptr that defaults to nil.
		// In order to scan values into a nested field,
		// we need to initialize all nil structs on its way.
		initializeNested(structValue, fieldIndex)

		fieldVal := structValue.FieldByIndex(fieldIndex)
		scans[i] = fieldVal.Addr().Interface()
	}
	err := rs.rows.Scan(scans...)
	return errors.Wrap(err, "vrm: scan row into struct fields")
}

func (rs *RowScanner) scanMap(mapValue reflect.Value) error {
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapValue.Type()))
	}

	scans := make([]interface{}, len(rs.columns))
	values := make([]reflect.Value, len(rs.columns))
	for i := range rs.columns {
		valuePtr := reflect.New(rs.mapElementType)
		scans[i] = valuePtr.Interface()
		values[i] = valuePtr.Elem()
	}
	if err := rs.rows.Scan(scans...); err != nil {
		return errors.Wrap(err, "vrm: scan rows into map")
	}
	// We can't set reflect values into destination map before scanning them,
	// because reflect will set a copy, just like regular map behaves,
	// and scan won't modify the map element.
	for i, column := range rs.columns {
		key := reflect.ValueOf(column)
		value := values[i]
		mapValue.SetMapIndex(key, value)
	}
	return nil
}

func (rs *RowScanner) scanPrimitive(value reflect.Value) error {
	err := rs.rows.Scan(value.Addr().Interface())
	return errors.Wrap(err, "vrm: scan row value into a primitive type")
}

func (rs *RowScanner) ensureDistinctColumns() error {
	seen := make(map[string]struct{}, len(rs.columns))
	for _, column := range rs.columns {
		if _, ok := seen[column]; ok {
			return errors.Errorf("vrm: rows contain a duplicate column '%s'", column)
		}
		seen[column] = struct{}{}
	}
	return nil
}

var dbStructTagKey = "db"

type toTraverse struct {
	Type         reflect.Type
	IndexPrefix  []int
	ColumnPrefix string
}

func getColumnToFieldIndexMap(structType reflect.Type) map[string][]int {
	result := make(map[string][]int, structType.NumField())
	var queue []*toTraverse
	queue = append(queue, &toTraverse{Type: structType, IndexPrefix: nil, ColumnPrefix: ""})
	for len(queue) > 0 {
		traversal := queue[0]
		queue = queue[1:]
		structType := traversal.Type
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)

			if field.PkgPath != "" && !field.Anonymous {
				// Field is unexported, skip it.
				continue
			}

			dbTag, dbTagPresent := field.Tag.Lookup(dbStructTagKey)

			if dbTag == "-" {
				// Field is ignored, skip it.
				continue
			}
			index := append(traversal.IndexPrefix, field.Index...)

			columnPart := dbTag
			if !dbTagPresent {
				columnPart = toSnakeCase(field.Name)
			}
			if !field.Anonymous {
				column := buildColumn(traversal.ColumnPrefix, columnPart)

				if _, exists := result[column]; !exists {
					result[column] = index
				}
			}

			childType := field.Type
			if field.Type.Kind() == reflect.Ptr {
				childType = field.Type.Elem()
			}
			if childType.Kind() == reflect.Struct {
				if field.Anonymous {
					// If "db" tag is present for embedded struct
					// use it with "." to prefix all column from the embedded struct.
					// the default behavior is to propagate columns as is.
					columnPart = dbTag
				}
				columnPrefix := buildColumn(traversal.ColumnPrefix, columnPart)
				queue = append(queue, &toTraverse{
					Type:         childType,
					IndexPrefix:  index,
					ColumnPrefix: columnPrefix,
				})
			}
		}
	}

	return result
}

func buildColumn(parts ...string) string {
	var notEmptyParts []string
	for _, p := range parts {
		if p != "" {
			notEmptyParts = append(notEmptyParts, p)
		}
	}
	return strings.Join(notEmptyParts, ".")
}

func initializeNested(structValue reflect.Value, fieldIndex []int) {
	i := fieldIndex[0]
	field := structValue.Field(i)

	// Create a new instance of a struct and set it to field,
	// if field is a nil pointer to a struct.
	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct && field.IsNil() {
		field.Set(reflect.New(field.Type().Elem()))
	}
	if len(fieldIndex) > 1 {
		initializeNested(reflect.Indirect(field), fieldIndex[1:])
	}
}

var (
	matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCapRe   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnakeCase(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
