module github.com/RedHatInsights/sources-api-go

go 1.16

require (
	github.com/RedHatInsights/rbac-client-go v1.0.0
	github.com/alicebob/miniredis/v2 v2.17.0
	github.com/aws/aws-sdk-go v1.42.22
	github.com/gertd/go-pluralize v0.1.7
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/google/uuid v1.1.2
	github.com/hashicorp/vault/api v1.1.1
	github.com/iancoleman/strcase v0.2.0
	github.com/jackc/pgx/v4 v4.11.0
	github.com/labstack/echo-contrib v0.12.0
	github.com/labstack/echo/v4 v4.6.1
	github.com/labstack/gommon v0.3.1
	github.com/neko-neko/echo-logrus/v2 v2.0.1
	github.com/prometheus/client_golang v1.11.0
	github.com/redhatinsights/app-common-go v1.6.0
	github.com/redhatinsights/platform-go-middlewares v0.10.0
	github.com/redhatinsights/sources-superkey-worker v0.0.0-20220110114734-d076299a7d68
	github.com/segmentio/kafka-go v0.4.25
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.10.0
	gorm.io/datatypes v1.0.1
	gorm.io/driver/postgres v1.1.0
	gorm.io/gorm v1.21.11
	sigs.k8s.io/yaml v1.3.0
)
