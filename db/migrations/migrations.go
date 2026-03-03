package migrations

import (
	"context"
	"time"

	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/redis"
	"github.com/go-gormigrate/gormigrate/v2"
	uuidpkg "github.com/google/uuid"
	"gorm.io/gorm"
)

var MigrationsCollection = []*gormigrate.Migration{
	InitialSchema(),
	AddOrgIdToTenants(),
	TranslateEbsAccountNumbersToOrgIds(),
	SourceTypesAddCategoryColumn(),
	AddRetryCounterToApplications(),
	AddTableUsers(),
	MakeEmptyExternalTenantsOrgIdsNull(),
	RemoveDuplicatedTenantIdsOrgIds(),
	AddTenantsExternalTenantOrgIdUniqueIndex(),
	AddUserIdColumnIntoTables(),
	AddResourceOwnershipToApplicationTypes(),
	AddApplicationConstraint(),
	RemoveOldMigrationsTable(),
	RenameForeignKeysIndexes(),
	RemoveProcessedDuplicatedTenants(),
	AvailabilityStatusColumnsNotNullConstraintDefaultValue(),
	MigrateAwsProvisioningToImageBuilder(),
	CleanupProvisioningAuthentications(),
}

var ctx = context.Background()

// redisLockKey is the key which will be used for the Redis lock when performing the migrations.
const redisLockKey = "sources-api-go-redis-lock"

// redisSleepTime is the amount of time that the client will wait until the next retry to obtain the lock.
const redisSleepTime = 3 * time.Second

// redisLockExpirationTime is the time in milliseconds that the lock will be held before it expires.
const redisLockExpirationTime = 30 * time.Second

// Migrate migrates the database schema to the latest version. Implements the single instance lock algorithm detailed
// in https://redis.io/topics/distlock#correct-implementation-with-a-single-instance. On error, it tries deleting the
// lock before exiting the program.
func Migrate(db *gorm.DB) {
	// Using UUID as the lock value since it's a safe way of obtaining a unique string among all the clients.
	uuid, err := uuidpkg.NewUUID()
	if err != nil {
		logging.Log.Fatalf(`could not generate a UUID for the Valkey lock: %s`, err)
	}

	// Before doing anything, check for the existence of the lock.
	exists, err := redis.Client.Do(ctx, redis.Client.B().Exists().Key(redisLockKey).Build()).AsInt64()
	if err != nil {
		logging.Log.Fatalf(`error when fetching the Valkey lock: %s`, err)
	}

	// If the lock is present, we must wait in order to be able to obtain it ourselves.
	lockExists := exists != 0
	for lockExists {
		time.Sleep(redisSleepTime)

		exists, err = redis.Client.Do(ctx, redis.Client.B().Exists().Key(redisLockKey).Build()).AsInt64()
		if err != nil {
			logging.Log.Fatalf(`error when checking if the Valkey lock exists: %s`, err)
		}

		lockExists = exists != 0
	}

	// Set the migrations lock with expiration.
	err = redis.Client.Do(ctx, redis.Client.B().Set().Key(redisLockKey).Value(uuid.String()).Px(redisLockExpirationTime).Build()).Error()
	if err != nil {
		logging.Log.Fatalf(`error when setting the Valkey lock: %s`, err)
	}

	// Perform the migrations and store the error for a proper return.
	migrateTool := gormigrate.New(db, gormigrate.DefaultOptions, MigrationsCollection)

	err = migrateTool.Migrate()
	if err != nil {
		logging.Log.Fatalf(`error when performing the database migrations: %s. The Valkey lock is going to try to be released...`, err)
	}

	// Once the migrations have finished, get the lock's value to attempt to release it.
	value, err := redis.Client.Do(ctx, redis.Client.B().Get().Key(redisLockKey).Build()).ToString()
	if err != nil {
		logging.Log.Fatalf(`error when getting the Valkey lock after the migrations have run: %s`, err)
	}

	// The lock's value should coincide with the one we set above. If it doesn't something very wrong happened.
	if value == uuid.String() {
		err = redis.Client.Do(ctx, redis.Client.B().Del().Key(redisLockKey).Build()).Error()
		if err != nil {
			logging.Log.Fatalf(`error when deleting the Valkey lock after the migrations have run: %s`, err)
		}
	} else {
		logging.Log.Fatalf(`migrations lock release failed. Expecting lock with value "%s", got "%s"`, uuid.String(), value)
	}
}
