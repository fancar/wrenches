module github.com/fancar/wrenches

go 1.15

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.14.1
	github.com/brocaar/chirpstack-api/go/v3 v3.12.5
	github.com/brocaar/chirpstack-network-server/v3 v3.16.8
	github.com/brocaar/lorawan v0.0.0-20220715134808-3b283dda1534
	github.com/go-redis/redis/v7 v7.4.0
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/protobuf v1.5.3
	github.com/jmoiron/sqlx v1.3.1
	github.com/lib/pq v1.10.2
	github.com/mohae/struct2csv v0.0.0-20151122200941-e72239694eae
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.7.0
	github.com/spf13/viper v1.7.1
	google.golang.org/protobuf v1.31.0
)

replace github.com/brocaar/chirpstack-api/go/v3 => github.com/fancar/chirpstack-api/go/v3 v3.12.6-0.20230911075212-d33324e2d198
