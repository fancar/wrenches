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

# Prometheus metrics configuration.
#
# Using Prometheus (and Grafana), it is possible to visualize various
# simulation metrics like:
#   * -
[prometheus]

# IP:port to bind the Prometheus endpoint to.
#
# Metrics can be retrieved from /metrics.
bind="{{ .Prometheus.Bind }}"
`

var configCmd = &cobra.Command{
  Use:   "configfile",
  Short: "Print the configuration file",
  RunE: func(cmd *cobra.Command, args []string) error {
    t := template.Must(template.New("config").Parse(configTemplate))
    err := t.Execute(os.Stdout, &config.C)
    if err != nil {
      return errors.Wrap(err, "execute config template error")
    }
    return nil
  },
}
