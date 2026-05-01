package grpcserver

import (
	"context"
	"errors"
	"strings"

	"github.com/Pelfox/go-shortener/internal/grpc/proto"
	"github.com/Pelfox/go-shortener/internal/services"
	"github.com/Pelfox/go-shortener/internal/storage"
	"github.com/Pelfox/go-shortener/pkg"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const authorizationMetadataKey = "authorization"

// Server реализует gRPC API сервиса сокращения ссылок.
type Server struct {
	shortenerpb.UnimplementedShortenerServiceServer

	shortenerService *services.ShortenerService
}

// NewServer создаёт gRPC-сервер и регистрирует на нём ShortenerService.
func NewServer(
	shortenerService *services.ShortenerService,
	userService *services.UserService,
	logger zerolog.Logger,
	options ...grpc.ServerOption,
) *grpc.Server {
	options = append([]grpc.ServerOption{
		grpc.UnaryInterceptor(AuthInterceptor(userService, logger)),
	}, options...)

	server := grpc.NewServer(options...)
	shortenerpb.RegisterShortenerServiceServer(server, &Server{
		shortenerService: shortenerService,
	})

	return server
}

// AuthInterceptor авторизует gRPC-запросы через metadata authorization.
func AuthInterceptor(
	userService *services.UserService,
	logger zerolog.Logger,
) grpc.UnaryServerInterceptor {
	logger = logger.With().Str("interceptor", "grpc_auth").Logger()

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		authorization := authorizationFromMetadata(ctx)
		if authorization != "" {
			userID, err := userService.VerifyUserCookieValue(authorization)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, "authorization metadata is invalid")
			}

			logger.Info().
				Str("user_id", userID).
				Str("method", info.FullMethod).
				Msg("user is authenticated")

			return handler(context.WithValue(ctx, pkg.ContextUserIDKey, userID), req)
		}

		userID, authorization := userService.CreateUserCookie()
		if err := grpc.SetHeader(ctx, metadata.Pairs(authorizationMetadataKey, authorization)); err != nil {
			return nil, status.Error(codes.Internal, "failed to set authorization metadata")
		}

		return handler(context.WithValue(ctx, pkg.ContextUserIDKey, userID), req)
	}
}

func authorizationFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get(authorizationMetadataKey)
	if len(values) == 0 {
		return ""
	}

	authorization := strings.TrimSpace(values[0])
	const bearerPrefix = "bearer "
	if strings.HasPrefix(strings.ToLower(authorization), bearerPrefix) {
		return strings.TrimSpace(authorization[len(bearerPrefix):])
	}

	return authorization
}

// ShortenURL сокращает URL и является gRPC-аналогом POST /api/shorten.
func (s *Server) ShortenURL(
	ctx context.Context,
	request *shortenerpb.URLShortenRequest,
) (*shortenerpb.URLShortenResponse, error) {
	shortURL, conflict, err := s.shortenerService.CreateShortLink(ctx, request.GetUrl())
	if err != nil {
		if errors.Is(err, services.ErrDestinationEmpty) {
			return nil, status.Error(codes.InvalidArgument, "destination URL is empty")
		}
		if errors.Is(err, storage.ErrInvalidContext) {
			return nil, status.Error(codes.Unauthenticated, "user is not authenticated")
		}
		return nil, status.Error(codes.Internal, "failed to shorten URL")
	}

	return &shortenerpb.URLShortenResponse{
		Result:   shortURL,
		Conflict: conflict,
	}, nil
}

// ExpandURL возвращает исходный URL и является gRPC-аналогом GET /<id>.
func (s *Server) ExpandURL(
	ctx context.Context,
	request *shortenerpb.URLExpandRequest,
) (*shortenerpb.URLExpandResponse, error) {
	destination, err := s.shortenerService.GetDestination(ctx, request.GetId())
	if err != nil {
		switch {
		case errors.Is(err, services.ErrShortIDEmpty):
			return nil, status.Error(codes.InvalidArgument, "short ID is empty")
		case errors.Is(err, services.ErrDestinationNotFound):
			return nil, status.Error(codes.NotFound, "destination not found")
		case errors.Is(err, services.ErrDeleted):
			return nil, status.Error(codes.FailedPrecondition, "destination has been deleted")
		default:
			return nil, status.Error(codes.Internal, "failed to expand URL")
		}
	}

	return &shortenerpb.URLExpandResponse{Result: destination}, nil
}

// ListUserURLs возвращает ссылки пользователя и является gRPC-аналогом GET /api/user/urls.
func (s *Server) ListUserURLs(
	ctx context.Context,
	_ *emptypb.Empty,
) (*shortenerpb.UserURLsResponse, error) {
	links, err := s.shortenerService.GetUserLinks(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidContext) {
			return nil, status.Error(codes.Unauthenticated, "user is not authenticated")
		}
		return nil, status.Error(codes.Internal, "failed to list user URLs")
	}

	response := &shortenerpb.UserURLsResponse{
		Url: make([]*shortenerpb.URLData, 0, len(links)),
	}
	for _, link := range links {
		response.Url = append(response.Url, &shortenerpb.URLData{
			ShortUrl:    link.ShortURL,
			OriginalUrl: link.OriginalURL,
		})
	}

	return response, nil
}
