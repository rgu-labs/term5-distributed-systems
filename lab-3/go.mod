module github.com/rgu-labs/term5-distributed-systems/lab-3

go 1.27.0

require github.com/twmb/franz-go v1.22.1

require (
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.30 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.14.0 // indirect
)

replace github.com/rgu-labs/term5-distributed-systems/lib => ../lib
