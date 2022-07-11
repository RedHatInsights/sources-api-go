-- Wrap everything inside a transaction, to be extremely conservative
BEGIN;

-- fk_exists(string, string) takes the name of a  foreign key constraint
-- and a table name as arguments, and checks if the constraint exists in
-- the given table.
--
-- Returns true if it exists.
CREATE OR REPLACE FUNCTION "fk_exists"(cn TEXT, tn TEXT) RETURNS BOOLEAN
AS $fk_exists$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM "information_schema"."table_constraints"
        WHERE "table_schema" = "current_schema"() AND "table_name" = tn AND "constraint_name" = cn
    );
END
$fk_exists$ LANGUAGE plpgsql;

-- table_exists(string) takes a table name as an argument and
-- checks if it exists in the current schema.
--
-- Returns true if it exists.
CREATE OR REPLACE FUNCTION "table_exists"(tn TEXT) RETURNS BOOLEAN
AS $table_exists$
BEGIN
	RETURN EXISTS (
		SELECT 1 FROM "information_schema"."tables"
		WHERE "table_schema" = "current_schema"() AND "table_name" = tn
	);
END
$table_exists$ LANGUAGE plpgsql;

----------------------------------------
-- Main tables, sequences and indexes --
----------------------------------------

--
-- Instead of relying on the standard "IF NOT EXISTS" features, anonymous
-- blocks are used to check if there's an existing table and the related
-- sequences and indexes. If the table doesn't exist, the rest of the
-- stuff shouldn't exist either.

--
-- application_authentications
--

DO
$$
BEGIN
    IF NOT "table_exists"('application_authentications') THEN
        CREATE TABLE "application_authentications" (
            "id" BIGINT NOT NULL,
            "tenant_id" BIGINT NOT NULL,
            "application_id" BIGINT NOT NULL,
            "authentication_id" BIGINT NOT NULL,
            "created_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "updated_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "paused_at" TIMESTAMP WITHOUT TIME ZONE
        );

        CREATE SEQUENCE "application_authentications_id_seq"
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            NO MAXVALUE
            CACHE 1;

        ALTER SEQUENCE "application_authentications_id_seq" OWNED BY "application_authentications"."id";

        ALTER TABLE ONLY "application_authentications" ALTER COLUMN "id" SET DEFAULT nextval('application_authentications_id_seq'::REGCLASS);

        ALTER TABLE ONLY "application_authentications"
            ADD CONSTRAINT "application_authentications_pkey" PRIMARY KEY ("id");

        CREATE INDEX "index_application_authentications_on_application_id" ON "application_authentications" USING btree ("application_id");
        CREATE INDEX "index_application_authentications_on_authentication_id" ON "application_authentications" USING btree ("authentication_id");
        CREATE INDEX "index_application_authentications_on_paused_at" ON "application_authentications" USING btree ("paused_at");
        CREATE INDEX "index_application_authentications_on_tenant_id" ON "application_authentications" USING btree ("tenant_id");
        CREATE UNIQUE INDEX "index_on_tenant_application_authentication" ON "application_authentications" USING btree ("tenant_id", "application_id", "authentication_id");

        RAISE NOTICE '"application_authentications": table, sequences and indexes created.';
    END IF;
END
$$;

--
-- application_types
--

DO
$$
BEGIN
    IF NOT "table_exists"('application_types') THEN
        CREATE TABLE "application_types" (
            "id" BIGINT NOT NULL,
            "name" CHARACTER VARYING NOT NULL,
            "created_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "updated_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "display_name" CHARACTER VARYING,
            "dependent_applications" JSONB,
            "supported_source_types" JSONB,
            "supported_authentication_types" JSONB
        );

        CREATE SEQUENCE "application_types_id_seq"
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            NO MAXVALUE
            CACHE 1;

        ALTER SEQUENCE "application_types_id_seq" OWNED BY "application_types"."id";

        ALTER TABLE ONLY "application_types" ALTER COLUMN "id" SET DEFAULT nextval('application_types_id_seq'::REGCLASS);

        ALTER TABLE ONLY "application_types"
            ADD CONSTRAINT "application_types_pkey" PRIMARY KEY ("id");

        CREATE UNIQUE INDEX "index_application_types_on_name" ON "application_types" USING btree ("name");

        RAISE NOTICE '"application_types": table, sequences and indexes created.';
    END IF;
END
$$;

--
-- applications
--

