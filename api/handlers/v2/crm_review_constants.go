package v2

const (
	crmReviewProject             = "577d03aa-8301-4d40-88ce-196f2f7a0324"
	crmReviewEnvironment         = "8eb5d8c2-1ff0-43a4-9008-7fef20bea388"
	crmReviewResource            = "360f6d94-f623-4e47-b02d-e6d1ac1ab2a0"
	crmReviewResourceEnvironment = "9253c5b9-efa4-422f-b3fc-9658acf89b15"
	crmReviewDealsTable          = "1db4d92c-0ae6-4cce-8834-287fd5a85eb2"
	crmReviewContactsTable       = "cdd353a8-7fa9-4a1b-9976-f8b826737af4"
	crmReviewRole                = "adedea48-d3bc-4c1b-93d7-e2ac68027bb3"
	crmReviewClientType          = "8dd86304-3778-46b5-8d77-dd9684d2012e"
	crmReviewCatalogLimit        = 1024
	crmReviewOperatorClientType  = "d3477291-2093-4a6e-a530-fd65097149fa"
	crmReviewConnectionLimit     = 64
	crmReviewSQL                 = `jsonb_build_object(
'deals_table_id',(SELECT id::text FROM "table" WHERE slug='deals'),
'contacts_table_id',(SELECT id::text FROM "table" WHERE slug='contacts'),
'native_catalog_count',(SELECT count(*) FROM public."table"),
'native_catalog',(SELECT jsonb_agg(jsonb_build_object('id',t.id::text,'slug',t.slug,'is_login_table',t.is_login_table,'is_system',t.is_system) ORDER BY t.slug,t.id)
 FROM (SELECT id,slug,is_login_table,is_system FROM public."table" ORDER BY slug,id LIMIT 1025) t),
'login_compatibility',` + crmReviewLoginSQL + `,
'native_schema',(SELECT n.nspname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.oid=to_regclass('deals')),
'columns',(SELECT jsonb_agg(jsonb_build_object('table',c.relname,'column',a.attname,'type',format_type(a.atttypid,NULL),'nullable',NOT a.attnotnull,'default_present',ad.oid IS NOT NULL) ORDER BY c.relname,a.attnum)
 FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid LEFT JOIN pg_attrdef ad ON ad.adrelid=a.attrelid AND ad.adnum=a.attnum
 WHERE a.attrelid IN(to_regclass('deals'),to_regclass('contacts')) AND a.attnum>0 AND NOT a.attisdropped),
'hook_counts',jsonb_build_object(
'deals_create',(SELECT count(*) FROM custom_event WHERE table_slug='deals' AND upper(method)='CREATE'),
'deals_update',(SELECT count(*) FROM custom_event WHERE table_slug='deals' AND upper(method)='UPDATE'),
'contacts_create',(SELECT count(*) FROM custom_event WHERE table_slug='contacts' AND upper(method)='CREATE'),
'contacts_update',(SELECT count(*) FROM custom_event WHERE table_slug='contacts' AND upper(method)='UPDATE')),
'executor_role',current_user,
'executor_bypasses_rls',(SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname=current_user),
'can_insert_deals',has_table_privilege(current_user,'deals','INSERT'),
'can_update_deals',has_table_privilege(current_user,'deals','UPDATE'),
'can_insert_contacts',has_table_privilege(current_user,'contacts','INSERT')) AS result`
	// Only two fixed client types and their configuration mappings are read.
	// Login rows, profile values, defaults and credentials are never selected.
	crmReviewLoginSQL = `jsonb_build_object(
'client_type_count',(SELECT count(*) FROM public.client_type WHERE guid IN ('8dd86304-3778-46b5-8d77-dd9684d2012e'::uuid,'d3477291-2093-4a6e-a530-fd65097149fa'::uuid)),
'client_types',(SELECT COALESCE(jsonb_agg(jsonb_build_object('guid',ct.guid::text,'table_slug',ct.table_slug) ORDER BY ct.guid),'[]'::jsonb)
 FROM public.client_type ct WHERE ct.guid IN ('8dd86304-3778-46b5-8d77-dd9684d2012e'::uuid,'d3477291-2093-4a6e-a530-fd65097149fa'::uuid)),
'connection_count',(SELECT count(*) FROM public.connections WHERE client_type_id IN ('8dd86304-3778-46b5-8d77-dd9684d2012e'::uuid,'d3477291-2093-4a6e-a530-fd65097149fa'::uuid)),
'connections',(SELECT COALESCE(jsonb_agg(jsonb_build_object('guid',c.guid::text,'table_slug',c.table_slug,'field_slug',c.field_slug,'client_type_id',c.client_type_id::text) ORDER BY c.guid),'[]'::jsonb)
 FROM (SELECT guid,table_slug,field_slug,client_type_id FROM public.connections
 WHERE client_type_id IN ('8dd86304-3778-46b5-8d77-dd9684d2012e'::uuid,'d3477291-2093-4a6e-a530-fd65097149fa'::uuid) ORDER BY guid LIMIT 65) c),
'login_columns',(SELECT COALESCE(jsonb_agg(jsonb_build_object('table',l.table_slug,'column',k.column_name,'present',a.attnum IS NOT NULL,'type',CASE WHEN a.attnum IS NOT NULL THEN pg_catalog.format_type(a.atttypid,NULL) ELSE NULL END) ORDER BY l.table_slug,k.column_name),'[]'::jsonb)
 FROM (SELECT DISTINCT COALESCE(NULLIF(table_slug,''),'user') AS table_slug FROM public.client_type
 WHERE guid IN ('8dd86304-3778-46b5-8d77-dd9684d2012e'::uuid,'d3477291-2093-4a6e-a530-fd65097149fa'::uuid)) l
 CROSS JOIN (VALUES ('guid'),('user_id_auth'),('client_type_id'),('role_id')) k(column_name)
 LEFT JOIN pg_catalog.pg_namespace n ON n.nspname='public'
 LEFT JOIN pg_catalog.pg_class c ON c.relnamespace=n.oid AND c.relname=l.table_slug AND c.relkind IN ('r','p')
 LEFT JOIN pg_catalog.pg_attribute a ON a.attrelid=c.oid AND a.attname=k.column_name AND a.attnum>0 AND NOT a.attisdropped))`
)
