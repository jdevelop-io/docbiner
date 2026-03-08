module github.com/docbiner/docbiner/services/worker

go 1.25.4

require (
	github.com/docbiner/docbiner/internal/config v0.0.0
	github.com/docbiner/docbiner/internal/database v0.0.0
	github.com/docbiner/docbiner/internal/delivery v0.0.0
	github.com/docbiner/docbiner/internal/domain v0.0.0
	github.com/docbiner/docbiner/internal/pdfutil v0.0.0
	github.com/docbiner/docbiner/internal/queue v0.0.0
	github.com/docbiner/docbiner/internal/renderer v0.0.0
	github.com/docbiner/docbiner/internal/storage v0.0.0
	github.com/google/uuid v1.6.0
)

require (
	github.com/chromedp/cdproto v0.0.0-20250803210736-d308e07a266d // indirect
	github.com/chromedp/chromedp v0.14.2 // indirect
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.2.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20250725192818-e39067aee2d2 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/hhrutter/lzw v1.0.0 // indirect
	github.com/hhrutter/pkcs7 v0.2.0 // indirect
	github.com/hhrutter/tiff v1.0.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.4 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/minio/crc64nvme v1.0.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.0.87 // indirect
	github.com/nats-io/nats.go v1.42.0 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pdfcpu/pdfcpu v0.11.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/image v0.32.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace (
	github.com/docbiner/docbiner/internal/config => ../../internal/config
	github.com/docbiner/docbiner/internal/database => ../../internal/database
	github.com/docbiner/docbiner/internal/delivery => ../../internal/delivery
	github.com/docbiner/docbiner/internal/domain => ../../internal/domain
	github.com/docbiner/docbiner/internal/pdfutil => ../../internal/pdfutil
	github.com/docbiner/docbiner/internal/queue => ../../internal/queue
	github.com/docbiner/docbiner/internal/renderer => ../../internal/renderer
	github.com/docbiner/docbiner/internal/storage => ../../internal/storage
)
