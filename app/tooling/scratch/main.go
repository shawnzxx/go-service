package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func main() {
	err := genToken()

	// test generate private and public pair of keys file on local
	// err := genKey()

	if err != nil {
		log.Fatalln(err)
	}
}

// genToken generates a JWT token and writes it to the console.
func genToken() error {

	// Generating a token requires defining a set of claims. In this applications
	// case, we only care about defining the subject and the user in question and
	// the roles they have on the database. This token will expire in a year.
	//
	// iss (issuer): Issuer of the JWT
	// sub (subject): Subject of the JWT (the user)
	// aud (audience): Recipient for which the JWT is intended
	// exp (expiration time): Time after which the JWT expires
	// nbf (not before time): Time before which the JWT must not be accepted for processing
	// iat (issued at time): Time at which the JWT was issued; can be used to determine age of the JWT
	// jti (JWT ID): Unique identifier; can be used to prevent the JWT from being replayed (allows a token to be used only once)
	claims := struct {
		jwt.RegisteredClaims
		Roles []string
	}{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "12345678",
			Issuer:    "service project",
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(8760 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
		Roles: []string{"ADMIN"},
	}

	method := jwt.GetSigningMethod(jwt.SigningMethodRS256.Name)
	token := jwt.NewWithClaims(method, claims)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}

	// use private ket to sign token
	str, err := token.SignedString(privateKey)
	if err != nil {
		return fmt.Errorf("signing token: %w", err)
	}

	fmt.Println("\n*********** TOKEN ************")
	fmt.Println(str)
	fmt.Print("\n")

	// -------------------------------------------------------------------------
	// dump public key out to console, so that you can use public key to validate token on jwt.io

	// Marshal the public key from the private key to PKIX.
	asn1Bytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("marshaling public key: %w", err)
	}

	publicBlock := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}
	// Construct a PEM block for the public key.

	fmt.Println("\n*********** PUBLIC KEY ************")

	if err := pem.Encode(os.Stdout, &publicBlock); err != nil {
		return fmt.Errorf("encoding to public file: %w", err)
	}

	fmt.Print("\n")

	// -------------------------------------------------------------------------
	// beside above print public key and validate token on jwt.io, you can also validate token in code

	// construct the parser
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name}))

	// create claim holder to assign result later
	var clm struct {
		jwt.RegisteredClaims
		Roles []string
	}

	// function input is parsed token string, but unverified token
	kf := func(jwt *jwt.Token) (interface{}, error) {
		return &privateKey.PublicKey, nil
	}

	// parse token string back to claim obj
	tkn, err := parser.ParseWithClaims(str, &clm, kf)
	if err != nil {
		return fmt.Errorf("parsing with claims: %w", err)
	}

	if !tkn.Valid {
		return fmt.Errorf("token not valid")
	}

	fmt.Println("\n*********** TOKEN VALID ************")
	fmt.Printf("%#v\n", clm)

	return nil

}

// genKey generates a private and public key pair and writes them to private.pem and public.pem respectively.
func genKey() error {

	// Generate a new private key.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}

	// Create a file for the private key information in PEM form.
	privateFile, err := os.Create("private.pem")
	if err != nil {
		return fmt.Errorf("creating private file: %w", err)
	}
	defer privateFile.Close()

	// Construct a PEM block for the private key.
	privateBlock := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Write the private key to the private key file.
	if err := pem.Encode(privateFile, &privateBlock); err != nil {
		return fmt.Errorf("encoding to private file: %w", err)
	}

	// -------------------------------------------------------------------------

	// Create a file for the public key information in PEM form.
	publicFile, err := os.Create("public.pem")
	if err != nil {
		return fmt.Errorf("creating public file: %w", err)
	}
	defer publicFile.Close()

	// Marshal the public key from the private key to PKIX.
	asn1Bytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("marshaling public key: %w", err)
	}

	// Construct a PEM block for the public key.
	publicBlock := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}

	// Write the public key to the public key file.
	if err := pem.Encode(publicFile, &publicBlock); err != nil {
		return fmt.Errorf("encoding to public file: %w", err)
	}

	fmt.Println("private and public key files generated")

	return nil
}