DO
$$
BEGIN
    IF NOT "table_exists"('applications') THEN
        CREATE TABLE "applications" (
            "id" BIGINT NOT NULL,
            "tenant_id" BIGINT NOT NULL,
            "source_id" BIGINT NOT NULL,
            "application_type_id" BIGINT NOT NULL,
            "created_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "updated_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "availability_status" CHARACTER VARYING,
            "availability_status_error" CHARACTER VARYING,
            "last_checked_at" TIMESTAMP WITHOUT TIME ZONE,
            "last_available_at" TIMESTAMP WITHOUT TIME ZONE,
            "extra" JSONB DEFAULT '{}'::JSONB,
            "superkey_data" JSONB,
            "paused_at" TIMESTAMP WITHOUT TIME ZONE
        );

        CREATE SEQUENCE "applications_id_seq"
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            NO MAXVALUE
            CACHE 1;

        ALTER SEQUENCE "applications_id_seq" OWNED BY "applications"."id";

        ALTER TABLE ONLY "applications" ALTER COLUMN "id" SET DEFAULT nextval('applications_id_seq'::REGCLASS);

        ALTER TABLE ONLY "applications"
            ADD CONSTRAINT "applications_pkey" PRIMARY KEY ("id");

        CREATE INDEX "index_applications_on_application_type_id" ON "applications" USING btree ("application_type_id");
        CREATE INDEX "index_applications_on_paused_at" ON "applications" USING btree ("paused_at");
        CREATE INDEX "index_applications_on_source_id" ON "applications" USING btree ("source_id");
        CREATE INDEX "index_applications_on_tenant_id" ON "applications" USING btree ("tenant_id");

        RAISE NOTICE '"applications": table, sequences and indexes created.';
    END IF;
END
$$;

--
-- authentications
--

DO
$$
BEGIN
    IF NOT "table_exists"('authentications') THEN
        CREATE TABLE "authentications" (
            "id" BIGINT NOT NULL,
            "resource_type" CHARACTER VARYING,
            "resource_id" integer,
            "name" CHARACTER VARYING,
            "authtype" CHARACTER VARYING,
            "username" CHARACTER VARYING,
            "password" CHARACTER VARYING,
            "tenant_id" BIGINT NOT NULL,
            "extra" JSONB,
            "availability_status" CHARACTER VARYING,
            "availability_status_error" CHARACTER VARYING,
            "last_checked_at" TIMESTAMP WITHOUT TIME ZONE,
            "last_available_at" TIMESTAMP WITHOUT TIME ZONE,
            "source_id" BIGINT,
            "paused_at" TIMESTAMP WITHOUT TIME ZONE,
            "password_hash" CHARACTER VARYING
        );

        CREATE SEQUENCE "authentications_id_seq"
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            NO MAXVALUE
            CACHE 1;

        ALTER SEQUENCE "authentications_id_seq" OWNED BY "authentications"."id";

        ALTER TABLE ONLY "authentications" ALTER COLUMN "id" SET DEFAULT nextval('authentications_id_seq'::REGCLASS);

        ALTER TABLE ONLY "authentications"
            ADD CONSTRAINT "authentications_pkey" PRIMARY KEY ("id");

        CREATE INDEX "index_authentications_on_paused_at" ON "authentications" USING btree ("paused_at");
        CREATE INDEX "index_authentications_on_resource_type_and_resource_id" ON "authentications" USING btree ("resource_type", "resource_id");
        CREATE INDEX "index_authentications_on_tenant_id" ON "authentications" USING btree ("tenant_id");

        RAISE NOTICE '"authentications": table, sequences and indexes created.';
    END IF;
END
$$;

--
-- endpoints
--

DO
$$
BEGIN
    IF NOT "table_exists"('endpoints') THEN
        CREATE TABLE "endpoints" (
            "id" BIGINT NOT NULL,
            "role" CHARACTER VARYING,
            "port" integer,
            "source_id" BIGINT,
            "created_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "updated_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "default" boolean DEFAULT false,
            "scheme" CHARACTER VARYING,
            "host" CHARACTER VARYING,
            "path" CHARACTER VARYING,
            "tenant_id" BIGINT NOT NULL,
            "verify_ssl" boolean,
            "certificate_authority" TEXT,
            "receptor_node" CHARACTER VARYING,
            "availability_status" CHARACTER VARYING,
            "availability_status_error" CHARACTER VARYING,
            "last_checked_at" TIMESTAMP WITHOUT TIME ZONE,
            "last_available_at" TIMESTAMP WITHOUT TIME ZONE,
            "paused_at" TIMESTAMP WITHOUT TIME ZONE
        );

        CREATE SEQUENCE "endpoints_id_seq"
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            NO MAXVALUE
            CACHE 1;

        ALTER SEQUENCE "endpoints_id_seq" OWNED BY "endpoints"."id";

        ALTER TABLE ONLY "endpoints" ALTER COLUMN "id" SET DEFAULT nextval('endpoints_id_seq'::REGCLASS);

        ALTER TABLE ONLY "endpoints"
            ADD CONSTRAINT "endpoints_pkey" PRIMARY KEY ("id");

        CREATE INDEX "index_endpoints_on_paused_at" ON "endpoints" USING btree ("paused_at");
        CREATE INDEX "index_endpoints_on_source_id" ON "endpoints" USING btree ("source_id");
        CREATE INDEX "index_endpoints_on_tenant_id" ON "endpoints" USING btree ("tenant_id");

        RAISE NOTICE '"endpoints": table, sequences and indexes created.';
    END IF;
