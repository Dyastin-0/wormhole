// Package server implements all RPC defined in ../proto/...
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Dyastin-0/wormhole/api/header"
	"github.com/Dyastin-0/wormhole/api/interceptor"
	"github.com/Dyastin-0/wormhole/api/proto/auth"
	"github.com/Dyastin-0/wormhole/api/proto/tunnel"
	"github.com/Dyastin-0/wormhole/api/proto/user"
	"github.com/Dyastin-0/wormhole/api/store"
	"github.com/Dyastin-0/wormhole/logger"
	"github.com/Dyastin-0/wormhole/token"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	addr        string
	gatewayAddr string
	store       *store.Store
	donech      chan bool
	Logger      logger.Logger
}

func New(addr, gatewayAddr string, store *store.Store) *Server {
	return &Server{
		addr:        addr,
		gatewayAddr: gatewayAddr,
		store:       store,
		donech:      make(chan bool),
		Logger:      &logger.NoopLogger{},
	}
}

func (s *Server) Stop() {
	s.donech <- true
}

func (s *Server) Start(ctx context.Context) error {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	newCtx, cancel := context.WithCancel(ctx)

	errch := make(chan error, 2)

	go func() {
		errch <- s.startGRPC(newCtx)
		cancel()
	}()

	go func() {
		errch <- s.startHTTP(newCtx)
		cancel()
	}()

	select {
	case <-ctx.Done():
		close(s.donech)
	case <-s.donech:
		close(errch)
	}

	s.Logger.Info("server shutting down...")

	var finalErr error
	for range 2 {
		if err := <-errch; err != nil && finalErr == nil {
			finalErr = err
		}
	}

	time.Sleep(3 * time.Second)
	s.Logger.Info("server shutdown.")

	return finalErr
}

func (s *Server) startHTTP(ctx context.Context) error {
	mux := runtime.NewServeMux(
		header.DefaultOutgoingHeaderMatcher,
		header.DefaultIncomingHeaderMatcher,
	)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := user.RegisterUserServiceHandlerFromEndpoint(ctx, mux, s.addr, opts); err != nil {
		s.Logger.Fatal(fmt.Sprintf("failed to register user gateway: %v", err))
	}

	if err := auth.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, s.addr, opts); err != nil {
		s.Logger.Fatal(fmt.Sprintf("failed to register auth gateway: %v", err))
	}

	if err := tunnel.RegisterTunnelServiceHandlerFromEndpoint(ctx, mux, s.addr, opts); err != nil {
		s.Logger.Fatal(fmt.Sprintf("failed to register tunnel gateway: %v", err))
	}

	server := http.Server{
		Handler: mux,
		Addr:    s.gatewayAddr,
	}

	go func() {
		select {
		case <-ctx.Done():
		case <-s.donech:
		}

		newCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(newCtx)
	}()

	s.Logger.Info("http started.")
	if err := server.ListenAndServe(); err == http.ErrServerClosed {
		return ctx.Err()
	} else {
		return err
	}
}

func (s *Server) startGRPC(ctx context.Context) error {
	accessSecret := os.Getenv("ACCESS_SECRET")
	if accessSecret == "" {
		s.Logger.Panic("ACCESS_SECRET is not defined in .env")
	}

	refreshSecret := os.Getenv("REFRESH_SECRET")
	if refreshSecret == "" {
		s.Logger.Panic("REFRESH_SECRET is not defined in .env")
	}

	issuer := token.New(accessSecret, refreshSecret)
	methods := interceptor.DefaultMethods()
	authInterceptor := interceptor.NewAuthInterceptor(methods, issuer)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.Unary()),
		grpc.StreamInterceptor(authInterceptor.Stream()),
	)

	auth.RegisterAuthServiceServer(
		grpcServer,
		NewAuthServer(
			s.store,
			issuer,
		),
	)
	user.RegisterUserServiceServer(grpcServer, NewUserServer(s.store))
	tunnel.RegisterTunnelServiceServer(grpcServer, NewTunnelServer(s.store))

	reflection.Register(grpcServer)

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.Logger.Fatal(fmt.Sprintf("failed to listen: %v", err))
	}

	s.Logger.Info("grpc started.")

	go func() {
		select {
		case <-ctx.Done():
		case <-s.donech:
		}

		grpcServer.GracefulStop()
	}()

	err = grpcServer.Serve(ln)
	if err != nil {
		return err
	}

	s.Logger.Info("grpc server shutdown.")
	return ctx.Err()
}
