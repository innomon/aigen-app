package main

import (
	"archive/zip"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: bundle <dir> <output.jar> <private_key.pem>")
		os.Exit(1)
	}

	srcDir := os.Args[1]
	outPath := os.Args[2]
	keyPath := os.Args[3]

	// 1. Load Private Key
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("failed to read private key: %v", err)
	}
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		log.Fatal("failed to decode PEM block containing private key")
	}

	var priv *rsa.PrivateKey
	if block.Type == "RSA PRIVATE KEY" {
		priv, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	} else if block.Type == "PRIVATE KEY" {
		pk, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 == nil {
			var ok bool
			priv, ok = pk.(*rsa.PrivateKey)
			if !ok {
				log.Fatal("not an RSA private key")
			}
		} else {
			err = err2
		}
	} else {
		log.Fatalf("unsupported key type: %s", block.Type)
	}

	if err != nil {
		log.Fatalf("failed to parse private key: %v", err)
	}

	// 2. Create JAR (ZIP)
	outFile, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)
	defer zw.Close()

	var manifestContent string

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, _ := filepath.Rel(srcDir, path)
		if relPath == "." || relPath == "META-INF" {
			return nil
		}

		f, err := zw.Create(relPath)
		if err != nil {
			return err
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		h := sha256.New()
		tr := io.TeeReader(srcFile, h)
		_, err = io.Copy(f, tr)
		if err != nil {
			return err
		}

		manifestContent += fmt.Sprintf("Name: %s\nSHA256-Digest: %x\n\n", relPath, h.Sum(nil))
		return nil
	})

	if err != nil {
		log.Fatalf("failed to walk source directory: %v", err)
	}

	// 3. Create PLUGIN.SF (Signature File)
	sfFile, err := zw.Create("META-INF/PLUGIN.SF")
	if err != nil {
		log.Fatalf("failed to create PLUGIN.SF: %v", err)
	}
	_, err = sfFile.Write([]byte(manifestContent))
	if err != nil {
		log.Fatalf("failed to write PLUGIN.SF: %v", err)
	}

	// 4. Create PLUGIN.RSA (Signature Block)
	// For this POC, we sign the entire PLUGIN.SF content
	hashed := sha256.Sum256([]byte(manifestContent))
	signature, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
	if err != nil {
		log.Fatalf("failed to sign: %v", err)
	}

	rsaFile, err := zw.Create("META-INF/PLUGIN.RSA")
	if err != nil {
		log.Fatalf("failed to create PLUGIN.RSA: %v", err)
	}
	_, err = rsaFile.Write(signature)
	if err != nil {
		log.Fatalf("failed to write PLUGIN.RSA: %v", err)
	}

	fmt.Printf("Plugin bundled and signed: %s\n", outPath)
}
