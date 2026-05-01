package grpcserver

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/Pelfox/go-shortener/internal/grpc/proto"
	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func prepareClient(t *testing.T) shortenerpb.ShortenerServiceClient {
	t.Helper()

	logger := zerolog.New(io.Discard)
	store := storage.NewInMemoryStorage("")
	userService := services.NewUserService([]byte("very-strong-secret"))
	shortenerService := services.NewShortenerService(
		context.Background(),
		"http://localhost:8080/test",
		store,
		logger,
		nil,
	)

	listener := bufconn.Listen(1024 * 1024)
	server := NewServer(shortenerService, userService, logger)
	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
		listener.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial gRPC server: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	return shortenerpb.NewShortenerServiceClient(conn)
}

func TestServer_ShortenExpandAndListUserURLs(t *testing.T) {
	client := prepareClient(t)

	var header metadata.MD
	shortenResponse, err := client.ShortenURL(
		context.Background(),
		&shortenerpb.URLShortenRequest{Url: "https://example.com"},
		grpc.Header(&header),
	)
	if err != nil {
		t.Fatalf("failed to shorten URL: %v", err)
	}

	if !strings.HasPrefix(shortenResponse.GetResult(), "http://localhost:8080/test/") {
		t.Fatalf("unexpected short URL: %q", shortenResponse.GetResult())
	}
	if shortenResponse.GetConflict() {
		t.Fatal("did not expect conflict for the first shortened URL")
	}

	authorization := header.Get("authorization")
	if len(authorization) != 1 || authorization[0] == "" {
		t.Fatalf("expected authorization response metadata, got %v", authorization)
	}

	authorizedContext := metadata.AppendToOutgoingContext(
		context.Background(),
		"authorization",
		authorization[0],
	)

	shortID := strings.TrimPrefix(shortenResponse.GetResult(), "http://localhost:8080/test/")
	expandResponse, err := client.ExpandURL(
		authorizedContext,
		&shortenerpb.URLExpandRequest{Id: shortID},
	)
	if err != nil {
		t.Fatalf("failed to expand URL: %v", err)
	}
	if expandResponse.GetResult() != "https://example.com" {
		t.Fatalf("unexpected destination: %q", expandResponse.GetResult())
	}

	listResponse, err := client.ListUserURLs(authorizedContext, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("failed to list user URLs: %v", err)
	}
	if len(listResponse.GetUrl()) != 1 {
		t.Fatalf("expected 1 user URL, got %d", len(listResponse.GetUrl()))
	}
	if listResponse.GetUrl()[0].GetOriginalUrl() != "https://example.com" {
		t.Fatalf("unexpected original URL: %q", listResponse.GetUrl()[0].GetOriginalUrl())
	}
}

func TestServer_ShortenURLInvalidAuthorization(t *testing.T) {
	client := prepareClient(t)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "invalid")
	_, err := client.ShortenURL(ctx, &shortenerpb.URLShortenRequest{Url: "https://example.com"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated error, got %v", err)
	}
}
