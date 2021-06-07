package main

var Table_tpl_string = `
{{- $SerialId := serialId .Columns -}}
{{- $allColumns := .Columns -}}
{{- $constraints := .Constraints -}}
{{- $PrimaryKeyOrUniqueConstraints:= primaryKeyOrUniqueConstraints .Constraints -}}
{{- $NonSerialIdColumns := nonSerialIdColumns .Columns -}}
{{- $UpperCaseTable := .UpperCaseTable -}}
{{/* - $NoPrimaryKeyNorUniqueConstraints:= noPrimaryKeyNorUniqueConstraints .Constraints - */}}

{{- $tick := "` + "`" + `" -}}{{/* Define a backtick variable */}}
{{ define "tplAllValueNums" }} {{- range $i, $col:= .Columns}}${{ nextInt $i}}{{ if (lessThanSizeMinusOne $i .Columns)}},{{ end }} {{ end -}} {{ end }}
{{- $Table := .Table }}

type {{ .CamelCaseTable }}Table struct {
	vrm.Table
	{{ variableNames ",\n\t" .Columns}} vrm.Column
}

var {{ $UpperCaseTable }} = {{ .CamelCaseTable }}Table{
	__name__: {{ $tick }}{{ escapeIfReservedWord $Table}}{{ $tick }},

{{ range .Columns }}
	{{ camelCase .Name }} : vrm.Column{
		Name:          	{{ $tick }}{{- escapeIfReservedWord .Name -}}{{ $tick }},
		Key:           	{{ .Key  -}},
		NotInsertable: 	{{ .NotInsertable -}},
		NotUpdatable:  	{{ .NotInsertable -}},
		Serial:        	{{ .Serial -}},
	},
{{- end }}
}

func init(){
	writeTable:= true
	pWriteTable:= &writeTable
	{{ range .Columns }}
	{{ $UpperCaseTable }}.{{ camelCase .Name }}.Table = &{{ $UpperCaseTable }}
	{{ $UpperCaseTable }}.{{ camelCase .Name }}.IsWriteTable = &pWriteTable
	{{ end }}
}

func Insert{{.CamelCaseTable}} (ctx context.Context, conn vrm.Quexecer,
		{{ variablesAndTypes ",\n\t\t" $NonSerialIdColumns }})
		{{- if $SerialId -}} ({{ variableAndType $SerialId }}, err error) {

	sql := {{ $tick -}} INSERT INTO {{ .Table }} ({{- escapedColumns "," $NonSerialIdColumns }}) VALUES ( {{- variableDollarNums "," $NonSerialIdColumns}}); {{- $tick }}
	row := conn.QueryRow(ctx, sql, {{ variableNames ",\n\t\t" $NonSerialIdColumns }})

	err = row.Scan(&{{- variableName $SerialId.Name -}})
	if err != nil {
		return -1, err
	}
	{{- else -}} (tag pgconn.CommandTag, err error) {
	
	sql:= {{ $tick }}INSERT INTO {{ .Table }} ({{- escapedColumns ","  $allColumns }}) VALUES ( {{- variableDollarNums "," $allColumns}});{{ $tick }}
	
	tag, err = conn.Exec(ctx, sql, {{ variableNames ",\n\t\t" $allColumns }})

	{{- end }}
	 
	return 
}

{{/* ========= DELETE FUNCTIONS =========== */}}
{{ if $PrimaryKeyOrUniqueConstraints -}}
{{- range  $PrimaryKeyOrUniqueConstraints -}}

func Delete{{$.CamelCaseTable }}By{{ range .Columns}}{{ variableName .Name}}{{ end }} (ctx context.Context, conn vrm.Quexecer,
	{{- variablesAndTypes ",\n\t\t" .Columns }}) (pgconn.CommandTag, error){

	sql:= {{ $tick }}DELETE FROM {{ $.Table }} WHERE 
		{{ escapedColumnsAndVariableNums " AND \n\t\t" .Columns}}); {{ $tick }}

	return conn.Exec(ctx, sql,{{ variableNames ", " .Columns }}) 
}
{{ end }}{{/* range end */}}
{{ else }}

func Delete{{ .CamelCaseTable }} (ctx context.Context, conn vrm.Quexecer,
		{{ variablesAndTypes ",\n\t\t" .Columns }}) (pgconn.CommandTag, error){

	sql:= {{ $tick }}DELETE FROM {{ .Table }} WHERE 
	{{ escapedColumnsAndVariableNums " AND " .Columns}}); {{ $tick }}

	return conn.Exec(ctx, sql,{{ variableNames ", " .Columns }})
}
{{- end }}

{{/* ========= UPDATE FUNCTIONS =========== */}}
{{ if $PrimaryKeyOrUniqueConstraints -}}
{{- range $PrimaryKeyOrUniqueConstraints -}}


func Update{{$.CamelCaseTable }}By{{ range .Columns}}{{ variableName .Name}}{{ end }} (ctx context.Context, conn vrm.Quexecer,
	{{- variablesAndTypes ",\n\t\t" .Columns}}, {{ variablesAndTypes ",\n\t\t" $.Columns "New"}}) (pgconn.CommandTag, error){

	sql:= {{ $tick }}UPDATE {{ $.Table }} SET 
	{{ escapedColumnsAndVariableNums ", " $.Columns}}
	WHERE 
		{{ escapedColumnsAndVariableNums " AND \n\t\t" .Columns}}); {{ $tick }}

	return conn.Exec(ctx, sql, {{ variableNames ", " .Columns}},{{ variableNames ", " $.Columns "New"}}) 
}
{{ end }}{{/* range end */}}
{{ else }}
{{- $StartWithId:= len $.Columns -}}
                                                                                                                                                                                                                                                                                                                                                              
func Update{{ .CamelCaseTable }} (ctx context.Context, conn vrm.Quexecer,
		{{ variablesAndTypes ",\n\t\t" .Columns }}, {{ variablesAndTypes ",\n\t\t" .Columns "New"}}) (pgconn.CommandTag, error){

	sql:= {{ $tick }}UPDATE {{ $.Table }} SET 
	{{ escapedColumnsAndVariableNums ", " $.Columns}}
	WHERE
	
	{{ escapedColumnsAndVariableNums " AND \n\t\t" .Columns $StartWithId}}); {{ $tick }}

	return conn.Exec(ctx, sql, {{ variableNames ", " $.Columns "New"}}, {{ variableNames ", " .Columns}})
}
{{ end }}

{{/* variablesAndTypes ",\n\t\t" $noPkNoUqColumns "*" */}} 

{{/* ========= SELECT FUNCTIONS =========== */}}
{{ if $PrimaryKeyOrUniqueConstraints -}}
{{- range $PrimaryKeyOrUniqueConstraints -}}
{{ $noPkNoUqColumns:= noPrimaryKeyNorUniqueColumns $allColumns .Columns}}
//{{$.CamelCaseTable }}By{{ range .Columns}}{{ variableName .Name}}{{ end }} expects only one table row and
//returns the results by reference 
func {{$.CamelCaseTable }}By{{ range .Columns}}{{ variableName .Name}}{{ end }} (ctx context.Context, conn vrm.Quexecer,
		{{ if .Columns -}} {{/* if there are PK or UQ Columns then use them as input*/}}
		{{- variablesAndTypes ",\n\t\t" .Columns -}}
		{{- end }}, 
		{{ variablesAndTypes ",\n\t\t" $noPkNoUqColumns "" "*" -}}) error{

	sql:= {{ $tick }}
SELECT {{ escapedColumns ",\n\t" $.Columns}}
FROM {{ $.Table }}
WHERE {{ escapedColumnsAndVariableNums " AND \n\t" .Columns}}; {{- $tick }}

	return conn.QueryRow(ctx, sql, 
		{{ variableNames ",\n\t\t" .Columns }}).Scan({{- variableNames ",\n\t\t" $noPkNoUqColumns }})
}
{{ end }}{{/* range end */}}
{{ end }}

func Select{{ .CamelCaseTable }} (ctx context.Context, conn vrm.Quexecer, after time.Time,limit int,
	{{ variablesAndTypes ",\n\t\t" .Columns "" "*" -}}) (pgx.Rows, error){

	sql:= {{ $tick }}
SELECT {{ escapedColumns ",\n\t" $.Columns}}
FROM {{ $.Table }}
WHERE created_at >=$1
ORDER BY created_at
LIMIT $2;{{- $tick }}
	return conn.Query(ctx, sql, after, limit,
				{{ variableNames ",\n\t\t" .Columns }})
}


`
var Package_Tpl_String = `package {{ .Package }}
// generated by vrm2 (Values-Relationship-Manager). DO NOT EDIT.
{{ $tick := "` + "`" + `" }}{{/* Define a backtick variable */}}

const (
	{{ range $i, $table:= .Tables }}{{ camelCase .Name}}TableName = "{{- $table.Name -}}"
	{{ end }})
`
