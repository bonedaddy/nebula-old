GO111MODULE = on
export GO111MODULE

cert.pb.go: cert.proto .FORCE
	# go build github.com/golang/protobuf/protoc-gen-go
	# PATH="$(PWD):$(PATH)" protoc --go_out=. $<
	go get github.com/gogo/protobuf/protoc-gen-gofaster
	protoc --gogofaster_out=. $<
	# rm protoc-gen-go

.FORCE:
