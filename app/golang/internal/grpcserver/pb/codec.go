package pb

import (
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/encoding"
)

// jsonCodec implements google.golang.org/grpc/encoding.Codec.
//
// It is registered under the name "proto" (the codec name gRPC selects by
// default when a call carries no content-subtype), which lets this hand
// rolled service run over a completely standard gRPC/HTTP2/TLS stack
// without depending on protoc-generated descriptors. Both server and
// client in this repo import this package, so they always agree on wire
// format. This is purely an offline-authoring workaround: swapping to
// real protobuf wire encoding later only requires regenerating pb.go with
// protoc and deleting this file.
type jsonCodec struct{}

func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("pb: json marshal: %w", err)
	}
	return b, nil
}

func (jsonCodec) Unmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("pb: json unmarshal: %w", err)
	}
	return nil
}

func (jsonCodec) Name() string { return "proto" }

func init() {
	encoding.RegisterCodec(jsonCodec{})
}
