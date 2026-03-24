package new_object_builder_service

// CRUD operation types used by the AI chat CRUD flow.
// These complement the protobuf-generated types when proto regeneration is not available.

// GetProjectTablesSchemaRequest is the request for GetProjectTablesSchema RPC.
type GetProjectTablesSchemaRequest struct {
	ResourceEnvId string `json:"resource_env_id"`
}

func (x *GetProjectTablesSchemaRequest) GetResourceEnvId() string {
	if x != nil {
		return x.ResourceEnvId
	}
	return ""
}

// DBColumn represents a database column.
type DBColumn struct {
	ColumnName string `json:"column_name"`
	DataType   string `json:"data_type"`
	IsNullable string `json:"is_nullable"`
}

func (x *DBColumn) GetColumnName() string {
	if x != nil {
		return x.ColumnName
	}
	return ""
}

func (x *DBColumn) GetDataType() string {
	if x != nil {
		return x.DataType
	}
	return ""
}

func (x *DBColumn) GetIsNullable() string {
	if x != nil {
		return x.IsNullable
	}
	return ""
}

// DBTableSchema represents a table with its columns.
type DBTableSchema struct {
	TableName string      `json:"table_name"`
	Columns   []*DBColumn `json:"columns"`
}

func (x *DBTableSchema) GetTableName() string {
	if x != nil {
		return x.TableName
	}
	return ""
}

func (x *DBTableSchema) GetColumns() []*DBColumn {
	if x != nil {
		return x.Columns
	}
	return nil
}

// GetProjectTablesSchemaResponse is the response for GetProjectTablesSchema RPC.
type GetProjectTablesSchemaResponse struct {
	Tables []*DBTableSchema `json:"tables"`
}

func (x *GetProjectTablesSchemaResponse) GetTables() []*DBTableSchema {
	if x != nil {
		return x.Tables
	}
	return nil
}

// ExecuteCrudOperationRequest is the request for ExecuteCrudOperation RPC.
type ExecuteCrudOperationRequest struct {
	ResourceEnvId string `json:"resource_env_id"`
	Operation     string `json:"operation"`
	Table         string `json:"table"`
	DataJson      string `json:"data_json"`
	WhereJson     string `json:"where_json"`
}

func (x *ExecuteCrudOperationRequest) GetResourceEnvId() string {
	if x != nil {
		return x.ResourceEnvId
	}
	return ""
}

func (x *ExecuteCrudOperationRequest) GetOperation() string {
	if x != nil {
		return x.Operation
	}
	return ""
}

func (x *ExecuteCrudOperationRequest) GetTable() string {
	if x != nil {
		return x.Table
	}
	return ""
}

func (x *ExecuteCrudOperationRequest) GetDataJson() string {
	if x != nil {
		return x.DataJson
	}
	return ""
}

func (x *ExecuteCrudOperationRequest) GetWhereJson() string {
	if x != nil {
		return x.WhereJson
	}
	return ""
}

// ExecuteCrudOperationResponse is the response for ExecuteCrudOperation RPC.
type ExecuteCrudOperationResponse struct {
	ResultJson   string `json:"result_json"`
	RowsAffected int32  `json:"rows_affected"`
}

func (x *ExecuteCrudOperationResponse) GetResultJson() string {
	if x != nil {
		return x.ResultJson
	}
	return ""
}

func (x *ExecuteCrudOperationResponse) GetRowsAffected() int32 {
	if x != nil {
		return x.RowsAffected
	}
	return 0
}
