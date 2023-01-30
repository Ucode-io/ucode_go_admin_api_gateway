package models

import "google.golang.org/protobuf/runtime/protoimpl"

type FieldsAndRelationsM struct {
	fields CreateFieldRequest `json:"fields"`
}

type CreateRelationRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TableFrom              string            `protobuf:"bytes,1,opt,name=table_from,json=tableFrom,proto3" json:"table_from,omitempty"`
	TableTo                string            `protobuf:"bytes,2,opt,name=table_to,json=tableTo,proto3" json:"table_to,omitempty"`
	Type                   string            `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
	ViewFields             []string          `protobuf:"bytes,4,rep,name=view_fields,json=viewFields,proto3" json:"view_fields,omitempty"`
	Summaries              []*Summary        `protobuf:"bytes,5,rep,name=summaries,proto3" json:"summaries,omitempty"`
	Editable               bool              `protobuf:"varint,6,opt,name=editable,proto3" json:"editable,omitempty"`
	IsEditable             bool              `protobuf:"varint,7,opt,name=is_editable,json=isEditable,proto3" json:"is_editable,omitempty"`
	Title                  string            `protobuf:"bytes,8,opt,name=title,proto3" json:"title,omitempty"`
	ViewType               string            `protobuf:"bytes,9,opt,name=view_type,json=viewType,proto3" json:"view_type,omitempty"`
	Columns                []string          `protobuf:"bytes,10,rep,name=columns,proto3" json:"columns,omitempty"`
	QuickFilters           []*QuickFilter    `protobuf:"bytes,11,rep,name=quick_filters,json=quickFilters,proto3" json:"quick_filters,omitempty"`
	GroupFields            []string          `protobuf:"bytes,12,rep,name=group_fields,json=groupFields,proto3" json:"group_fields,omitempty"`
	RelationTableSlug      string            `protobuf:"bytes,13,opt,name=relation_table_slug,json=relationTableSlug,proto3" json:"relation_table_slug,omitempty"`
	DynamicTables          []*DynamicTable   `protobuf:"bytes,14,rep,name=dynamic_tables,json=dynamicTables,proto3" json:"dynamic_tables,omitempty"`
	RelationFieldSlug      string            `protobuf:"bytes,15,opt,name=relation_field_slug,json=relationFieldSlug,proto3" json:"relation_field_slug,omitempty"`
	AutoFilters            []*AutoFilter     `protobuf:"bytes,16,rep,name=auto_filters,json=autoFilters,proto3" json:"auto_filters,omitempty"`
	DefaultValues          []string          `protobuf:"bytes,17,rep,name=default_values,json=defaultValues,proto3" json:"default_values,omitempty"`
	IsUserIdDefault        bool              `protobuf:"varint,18,opt,name=is_user_id_default,json=isUserIdDefault,proto3" json:"is_user_id_default,omitempty"`
	Cascadings             []*Cascading      `protobuf:"bytes,19,rep,name=cascadings,proto3" json:"cascadings,omitempty"`
	ObjectIdFromJwt        bool              `protobuf:"varint,20,opt,name=object_id_from_jwt,json=objectIdFromJwt,proto3" json:"object_id_from_jwt,omitempty"`
	CascadingTreeTableSlug string            `protobuf:"bytes,21,opt,name=cascading_tree_table_slug,json=cascadingTreeTableSlug,proto3" json:"cascading_tree_table_slug,omitempty"`
	CascadingTreeFieldSlug string            `protobuf:"bytes,22,opt,name=cascading_tree_field_slug,json=cascadingTreeFieldSlug,proto3" json:"cascading_tree_field_slug,omitempty"`
	ActionRelations        []*ActionRelation `protobuf:"bytes,23,rep,name=action_relations,json=actionRelations,proto3" json:"action_relations,omitempty"`
	DefaultLimit           string            `protobuf:"bytes,24,opt,name=default_limit,json=defaultLimit,proto3" json:"default_limit,omitempty"`
	MultipleInsert         bool              `protobuf:"varint,25,opt,name=multiple_insert,json=multipleInsert,proto3" json:"multiple_insert,omitempty"`
	UpdatedFields          []string          `protobuf:"bytes,26,rep,name=updated_fields,json=updatedFields,proto3" json:"updated_fields,omitempty"`
	MultipleInsertField    string            `protobuf:"bytes,27,opt,name=multiple_insert_field,json=multipleInsertField,proto3" json:"multiple_insert_field,omitempty"`
	ProjectId              string            `protobuf:"bytes,28,opt,name=project_id,json=projectId,proto3" json:"project_id,omitempty"`
	CommitId               int64             `protobuf:"varint,29,opt,name=commit_id,json=commitId,proto3" json:"commit_id,omitempty"`
	CommitGuid             string            `protobuf:"bytes,30,opt,name=commit_guid,json=commitGuid,proto3" json:"commit_guid,omitempty"`
}
