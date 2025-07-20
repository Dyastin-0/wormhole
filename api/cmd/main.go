package main

import (
	"context"
	dbsql "database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/Dyastin-0/wormhole/api/db"
	"github.com/Dyastin-0/wormhole/api/header"
	"github.com/Dyastin-0/wormhole/api/interceptor"
	"github.com/Dyastin-0/wormhole/api/proto/auth"
	"github.com/Dyastin-0/wormhole/api/proto/tunnel"
	"github.com/Dyastin-0/wormhole/api/proto/user"
	"github.com/Dyastin-0/wormhole/api/server"
	"github.com/Dyastin-0/wormhole/api/store"
	"github.com/Dyastin-0/wormhole/api/token"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	conn, err := dbsql.Open("sqlite3", "file:dev.db?_foreign_keys=on&_journal_mode=WAL&_cache=shared&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

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
		server.NewAuthServer(
			store,
			issuer,
		),
	)
	user.RegisterUserServiceServer(grpcServer, server.NewUserServer(store))
	tunnel.RegisterTunnelServiceServer(grpcServer, server.NewTunnelServer(store))

	reflection.Register(grpcServer)
	go startGRPC(grpcServer)
	startHTTP()
}

func startHTTP() {
	ctx := context.Background()
	mux := runtime.NewServeMux(
		header.DefaultOutgoingHeaderMatcher,
		header.DefaultIncomingHeaderMatcher,
	)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := user.RegisterUserServiceHandlerFromEndpoint(ctx, mux, "localhost:42069", opts); err != nil {
		log.Fatalf("failed to register user gateway: %v", err)
	}

	if err := auth.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, "localhost:42069", opts); err != nil {
		log.Fatalf("failed to register auth gateway: %v", err)
	}

	if err := tunnel.RegisterTunnelServiceHandlerFromEndpoint(ctx, mux, "localhost:42069", opts); err != nil {
		log.Fatalf("failed to register tunnel gateway: %v", err)
	}

	fmt.Println("http started")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}
}

func startGRPC(s *grpc.Server) {
	ln, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	fmt.Println("grpc started")

	if err := s.Serve(ln); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
