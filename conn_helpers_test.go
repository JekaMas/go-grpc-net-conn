package grpc_net_conn_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	goc "github.com/JekaMas/go-grpc-net-conn"
	"github.com/JekaMas/go-grpc-net-conn/testproto"
)

func dataFieldFunc[_ proto.Message](msg *testproto.Bytes) *[]byte {
	return &msg.Data
}

func dataFieldFuncOld(msg proto.Message) *[]byte {
	return &msg.(*testproto.Bytes).Data
}

func TestDataFieldGetter(t *testing.T) {
	t.Parallel()

	t.Run("generic getter", func(t *testing.T) {
		msg := &testproto.Bytes{Data: []byte{3, 2, 1}}

		msgDataPtr := dataFieldFunc[*testproto.Bytes](msg)

		expected := []byte{1, 2, 3, 4}
		*msgDataPtr = expected

		require.Equal(t, expected, msg.Data)
	})

	t.Run("normal getter", func(t *testing.T) {
		msg := &testproto.Bytes{Data: []byte{3, 2, 1}}

		msgDataPtr := dataFieldFuncOld(msg)

		expected := []byte{1, 2, 3, 4}
		*msgDataPtr = expected

		require.Equal(t, expected, msg.Data)
	})
}

func TestDataFieldGetterOld(t *testing.T) {
	msg := &testproto.Bytes{Data: []byte{3, 2, 1}}

	msgDataPtr := dataFieldFunc[*testproto.Bytes](msg)

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
		Encode:   goc.SimpleEncoder[*testproto.Bytes](dataFieldFunc[*testproto.Bytes]),
		Decode:   goc.SimpleDecoder[*testproto.Bytes](dataFieldFunc[*testproto.Bytes]),
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
