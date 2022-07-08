package grpc_net_conn_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	goc "go-grpc-net-conn"
	"go-grpc-net-conn/testproto"
)

type dataer interface {
	GetData() []byte
}

func dataFieldFunc[T proto.Message, T1 dataer](msg T) *[]byte {
	dat, ok := any(msg).(T1)
	data := dat.GetData()

	fmt.Println("!!!!!!!!!!!-dataFieldFunc-1.0", len(data), data == nil)
	data = append(data, byte(1))
	fmt.Println("!!!!!!!!!!!-dataFieldFunc-1.1", len(data), data == nil, string(data))
	fmt.Println("!!!!!!!!!!!-dataFieldFunc-1.2", dat, ok, data, dat.GetData())
	return &data
}

func TestDataFieldGetter(t *testing.T) {
	msg := &testproto.Bytes{Data: []byte{}}

	msgDataPtr := dataFieldFunc[*testproto.Bytes, *testproto.Bytes](msg)

	expected := []byte{1, 2, 3, 4}
	*msgDataPtr = expected

	require.Equal(t, expected, msg.Data)
}

func testStreamConn(
	stream goc.Stream,
) *goc.Conn[*testproto.Bytes, *testproto.Bytes] {
	return &goc.Conn[*testproto.Bytes, *testproto.Bytes]{
		Stream:   stream,
		Request:  &testproto.Bytes{Data: []byte{}},
		Response: &testproto.Bytes{Data: []byte{}},
		Encode:   goc.SimpleEncoder[*testproto.Bytes](dataFieldFunc[*testproto.Bytes, *testproto.Bytes]),
		Decode:   goc.SimpleDecoder[*testproto.Bytes](dataFieldFunc[*testproto.Bytes, *testproto.Bytes]),
	}
}

// testStreamClient returns a fully connected stream client.
func testStreamClient(
	t *testing.T,
	impl testproto.TestServiceServer,
) testproto.TestService_StreamClient {
	// Get our gRPC client/server
	conn, server := testGRPCConn(t, func(s *grpc.Server) {
		testproto.RegisterTestServiceServer(s, impl)
	})
	t.Cleanup(func() { server.Stop() })
	t.Cleanup(func() { conn.Close() })

	// Connect for streaming
	resp, err := testproto.NewTestServiceClient(conn).Stream(
		context.Background())
	require.NoError(t, err)

	// Return our client
	return resp
}

// testGRPCConn returns a gRPC client conn and grpc server that are connected
// together and configured. The register function is used to register services
// prior to the Serve call. This is used to test gRPC connections.
func testGRPCConn(t *testing.T, register func(*grpc.Server)) (*grpc.ClientConn, *grpc.Server) {
	t.Helper()

	// Create a listener
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	server := grpc.NewServer()
	register(server)
	go server.Serve(l)

	// Connect to the server
	conn, err := grpc.Dial(
		l.Addr().String(),
		grpc.WithBlock(),
		grpc.WithInsecure())
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Connection successful, close the listener
	l.Close()

	return conn, server
}
