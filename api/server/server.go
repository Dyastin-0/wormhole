// Package server implements all RPC defined in ../proto/...
package server

import (
	"context"
	dbsql "database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Dyastin-0/wormhole/api/db"
	"github.com/Dyastin-0/wormhole/api/header"
	"github.com/Dyastin-0/wormhole/api/interceptor"
	"github.com/Dyastin-0/wormhole/api/proto/auth"
	"github.com/Dyastin-0/wormhole/api/proto/tunnel"
	"github.com/Dyastin-0/wormhole/api/proto/user"
	"github.com/Dyastin-0/wormhole/api/store"
	"github.com/Dyastin-0/wormhole/token"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func Start(ctx context.Context, dbPath, httpAddr, grpcAddr string) error {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	errch := make(chan error, 2)

	go func() { errch <- startGRPC(ctx, grpcAddr, dbPath) }()
	go func() { errch <- startHTTP(ctx, httpAddr, grpcAddr) }()

	<-ctx.Done()
	log.Println("gRPC server shutting down...")

	var finalErr error
	for range 3 {
		if err := <-errch; err != nil && finalErr == nil {
			finalErr = err
		}
	}

	time.Sleep(2 * time.Second)
	log.Println("server shutdown.")

	return finalErr
}

func startHTTP(ctx context.Context, addr, grpcAddr string) error {
	mux := runtime.NewServeMux(
		header.DefaultOutgoingHeaderMatcher,
		header.DefaultIncomingHeaderMatcher,
	)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := user.RegisterUserServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatalf("failed to register user gateway: %v", err)
	}

	if err := auth.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatalf("failed to register auth gateway: %v", err)
	}

	if err := tunnel.RegisterTunnelServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatalf("failed to register tunnel gateway: %v", err)
	}

	server := http.Server{
		Handler: mux,
		Addr:    addr,
	}

	go func() {
		<-ctx.Done()

		fmt.Println("context canceled, grafully shutting down...")

		newCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(newCtx)
	}()

	fmt.Println("http started")
	if err := server.ListenAndServe(); err == http.ErrServerClosed {
		return ctx.Err()
	} else {
		return err
	}
}

func startGRPC(ctx context.Context, addr, dbPath string) error {
	// for dev use:
	// file:dev.db?_foreign_keys=on&_journal_mode=WAL&_cache=shared&_busy_timeout=5000
	conn, err := dbsql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer conn.Close()

	accessSecret := os.Getenv("ACCESS_SECRET")
	if accessSecret == "" {
		panic("ACCESS_SECRET is not defined in .env")
	}

	refreshSecret := os.Getenv("REFRESH_SECRET")
	if refreshSecret == "" {
		panic("REFRESH_SECRET is not defined in .env")
	}

	issuer := token.New(accessSecret, refreshSecret)
	methods := interceptor.DefaultMethods()
	authInterceptor := interceptor.NewAuthInterceptor(methods, issuer)

	queries := db.New(conn)
	store := store.New(queries)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.Unary()),
		grpc.StreamInterceptor(authInterceptor.Stream()),
	)

	auth.RegisterAuthServiceServer(
		grpcServer,
		NewAuthServer(
			store,
			issuer,
		),
	)
	user.RegisterUserServiceServer(grpcServer, NewUserServer(store))
	tunnel.RegisterTunnelServiceServer(grpcServer, NewTunnelServer(store))

	reflection.Register(grpcServer)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	fmt.Println("grpc started")

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	err = grpcServer.Serve(ln)
	if err != nil {
		return err
	}

	return ctx.Err()
}
