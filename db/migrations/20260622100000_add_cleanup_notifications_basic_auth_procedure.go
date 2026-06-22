package migrations

import (
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// AddCleanupNotificationsBasicAuthProcedure creates a stored procedure for manually
// cleaning up orphaned notifications-basic-authentication records.
//
// IMPORTANT: The procedure is created but NOT executed automatically.
// To run the cleanup, manually execute via gabi API:
//
//	{"query": "SELECT cleanup_notifications_basic_auth()"}
func AddCleanupNotificationsBasicAuthProcedure() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20260622100000",
		Migrate: func(db *gorm.DB) error {
			logging.Log.Info(`Migration "add cleanup_notifications_basic_auth procedure" started`)
			defer logging.Log.Info(`Migration "add cleanup_notifications_basic_auth procedure" ended`)

			err := db.Transaction(func(tx *gorm.DB) error {
				// Create the stored procedure
				procedureSQL := `
CREATE OR REPLACE FUNCTION cleanup_notifications_basic_auth()
RETURNS JSON
LANGUAGE plpgsql
AS $$
DECLARE
    deleted_ids_array BIGINT[];
    deleted_count INTEGER;
    result_json JSON;
BEGIN
    -- Collect IDs that will be deleted
    SELECT ARRAY_AGG(id)
    INTO deleted_ids_array
    FROM authentications
    WHERE authtype = 'notifications-basic-authentication';

    -- Delete the authentications and get the count
    -- Note: ON DELETE CASCADE will automatically handle any related
    -- application_authentications records if they exist
    DELETE FROM authentications
    WHERE authtype = 'notifications-basic-authentication';

    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    -- Handle case where nothing was deleted
    IF deleted_ids_array IS NULL THEN
        deleted_ids_array := ARRAY[]::BIGINT[];
    END IF;

    -- Build the result JSON
    result_json := json_build_object(
        'deleted_count', deleted_count,
        'deleted_ids', deleted_ids_array
    );

    -- Log the operation
    RAISE NOTICE 'cleanup_notifications_basic_auth: deleted % records with IDs: %',
        deleted_count, deleted_ids_array;

    RETURN result_json;
END;
$$;

COMMENT ON FUNCTION cleanup_notifications_basic_auth() IS
'Deletes orphaned authentications with authtype ''notifications-basic-authentication''. Returns JSON with deleted_count and deleted_ids. Must be called manually - not executed automatically.';
`

				result := tx.Exec(procedureSQL)
				if result.Error != nil {
					return result.Error
				}

				logging.Log.Info("Created cleanup_notifications_basic_auth() stored procedure")
				logging.Log.Warn("MANUAL ACTION REQUIRED: To run the cleanup, execute via gabi: SELECT cleanup_notifications_basic_auth()")
				return nil
			})

			return err
		},
		Rollback: func(db *gorm.DB) error {
			err := db.Transaction(func(tx *gorm.DB) error {
				result := tx.Exec("DROP FUNCTION IF EXISTS cleanup_notifications_basic_auth()")
				if result.Error != nil {
					return result.Error
				}

				logging.Log.Info("Dropped cleanup_notifications_basic_auth() stored procedure")
				return nil
			})

			return err
		},
	}
}
