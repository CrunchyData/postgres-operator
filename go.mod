module github.com/crunchydata/postgres-operator

go 1.15

require (
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/go-logr/logr v0.3.0
	github.com/google/go-cmp v0.5.4
	github.com/sirupsen/logrus v1.6.0
	github.com/wojas/genericr v0.2.0
	github.com/xdg/stringprep v1.0.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.14.0
	go.opentelemetry.io/otel v0.14.0
	go.opentelemetry.io/otel/exporters/stdout v0.14.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.14.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	gotest.tools/v3 v3.0.3
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.7.0-alpha.6
	sigs.k8s.io/yaml v1.2.0
)