END
$$;

--
-- meta_data
--

DO
$$
BEGIN
    IF NOT "table_exists"('meta_data') THEN
        CREATE TABLE "meta_data" (
            "id" BIGINT NOT NULL,
            "application_type_id" integer,
            "step" integer,
            "name" CHARACTER VARYING,
            "payload" JSONB,
            "substitutions" JSONB,
            "created_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "updated_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "type" CHARACTER VARYING
        );

        CREATE SEQUENCE "meta_data_id_seq"
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            NO MAXVALUE
            CACHE 1;

        ALTER SEQUENCE "meta_data_id_seq" OWNED BY "meta_data"."id";

        ALTER TABLE ONLY "meta_data" ALTER COLUMN "id" SET DEFAULT nextval('meta_data_id_seq'::REGCLASS);

        ALTER TABLE ONLY "meta_data"
            ADD CONSTRAINT "meta_data_pkey" PRIMARY KEY ("id");

        RAISE NOTICE '"meta_data": table, sequences and indexes created.';
    END IF;
END
$$;

--
-- rhc_connections
--

DO
$$
BEGIN
    IF NOT "table_exists"('rhc_connections') THEN
        CREATE TABLE "rhc_connections" (
            "id" BIGINT NOT NULL,
            "rhc_id" CHARACTER VARYING,
            "extra" JSONB DEFAULT '{}'::JSONB,
            "availability_status" CHARACTER VARYING,
            "availability_status_error" CHARACTER VARYING,
            "last_checked_at" TIMESTAMP WITHOUT TIME ZONE,
            "last_available_at" TIMESTAMP WITHOUT TIME ZONE,
            "created_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "updated_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL
        );

        CREATE SEQUENCE "rhc_connections_id_seq"
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            NO MAXVALUE
            CACHE 1;

        ALTER SEQUENCE "rhc_connections_id_seq" OWNED BY "rhc_connections"."id";

        ALTER TABLE ONLY "rhc_connections" ALTER COLUMN "id" SET DEFAULT nextval('rhc_connections_id_seq'::REGCLASS);

        ALTER TABLE ONLY "rhc_connections"
            ADD CONSTRAINT "rhc_connections_pkey" PRIMARY KEY ("id");

        CREATE UNIQUE INDEX "index_rhc_connections_on_rhc_id" ON "rhc_connections" USING btree ("rhc_id");

        RAISE NOTICE '"rhc_connections": table, sequences and indexes created.';
    END IF;
END
$$;

--
-- source_rhc_connections
--

DO
$$
    BEGIN
        IF NOT "table_exists"('source_rhc_connections') THEN
            CREATE TABLE "source_rhc_connections" (
                "source_id" INTEGER,
                "rhc_connection_id" INTEGER,
                "tenant_id" BIGINT
            );

            CREATE UNIQUE INDEX "index_source_rhc_connections_on_source_id_and_rhc_connection_id" ON "source_rhc_connections" USING btree ("source_id", "rhc_connection_id");

            RAISE NOTICE '"source_rhc_connections": table and index created.';
        END IF;
    END
$$;

--
-- source_types
--

DO
$$
BEGIN
    IF NOT "table_exists"('source_types') THEN
        CREATE TABLE "source_types" (
            "id" BIGINT NOT NULL,
            "name" CHARACTER VARYING NOT NULL,
            "product_name" CHARACTER VARYING NOT NULL,
            "vendor" CHARACTER VARYING NOT NULL,
            "created_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "updated_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "schema" JSONB,
            "icon_url" CHARACTER VARYING
        );

        CREATE SEQUENCE "source_types_id_seq"
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            NO MAXVALUE
            CACHE 1;

        ALTER SEQUENCE "source_types_id_seq" OWNED BY "source_types"."id";

        ALTER TABLE ONLY "source_types" ALTER COLUMN "id" SET DEFAULT nextval('source_types_id_seq'::REGCLASS);

        ALTER TABLE ONLY "source_types"
            ADD CONSTRAINT "source_types_pkey" PRIMARY KEY ("id");

        CREATE UNIQUE INDEX "index_source_types_on_name" ON "source_types" USING btree ("name");

        RAISE NOTICE '"source_types": table, sequences and indexes created.';
    END IF;
