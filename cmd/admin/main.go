package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]
	switch subcommand {
	case "create":
		handleCreate(os.Args[2:])
	case "reset-pass":
		handleResetPass(os.Args[2:])
	default:
		fmt.Printf("Unknown subcommand: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: aigen-admin <subcommand> [flags]")
	fmt.Println("Subcommands:")
	fmt.Println("  create      Create a new super-admin user")
	fmt.Println("  reset-pass  Reset the password of an existing user")
	fmt.Println("Example:")
	fmt.Println("  aigen-admin create -db=\"postgres://...\" -email=\"admin@aigen.local\" -password=\"securepass\"")
}

func handleCreate(args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	dbConn := fs.String("db", "", "Database DSN connection string (e.g. postgres://... or memory://)")
	email := fs.String("email", "", "Email address for the new admin")
	password := fs.String("password", "", "Password for the new admin")
	fs.Parse(args)

	if *dbConn == "" || *email == "" || *password == "" {
		fmt.Println("Error: db, email, and password parameters are required.")
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	dao, err := relationdbdao.CreateDao(*dbConn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dao.Close()

	if err := dao.EnsureTable(ctx); err != nil {
		log.Fatalf("Failed to ensure records table: %v", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	now := time.Now()
	user := &descriptors.User{
		Id:           now.Unix(),
		Email:        *email,
		PasswordHash: string(hashedPassword),
		Roles:        []string{descriptors.RoleSa, descriptors.RoleAdmin, descriptors.RoleUser},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	rec := datamodels.RecJSON{
		Namespace: services.UserNamespace,
		Key:       *email,
		Rec:       user,
		Tmstamp:   now,
	}

	if err := dao.Save(ctx, rec); err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	fmt.Printf("Successfully created super-admin user: %s\n", *email)
}

func handleResetPass(args []string) {
	fs := flag.NewFlagSet("reset-pass", flag.ExitOnError)
	dbConn := fs.String("db", "", "Database DSN connection string (e.g. postgres://... or memory://)")
	email := fs.String("email", "", "Email address of the user")
	password := fs.String("password", "", "New password for the user")
	fs.Parse(args)

	if *dbConn == "" || *email == "" || *password == "" {
		fmt.Println("Error: db, email, and password parameters are required.")
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	dao, err := relationdbdao.CreateDao(*dbConn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dao.Close()

	if err := dao.EnsureTable(ctx); err != nil {
		log.Fatalf("Failed to ensure records table: %v", err)
	}

	// Retrieve user
	rec, err := dao.Get(ctx, services.UserNamespace, *email)
	if err != nil || rec == nil {
		log.Fatalf("User not found: %s", *email)
	}

	var user descriptors.User
	if err := mapstructure.Decode(rec.Rec, &user); err != nil {
		log.Fatalf("Failed to decode user: %v", err)
	}

	// Update password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	rec.Rec = user
	rec.Tmstamp = user.UpdatedAt

	if err := dao.Save(ctx, *rec); err != nil {
		log.Fatalf("Failed to update password: %v", err)
	}

	fmt.Printf("Successfully updated password for user: %s\n", *email)
}
