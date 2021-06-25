BEGIN;

create table if not exists application_authentications
(
	id bigserial not null
		constraint application_authentications_pkey
			primary key,
	tenant_id bigint not null
		constraint fk_rails_85a04922b1
			references tenants
				on delete cascade,
	application_id bigint not null
		constraint fk_rails_a051188e10
			references applications
				on delete cascade,
	authentication_id bigint not null
		constraint fk_rails_d709bbbff3
			references authentications
				on delete cascade,
	created_at timestamp not null,
	updated_at timestamp not null,
	paused_at timestamp
);

create index if not exists index_application_authentications_on_application_id
	on application_authentications (application_id);

create index if not exists index_application_authentications_on_authentication_id
	on application_authentications (authentication_id);

create index if not exists index_application_authentications_on_paused_at
	on application_authentications (paused_at);

create index if not exists index_application_authentications_on_tenant_id
	on application_authentications (tenant_id);

create unique index if not exists index_on_tenant_application_authentication
	on application_authentications (tenant_id, application_id, authentication_id);

create table if not exists applications
(
	id bigserial not null
		constraint applications_pkey
			primary key,
	tenant_id bigint not null
		constraint fk_rails_cbcddd5826
			references tenants
				on delete cascade,
	source_id bigint not null
		constraint fk_rails_064e03ae58
			references sources
				on delete cascade,
	application_type_id bigint not null
		constraint fk_rails_ad5ea13d24
			references application_types
				on delete cascade,
	created_at timestamp not null,
	updated_at timestamp not null,
	availability_status varchar,
	availability_status_error varchar,
	last_checked_at timestamp,
	last_available_at timestamp,
	extra jsonb default '{}'::jsonb,
	superkey_data jsonb,
	paused_at timestamp
);

create index if not exists index_applications_on_application_type_id
	on applications (application_type_id);

create index if not exists index_applications_on_paused_at
	on applications (paused_at);

create index if not exists index_applications_on_source_id
	on applications (source_id);

create index if not exists index_applications_on_tenant_id
	on applications (tenant_id);

create table if not exists application_types
(
	id bigserial not null
		constraint application_types_pkey
			primary key,
	name varchar not null,
	created_at timestamp not null,
	updated_at timestamp not null,
	display_name varchar,
	dependent_applications jsonb,
	supported_source_types jsonb,
	supported_authentication_types jsonb
);

create unique index if not exists index_application_types_on_name
	on application_types (name);

create table if not exists ar_internal_metadata
(
	key varchar not null
		constraint ar_internal_metadata_pkey
			primary key,
	value varchar,
	created_at timestamp not null,
	updated_at timestamp not null
);

create table if not exists authentications
(
	id bigserial not null
		constraint authentications_pkey
			primary key,
	resource_type varchar,
	resource_id integer,
	name varchar,
	authtype varchar,
	username varchar,
	password varchar,
	tenant_id bigint not null
		constraint fk_rails_28143f952b
			references tenants
				on delete cascade,
	extra jsonb,
	availability_status varchar,
	availability_status_error varchar,
	last_checked_at timestamp,
	last_available_at timestamp,
	source_id bigint,
	paused_at timestamp
);

create index if not exists index_authentications_on_paused_at
	on authentications (paused_at);

create index if not exists index_authentications_on_resource_type_and_resource_id
	on authentications (resource_type, resource_id);

create index if not exists index_authentications_on_tenant_id
	on authentications (tenant_id);

create table if not exists availabilities
(
	id bigserial not null
		constraint availabilities_pkey
			primary key,
	resource_type varchar not null,
	resource_id bigint not null,
	action varchar not null,
	identifier varchar not null,
	availability varchar not null,
	last_checked_at timestamp,
	last_valid_at timestamp,
	created_at timestamp not null,
	updated_at timestamp not null
);

create unique index if not exists index_on_resource_action_identifier
	on availabilities (resource_type, resource_id, action, identifier);

create table if not exists endpoints
(
	id bigserial not null
		constraint endpoints_pkey
			primary key,
	role varchar,
	port integer,
	source_id bigint
		constraint fk_rails_67ee0f0d63
			references sources
				on delete cascade,
	created_at timestamp not null,
	updated_at timestamp not null,
	"default" boolean default false,
	scheme varchar,
	host varchar,
	path varchar,
	tenant_id bigint not null
		constraint fk_rails_430e742d27
			references tenants
				on delete cascade,
	verify_ssl boolean,
	certificate_authority text,
	receptor_node varchar,
	availability_status varchar,
	availability_status_error varchar,
	last_checked_at timestamp,
	last_available_at timestamp
);

create index if not exists index_endpoints_on_source_id
	on endpoints (source_id);

create index if not exists index_endpoints_on_tenant_id
	on endpoints (tenant_id);

create table if not exists meta_data
(
	id bigserial not null
		constraint meta_data_pkey
			primary key,
	application_type_id integer,
	step integer,
	name varchar,
	payload jsonb,
	substitutions jsonb,
	created_at timestamp not null,
	updated_at timestamp not null,
	type varchar
);

create table if not exists schema_migrations
(
	version varchar not null
		constraint schema_migrations_pkey
			primary key
);

create table if not exists sources
(
	id bigserial not null
		constraint sources_pkey
			primary key,
	name varchar not null,
	uid varchar not null,
	created_at timestamp not null,
	updated_at timestamp not null,
	tenant_id bigint not null
		constraint fk_rails_f830a376e4
			references tenants
				on delete cascade,
	source_type_id bigint not null
		constraint fk_rails_e7365b4f5b
			references source_types
				on delete cascade,
	version varchar,
	availability_status varchar,
	imported varchar,
	source_ref varchar,
	last_checked_at timestamp,
	last_available_at timestamp,
	app_creation_workflow varchar default 'manual_configuration'::character varying,
	paused_at timestamp
);

create index if not exists index_sources_on_paused_at
	on sources (paused_at);

create index if not exists index_sources_on_source_type_id
	on sources (source_type_id);

create index if not exists index_sources_on_tenant_id
	on sources (tenant_id);

create unique index if not exists index_sources_on_uid
	on sources (uid);

create table if not exists source_types
(
	id bigserial not null
		constraint source_types_pkey
			primary key,
	name varchar not null,
	product_name varchar not null,
	vendor varchar not null,
	created_at timestamp not null,
	updated_at timestamp not null,
	schema jsonb,
	icon_url varchar
);

create unique index if not exists index_source_types_on_name
	on source_types (name);

create table if not exists tenants
(
	id bigserial not null
		constraint tenants_pkey
			primary key,
	name varchar,
	description text,
	external_tenant varchar,
	created_at timestamp not null,
	updated_at timestamp not null
);

COMMIT;
