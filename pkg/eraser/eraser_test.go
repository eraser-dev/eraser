package main

import (
	"context"
	"errors"
	"testing"

	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

func TestParseEndpointWithFallBackProtocol(t *testing.T) {
	var testCases = []struct {
		endpoint         string
		fallbackProtocol string
		protocol         string
		addr             string
		err              error
	}{
		{
			endpoint:         "unix:///run/containerd/containerd.sock",
			fallbackProtocol: "unix",
			protocol:         "unix",
			addr:             "/run/containerd/containerd.sock",
			err:              nil,
		},
		{
			endpoint:         "192.168.123.132",
			fallbackProtocol: "unix",
			protocol:         "unix",
			addr:             "",
			err:              nil,
		},
	}

	for _, tc := range testCases {
		p, a, e := parseEndpointWithFallbackProtocol(tc.endpoint, tc.fallbackProtocol)
		if p != tc.protocol || a != tc.addr || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
	}

}

func TestParseEndpoint(t *testing.T) {
	var testCases = []struct {
		endpoint string
		protocol string
		addr     string
		err      error
	}{
		{
			endpoint: "unix:///run/containerd/containerd.sock",
			protocol: "unix",
			addr:     "/run/containerd/containerd.sock",
			err:      nil,
		},
		{
			endpoint: "192.168.123.132",
			protocol: "",
			addr:     "",
			err:      ErrEndpointDeprecated,
		},
		{
			endpoint: "https://myaccount.blob.core.windows.net/mycontainer/myblob",
			protocol: "https",
			addr:     "",
			err:      ErrProtocolNotSupported,
		},
	}

	for _, tc := range testCases {
		p, a, e := parseEndpoint(tc.endpoint)
		if p != tc.protocol || a != tc.addr || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
	}
}

type testClient struct {
	containers []*pb.Container
	images     []*pb.Image
}

func (client *testClient) ListImages(ctx context.Context) ([]*pb.Image, error) {
	images := make([]*pb.Image, len(client.images))
	return copy(images, client.images), nil
}

func (client *testClient) ListContainers(ctx context.Context) ([]*pb.Container, error) {
	containers := make([]*pb.Container, len(client.containers))
	return copy(containers, client.containers), nil
}
