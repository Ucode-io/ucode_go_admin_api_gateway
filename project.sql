create table if not exists Projects {
    name varchar,
    namespace varchar unique,
    object_builder_service_host varchar,
    object_builder_service_port varchar,
    auth_service_host varchar,
    auth_service_port varchar,
    analytics_service_host varchar,
    analytics_service_port varchar,
}