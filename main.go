package main

import (
	"log"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/anypb"
	mcp_v1alpha1 "istio.io/api/mcp/v1alpha1"
	security_v1beta1 "istio.io/api/security/v1beta1"
	"istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"mcpoverxds/xds"
)

type myServer struct {
	pushc chan struct{}
}

func (m *myServer) StreamAggregatedResources(stream xds.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	if peerInfo, ok := peer.FromContext(stream.Context()); ok {
		log.Println(peerInfo)
	}
	pushall(stream)
	for {
		select {
		case <-m.pushc:
			pushall(stream)
		}
	}

	return nil
}

func (m *myServer) DeltaAggregatedResources(delta xds.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	return nil
}

func pushall(stream xds.AggregatedDiscoveryService_StreamAggregatedResourcesServer) {

	pa := v1beta1.PeerAuthentication{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.istio.io/v1beta1",
			Kind:       "PeerAuthentication",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "istio-system",
		},
		Spec: security_v1beta1.PeerAuthentication{
			Mtls: &security_v1beta1.PeerAuthentication_MutualTLS{
				Mode: security_v1beta1.PeerAuthentication_MutualTLS_STRICT,
			},
		},
	}

	a, _ := anypb.New(&pa.Spec)

	mcpResource := mcp_v1alpha1.Resource{
		Metadata: &mcp_v1alpha1.Metadata{
			Name:       pa.ObjectMeta.Namespace + "/" + pa.ObjectMeta.Name,
			CreateTime: &timestamp.Timestamp{Seconds: time.Now().Unix()},
			Labels:     map[string]string{"hello": "test", "pa": "true"},
			Version:    "1111415485643",
		},
		Body: a,
	}

	apb, _ := anypb.New(&mcpResource)

	stream.Send(&xds.DiscoveryResponse{
		TypeUrl:     "security.istio.io/v1beta1/PeerAuthentication",
		VersionInfo: "1",
		Nonce:       "",
		Resources:   []*anypb.Any{apb},
	})
}

func main() {
	server := grpc.NewServer()
	listen, err := net.Listen("tcp", "10.114.10.202:9090")
	if err != nil {
		panic(err)
	}
	xds.RegisterAggregatedDiscoveryServiceServer(server, &myServer{})
	err = server.Serve(listen)
	if err != nil {
		return
	}
}
