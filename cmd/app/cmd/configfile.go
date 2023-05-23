package cmd

import (
  "os"
  "text/template"

  "github.com/pkg/errors"
  "github.com/spf13/cobra"

  "github.com/fancar/wrenches/internal/config"
)

const configTemplate = `[general]
# Log level
#
# debug=5, info=4, warning=3, error=2, fatal=1, panic=0
log_level={{ .General.LogLevel }}

# Redis settings
#
# Please note that Redis 2.6.0+ is required.
[redis]

# Server address or addresses.
#
# Set multiple addresses when connecting to a cluster.
servers=[{{ range $index, $elm := .Redis.Servers }}
  "{{ $elm }}",{{ end }}
]

# Password.
#
# Set the password when connecting to Redis requires password authentication.
password="{{ .Redis.Password }}"

# Database index.
#
# By default, this can be a number between 0-15.
database={{ .Redis.Database }}

# Redis Cluster.
#
# Set this to true when the provided URLs are pointing to a Redis Cluster
# instance.
cluster={{ .Redis.Cluster }}

# Master name.
#
# Set the master name when the provided URLs are pointing to a Redis Sentinel
# instance.
master_name="{{ .Redis.MasterName }}"

# Connection pool size.
#
# Default (when set to 0) is 10 connections per every CPU.
pool_size={{ .Redis.PoolSize }}

# infrastructure of network-server
[network_server]
  # PostgreSQL settings.
  #
  # Please note that PostgreSQL 9.5+ is required.
  [postgres-ns]
  # PostgreSQL dsn (e.g.: postgres://user:password@hostname/database?sslmode=disable).
  #
  # Besides using an URL (e.g. 'postgres://user:password@hostname/database?sslmode=disable')
  # it is also possible to use the following format:
  # 'user=chirpstack_as dbname=chirpstack_as sslmode=disable'.
  #
  # The following connection parameters are supported:
  #
  # * dbname - The name of the database to connect to
  # * user - The user to sign in as
  # * password - The user's password
  # * host - The host to connect to. Values that start with / are for unix domain sockets. (default is localhost)
  # * port - The port to bind to. (default is 5432)
  # * sslmode - Whether or not to use SSL (default is require, this is not the default for libpq)
  # * fallback_application_name - An application_name to fall back to if one isn't provided.
  # * connect_timeout - Maximum wait for connection, in seconds. Zero or not specified means wait indefinitely.
  # * sslcert - Cert file location. The file must contain PEM encoded data.
  # * sslkey - Key file location. The file must contain PEM encoded data.
  # * sslrootcert - The location of the root certificate file. The file must contain PEM encoded data.
  #
  # Valid values for sslmode are:
  #
  # * disable - No SSL
  # * require - Always SSL (skip verification)
  # * verify-ca - Always SSL (verify that the certificate presented by the server was signed by a trusted CA)
  # * verify-full - Always SSL (verify that the certification presented by the server was signed by a trusted CA and the server host name matches the one in the certificate)
  dsn="{{ .NetworkServer.PostgreSQL.DSN }}"

  # Max open connections.
  #
  # This sets the max. number of open connections that are allowed in the
  # PostgreSQL connection pool (0 = unlimited).
  max_open_connections={{ .NetworkServer.PostgreSQL.MaxOpenConnections }}

  # Max idle connections.
  #
  # This sets the max. number of idle connections in the PostgreSQL connection
  # pool (0 = no idle connections are retained).
  max_idle_connections={{ .NetworkServer.PostgreSQL.MaxIdleConnections }}

# infrastructure of application_server
[application_server]
  
  # all the same as for network-server
  [postgres-as]
  # PostgreSQL dsn (e.g.: postgres://user:password@hostname/database?sslmode=disable).
  dsn="{{ .ApplicationServer.PostgreSQL.DSN }}"

  # Max open connections.
  # PostgreSQL connection pool (0 = unlimited).
  max_open_connections={{ .ApplicationServer.PostgreSQL.MaxOpenConnections }}

  # Max idle connections.
  # pool (0 = no idle connections are retained).
  max_idle_connections={{ .ApplicationServer.PostgreSQL.MaxIdleConnections }}

`

var configCmd = &cobra.Command{
  Use:   "configfile",
  Short: "Print the configuration file",
  RunE: func(cmd *cobra.Command, args []string) error {
    t := template.Must(template.New("config").Parse(configTemplate))
    cfg := config.Get()
    err := t.Execute(os.Stdout, &cfg)
    if err != nil {
      return errors.Wrap(err, "execute config template error")
    }
    return nil
  },
}
