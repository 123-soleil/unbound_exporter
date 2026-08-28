package metrics

import (
	"bytes"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"

	"github.com/letsencrypt/unbound_exporter/exporter"
)

const homePageTemplate string = `
<!DOCTYPE html>
<html>
	<head><title>Unbound Exporter</title></head>
	<body>
		<h1>Unbound Exporter</h1>
		<p><a href='{{ .MetricsPath }}'>Metrics</a></p>
		<p><a href='{{ .HealthPath }}'>Health</a></p>
	</body>
</html>
`

// homePageText renders the html template for the homepage with the user-configured links
func homePageText(metricsPath, healthPath string) []byte {
	tmpl := template.Must(template.New("homePage").Parse(homePageTemplate))

	var out bytes.Buffer
	err := tmpl.Execute(&out, struct {
		MetricsPath string
		HealthPath  string
	}{
		MetricsPath: metricsPath,
		HealthPath:  healthPath,
	})
	if err != nil {
		panic(err) // Unreachable: Template is static and known to be well-formed
	}
	return out.Bytes()
}

// NewMetricServer starts the http server on listenAddress. If webConfigFile
// is non-empty, it is used to enable TLS (and optionally mTLS via
// client_auth_type/client_ca_file) as documented at
// https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md.
func NewMetricServer(listenAddress, webConfigFile, metricsPath, healthPath string, exp *exporter.UnboundExporter, log *slog.Logger) error {
	prometheus.MustRegister(exp)
	prometheus.MustRegister(version.NewCollector("unbound_exporter"))

	// Deliberately use our own mux to add routes rather than relying on the
	// global variable created by http.DefaultServeMux
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())
	mux.HandleFunc(healthPath, func(w http.ResponseWriter, req *http.Request) {
		if exp.UnboundUp() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("sad"))
		}
	})

	renderedHomePage := homePageText(metricsPath, healthPath)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(renderedHomePage)
	})

	server := &http.Server{Handler: mux}
	flagConfig := &web.FlagConfig{
		WebListenAddresses: &[]string{listenAddress},
		WebConfigFile:      &webConfigFile,
	}
	return web.ListenAndServe(server, flagConfig, log)
}