END
$$;

--
-- sources
--

DO
$$
BEGIN
    IF NOT "table_exists"('sources') THEN
        CREATE TABLE "sources" (
            "id" BIGINT NOT NULL,
            "name" CHARACTER VARYING NOT NULL,
            "uid" CHARACTER VARYING NOT NULL,
            "created_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "updated_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "tenant_id" BIGINT NOT NULL,
            "source_type_id" BIGINT NOT NULL,
            "version" CHARACTER VARYING,
            "availability_status" CHARACTER VARYING,
            "imported" CHARACTER VARYING,
            "source_ref" CHARACTER VARYING,
            "last_checked_at" TIMESTAMP WITHOUT TIME ZONE,
            "last_available_at" TIMESTAMP WITHOUT TIME ZONE,
            "app_creation_workflow" CHARACTER VARYING DEFAULT 'manual_configuration'::CHARACTER VARYING,
            "paused_at" TIMESTAMP WITHOUT TIME ZONE
        );

        CREATE SEQUENCE "sources_id_seq"
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            NO MAXVALUE
            CACHE 1;

        ALTER SEQUENCE "sources_id_seq" OWNED BY "sources"."id";

        ALTER TABLE ONLY "sources" ALTER COLUMN "id" SET DEFAULT nextval('sources_id_seq'::REGCLASS);

        ALTER TABLE ONLY "sources"
            ADD CONSTRAINT "sources_pkey" PRIMARY KEY ("id");

        CREATE INDEX "index_sources_on_paused_at" ON "sources" USING btree ("paused_at");
        CREATE INDEX index_sources_on_source_type_id ON "sources" USING btree ("source_type_id");
        CREATE INDEX index_sources_on_tenant_id ON "sources" USING btree ("tenant_id");
        CREATE UNIQUE INDEX index_sources_on_uid ON "sources" USING btree ("uid");

        RAISE NOTICE '"sources": table, sequences and indexes created.';
    END IF;
END
$$;

--
-- tenants
--

DO
$$
BEGIN
    IF NOT "table_exists"('tenants') THEN
        CREATE TABLE "tenants" (
            "id" BIGINT NOT NULL,
            "name" CHARACTER VARYING,
            "description" TEXT,
            "external_tenant" CHARACTER VARYING,
            "created_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "updated_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL
        );

        CREATE SEQUENCE "tenants_id_seq"
            START WITH 1
            INCREMENT BY 1
            NO MINVALUE
            NO MAXVALUE
            CACHE 1;

        ALTER SEQUENCE "tenants_id_seq" OWNED BY "tenants"."id";

        ALTER TABLE ONLY "tenants" ALTER COLUMN "id" SET DEFAULT nextval('tenants_id_seq'::REGCLASS);

        ALTER TABLE ONLY "tenants"
            ADD CONSTRAINT "tenants_pkey" PRIMARY KEY ("id");

        RAISE NOTICE '"tenants": table, sequences and indexes created.';
    END IF;
END
$$;

------------------
-- Foreign Keys --
------------------
-- As with the tables' checks, we assume that if one foreign key doesn't exist
-- for a table, then the other foreign keys don't exist either. So only the
-- first constraint is checked.

--
-- application_authentications
--

DO
$$
BEGIN
    IF NOT "fk_exists"('fk_rails_85a04922b1', 'application_authentications') THEN
        ALTER TABLE ONLY "application_authentications"
            ADD CONSTRAINT "fk_rails_85a04922b1" FOREIGN KEY ("tenant_id") REFERENCES "tenants"("id") ON DELETE CASCADE;

        ALTER TABLE ONLY "application_authentications"
            ADD CONSTRAINT "fk_rails_d709bbbff3" FOREIGN KEY ("authentication_id") REFERENCES "authentications"("id") ON DELETE CASCADE;

        ALTER TABLE ONLY "application_authentications"
            ADD CONSTRAINT "fk_rails_a051188e10" FOREIGN KEY ("application_id") REFERENCES "applications"("id") ON DELETE CASCADE;

        RAISE NOTICE '"application_authentications": foreign keys created.';
    END IF;
END
$$;

--
-- applications
--

