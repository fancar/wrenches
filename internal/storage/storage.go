package storage

import (
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	"github.com/fancar/wrenches/internal/config"
)

// deviceSessionTTL holds the device-session TTL.
var deviceSessionTTL time.Duration

// schedulerInterval holds the interval in which the Class-B and -C
// scheduler runs.
var schedulerInterval time.Duration

// Setup configures the storage backend.
func Setup(c config.Config) error {
	deviceSessionTTL = c.NetworkServer.DeviceSessionTTL
	// schedulerInterval = c.NetworkServer.Scheduler.SchedulerInterval

	log.Info("storage: setting up Redis client ...")
	if len(c.Redis.Servers) == 0 {
		return errors.New("at least one redis server must be configured")
	}

	if c.Redis.Cluster {
		redisClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    c.Redis.Servers,
			PoolSize: c.Redis.PoolSize,
			Password: c.Redis.Password,
		})
	} else if c.Redis.MasterName != "" {
		redisClient = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       c.Redis.MasterName,
			SentinelAddrs:    c.Redis.Servers,
			SentinelPassword: c.Redis.Password,
			DB:               c.Redis.Database,
			PoolSize:         c.Redis.PoolSize,
		})
	} else {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     c.Redis.Servers[0],
			DB:       c.Redis.Database,
			Password: c.Redis.Password,
			PoolSize: c.Redis.PoolSize,
		})
	}

	log.Info("storage: connecting to NetworkServer-PostgreSQL ... ")
	d, err := sqlx.Open("postgres", c.NetworkServer.PostgreSQL.DSN)
	if err != nil {
		return errors.Wrap(err, "storage: NetworkServer-PostgreSQL connection error")
	}
	d.SetMaxOpenConns(c.NetworkServer.PostgreSQL.MaxOpenConnections)
	d.SetMaxIdleConns(c.NetworkServer.PostgreSQL.MaxIdleConnections)
	for {
		if err := d.Ping(); err != nil {
			log.WithError(err).Warning("storage: ping NetworkServer-PostgreSQL database error, will retry in 2s")
			time.Sleep(2 * time.Second)
		} else {
			break
		}
	}
	nsDB = &DBLogger{d}

	log.Info("storage: connecting to AppServer-PostgreSQL ... ")
	d, err = sqlx.Open("postgres", c.ApplicationServer.PostgreSQL.DSN)
	if err != nil {
		return errors.Wrap(err, "storage: AppServer-PostgreSQL connection error")
	}
	d.SetMaxOpenConns(c.ApplicationServer.PostgreSQL.MaxOpenConnections)
	d.SetMaxIdleConns(c.ApplicationServer.PostgreSQL.MaxIdleConnections)
	for {
		if err := d.Ping(); err != nil {
			log.WithError(err).Warning("storage: ping AppServer-PostgreSQL database error, will retry in 2s")
			time.Sleep(2 * time.Second)
		} else {
			break
		}
	}
	asDB = &DBLogger{d}

	log.Debug("storage: setting up ClickHouse client ... ")
	// var err error
	hrDB, err = InitClickhouse(c)
	if err != nil {
		return errors.Wrap(err, "storage: ClickHouse setup error")
	}

	return nil
}

// SetupSecondRedis configures the second redis storage backend.
func SetupSecondRedis(c config.Config) error {
	cfg := c.RedisSecond
	log.Info("storage: setting up client for Second Redis storage ...")
	if len(cfg.Servers) == 0 {
		return errors.New("at least one SECOND redis server must be configured")
	}

	if cfg.Cluster {
		redisSecondClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.Servers,
			PoolSize: cfg.PoolSize,
			Password: cfg.Password,
		})
	} else if c.RedisSecond.MasterName != "" {
		redisSecondClient = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       cfg.MasterName,
			SentinelAddrs:    cfg.Servers,
			SentinelPassword: cfg.Password,
			DB:               cfg.Database,
			PoolSize:         cfg.PoolSize,
		})
	} else {
		redisSecondClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Servers[0],
			DB:       cfg.Database,
			Password: cfg.Password,
			PoolSize: cfg.PoolSize,
		})
	}
	return nil
}
