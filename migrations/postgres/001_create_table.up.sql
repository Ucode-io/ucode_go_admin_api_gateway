create table if not exists "project" (
    "id" uuid primary key,
    "name" varchar not null,
    "namespace" varchar not null,
    "object_builder_service_host" varchar not null,
    "object_builder_service_port" varchar not null,
    "auth_service_host" varchar not null,
    "auth_service_port" varchar not null,
    "analytics_service_host" varchar not null,
    "analytics_service_port" varchar not null,
    "created_at" timestamp default now(),
    "updated_at" timestamp default now(),
    "deleted_at" timestamp,
    unique(namespace, deleted_at)
);