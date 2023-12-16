package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/open-policy-agent/opa/rego"
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
	token.Header["kid"] = "54bb2165-71e1-41a6-af3e-7da4a0e1e2c1"

	file, err := os.Open("zarf/keys/54bb2165-71e1-41a6-af3e-7da4a0e1e2c1.pem")
	if err != nil {
		return fmt.Errorf("opening key file: %w", err)
	}
	defer file.Close()

	pemData, err := io.ReadAll(io.LimitReader(file, 1024*1024))
	if err != nil {
		return fmt.Errorf("reading auth private key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemData)
	if err != nil {
		return fmt.Errorf("parsing auth private key: %w", err)
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

	pemBlock := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}
	// Construct a PEM block for the public key.

	fmt.Println("\n*********** PUBLIC KEY ************")

	if err := pem.Encode(os.Stdout, &pemBlock); err != nil {
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

	// use code to check if jwt is valid
	if !tkn.Valid {
		return fmt.Errorf("token not valid")
	}

	fmt.Println("\n*********** TOKEN VALID BY Go CODE ************")

	// -------------------------------------------------------------------------
	// this is hack we are using buffer to convert public key to binary string, so that you can use it to validate token on OPA
	// no need to read public from file
	var b bytes.Buffer
	if err := pem.Encode(&b, &pemBlock); err != nil {
		return fmt.Errorf("encoding to public file: %w", err)
	}

	ctx := context.Background()
	if err := opaPolicyEvaluationAuthenticate(ctx, b.String(), str, clm.Issuer); err != nil {
		return fmt.Errorf("OPS authentication failed: %w", err)
	}
	fmt.Println("\n*********** authentication VALIDATED BY OPA ************")

	// -------------------------------------------------------------------------
	if err := opaPolicyEvaluationAuthorize(ctx); err != nil {
		return fmt.Errorf("OPS authorization failed: %w", err)
	}

	fmt.Println("\n*********** authorization VALIDATED BY OPA ************")

	// -------------------------------------------------------------------------
	fmt.Println("\n*********** Unmarshal jwt to claim obj ************")
	fmt.Printf("\n%#v\n", clm)

	return nil

}

// Core OPA policies.
var (
	//go:embed rego/authentication.rego
	opaAuthentication string
	//go:embed rego/authorization.rego
	opaAuthorization string
)

func opaPolicyEvaluationAuthenticate(ctx context.Context, pem string, tokenString string, issuer string) error {
	const rule = "allow"
	const opaPackage string = "shawn.rego"

	query := fmt.Sprintf("x = data.%s.%s", opaPackage, rule)

	q, err := rego.New(
		rego.Query(query),
		rego.Module("policy.rego", opaAuthentication),
	).PrepareForEval(ctx)
	if err != nil {
		return err
	}

	input := map[string]any{
		"Key":   pem,
		"Token": tokenString,
		"ISS":   issuer,
	}

	results, err := q.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if len(results) == 0 {
		return errors.New("no results")
	}

	result, ok := results[0].Bindings["x"].(bool)
	if !ok || !result {
		return fmt.Errorf("bindings results[%v] ok[%v]", results, ok)
	}

	return err
}

func opaPolicyEvaluationAuthorize(ctx context.Context) error {
	const rule = "ruleAdminOrSubject"
	const opaPackage string = "shawn.rego"

	query := fmt.Sprintf("x = data.%s.%s", opaPackage, rule)

	q, err := rego.New(
		rego.Query(query),
		rego.Module("policy.rego", opaAuthorization),
	).PrepareForEval(ctx)
	if err != nil {
		return err
	}

	// ruleAdminOrSubject rule defined in rego/authorization.rego
	input := map[string]any{
		"Roles":   []string{"USER"},
		"Subject": "1234567",
		"UserID":  "1234567",
	}

	results, err := q.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	if len(results) == 0 {
		return errors.New("no results")
	}

	fmt.Printf("\nresults set is: %#v\n", results)

	result, ok := results[0].Bindings["x"].(bool)
	if !ok || !result {
		return fmt.Errorf("bindings results[%v] ok[%v]", results, ok)
	}

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