DO
$$
BEGIN
    IF NOT "fk_exists"('fk_rails_ad5ea13d24', 'applications') THEN
        ALTER TABLE ONLY "applications"
            ADD CONSTRAINT "fk_rails_ad5ea13d24" FOREIGN KEY ("application_type_id") REFERENCES "application_types"("id") ON DELETE CASCADE;

        ALTER TABLE ONLY "applications"
            ADD CONSTRAINT "fk_rails_cbcddd5826" FOREIGN KEY ("tenant_id") REFERENCES "tenants"("id") ON DELETE CASCADE;

        ALTER TABLE ONLY applications
            ADD CONSTRAINT "fk_rails_064e03ae58" FOREIGN KEY ("source_id") REFERENCES "sources"("id") ON DELETE CASCADE;

        RAISE NOTICE '"applications": foreign keys created.';
    END IF;
END
$$;

--
-- authentications
--

DO
$$
BEGIN
    IF NOT "fk_exists"('fk_rails_28143f952b', 'authentications') THEN
        ALTER TABLE ONLY authentications
            ADD CONSTRAINT "fk_rails_28143f952b" FOREIGN KEY ("tenant_id") REFERENCES "tenants"("id") ON DELETE CASCADE;

        RAISE NOTICE '"authentications": foreign keys created.';
    END IF;
END
$$;

--
-- endpoints
--

DO
$$
BEGIN
    IF NOT "fk_exists"('fk_rails_430e742d27', 'endpoints') THEN
        ALTER TABLE ONLY "endpoints"
            ADD CONSTRAINT "fk_rails_430e742d27" FOREIGN KEY ("tenant_id") REFERENCES "tenants"("id") ON DELETE CASCADE;

        ALTER TABLE ONLY "endpoints"
            ADD CONSTRAINT "fk_rails_67ee0f0d63" FOREIGN KEY ("source_id") REFERENCES "sources"("id") ON DELETE CASCADE;

        RAISE NOTICE '"endpoints": foreign keys created.';
    END IF;
END
$$;

--
-- source_rhc_connections
--

DO
$$
BEGIN
    IF NOT "fk_exists"('fk_rhc_connection_id', 'source_rhc_connections') THEN
        ALTER TABLE ONLY "source_rhc_connections"
            ADD CONSTRAINT "fk_rhc_connection_id" FOREIGN KEY ("rhc_connection_id") REFERENCES "rhc_connections"("id") ON DELETE CASCADE;

        ALTER TABLE ONLY "source_rhc_connections"
            ADD CONSTRAINT "fk_source_id" FOREIGN KEY ("source_id") REFERENCES "sources"("id") ON DELETE CASCADE;

        ALTER TABLE ONLY "source_rhc_connections"
            ADD CONSTRAINT "fk_tenant_id" FOREIGN KEY ("tenant_id") REFERENCES "tenants"("id");
    END IF;
END
$$;

--
-- sources
--

DO
$$
BEGIN
    IF NOT "fk_exists"('fk_rails_e7365b4f5b', 'sources') THEN
        ALTER TABLE ONLY "sources"
            ADD CONSTRAINT "fk_rails_e7365b4f5b" FOREIGN KEY ("source_type_id") REFERENCES "source_types"("id") ON DELETE CASCADE;


        ALTER TABLE ONLY "sources"
            ADD CONSTRAINT "fk_rails_f830a376e4" FOREIGN KEY ("tenant_id") REFERENCES "tenants"("id") ON DELETE CASCADE;

        RAISE NOTICE '"sources": foreign keys created.';
    END IF;
END
$$;

-------------------------------
-- Legacy ActiveRecord stuff --
-------------------------------

DO
$$
BEGIN
    IF NOT "table_exists"('ar_internal_metadata') THEN
        CREATE TABLE "ar_internal_metadata" (
            "key" CHARACTER VARYING NOT NULL,
            "value" CHARACTER VARYING,
            "created_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL,
            "updated_at" TIMESTAMP WITHOUT TIME ZONE NOT NULL
        );

        ALTER TABLE ONLY "ar_internal_metadata"
            ADD CONSTRAINT "ar_internal_metadata_pkey" PRIMARY KEY ("key");
    END IF;
END
$$;

DO

$$
BEGIN
    IF NOT "table_exists"('ar_internal_metadata') THEN
        CREATE TABLE "schema_migrations" (
            "version" CHARACTER VARYING NOT NULL
        );

        ALTER TABLE ONLY "schema_migrations"
            ADD CONSTRAINT "schema_migrations_pkey" PRIMARY KEY ("version");
    END IF;
END
$$;

-- Drop the helper functions
DROP FUNCTION "fk_exists";
DROP FUNCTION "table_exists";

-- Finish the transaction and commit the changes
COMMIT;
