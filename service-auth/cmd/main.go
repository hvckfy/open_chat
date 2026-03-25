package main

import (
	"fmt"
	"log"
	"os"
	serviceauth "service-auth"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "generate-keys":
		if len(os.Args) != 4 {
			fmt.Println("Usage: go run main.go generate-keys <service-name> <key-dir>")
			os.Exit(1)
		}
		serviceName := os.Args[2]
		keyDir := os.Args[3]

		if err := serviceauth.GenerateKeys(serviceName, keyDir); err != nil {
			log.Fatalf("Failed to generate keys: %v", err)
		}

	case "setup-services":
		if len(os.Args) < 4 {
			fmt.Println("Usage: go run main.go setup-services <key-dir> <service1> <service2> ...")
			os.Exit(1)
		}
		keyDir := os.Args[2]
		services := os.Args[3:]

		if err := serviceauth.SetupServiceKeys(services, keyDir); err != nil {
			log.Fatalf("Failed to setup service keys: %v", err)
		}

	case "validate-setup":
		if len(os.Args) < 5 {
			fmt.Println("Usage: go run main.go validate-setup <service-name> <private-key-path> <public-key-path> <key-dir> <service1> <service2> ...")
			os.Exit(1)
		}
		serviceName := os.Args[2]
		privateKeyPath := os.Args[3]
		publicKeyPath := os.Args[4]
		keyDir := os.Args[5]
		services := os.Args[6:]

		config := serviceauth.ServiceConfig{
			ServiceName:    serviceName,
			PrivateKeyPath: privateKeyPath,
			PublicKeyPath:  publicKeyPath,
			KeyDir:         keyDir,
		}

		if err := serviceauth.ValidateSetup(config, services); err != nil {
			log.Fatalf("Setup validation failed: %v", err)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Service Auth Key Management Tool")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  generate-keys <service-name> <key-dir>")
	fmt.Println("    Generate RSA key pair for a service")
	fmt.Println()
	fmt.Println("  setup-services <key-dir> <service1> <service2> ...")
	fmt.Println("    Generate keys for multiple services")
	fmt.Println()
	fmt.Println("  validate-setup <service-name> <private-key-path> <public-key-path> <key-dir> <service1> <service2> ...")
	fmt.Println("    Validate key setup and configuration")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run main.go generate-keys account-service ./keys")
	fmt.Println("  go run main.go setup-services ./keys account-service order-service user-service")
	fmt.Println("  go run main.go validate-setup account-service ./keys/account-service_private.pem ./keys/account-service_public.pem ./keys order-service user-service")
}
