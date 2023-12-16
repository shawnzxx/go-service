// Package auth provides authentication and authorization support.
// Authentication: You are who you say you are.
// Authorization:  You have permission to do what you are requesting to do.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v4"
	"github.com/open-policy-agent/opa/rego"
	"github.com/shawnzxx/service/business/core/user"
	"go.uber.org/zap"
)

// ErrForbidden is returned when a auth issue is identified.
var ErrForbidden = errors.New("attempted action is not allowed")

// Claims represents the authorization claims transmitted via a JWT.
type Claims struct {
	jwt.RegisteredClaims
	Roles []user.Role `json:"roles"`
}

// KeyLookup declares a method set of behavior for looking up private and public keys for JWT use.
// here we named is as key instead of pem, why? because we are using OPA which needs pem format for the public key,
// if in future we are not use OPA, for example jwt format, then we can rename it
type KeyLookup interface {
	PrivateKey(kid string) (key string, err error)
	PublicKey(kid string) (key string, err error)
}

// Config represents information required to initialize auth.
type Config struct {
	Log       *zap.SugaredLogger
	KeyLookup KeyLookup
	Issuer    string
}

// Auth is used to authenticate clients. It can generate a token for a
// set of user claims and recreate the claims by parsing the token.
type Auth struct {
	log       *zap.SugaredLogger
	keyLookup KeyLookup
	method    jwt.SigningMethod
	parser    *jwt.Parser
	issuer    string
	mu        sync.RWMutex
	cache     map[string]string
}

// New creates an Auth to support authentication/authorization.
func New(cfg Config) (*Auth, error) {
	a := Auth{
		log:       cfg.Log,
		keyLookup: cfg.KeyLookup,
		method:    jwt.GetSigningMethod(jwt.SigningMethodRS256.Name),                          // generate token with private key using RS256 algorithm
		parser:    jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})), // parser back claim obj from JWT
		issuer:    cfg.Issuer,
		cache:     make(map[string]string),
	}

	return &a, nil
}

// GenerateToken generates a signed JWT token string representing the user Claims.
func (a *Auth) GenerateToken(kid string, claims Claims) (string, error) {
	// if you want to generate token, you give kid you give claims
	token := jwt.NewWithClaims(a.method, claims)
	// put kid in to the token
	token.Header["kid"] = kid

	// do the private key lookup from that kid, may be background hit the Vault or something else, then get the private key
	privateKeyPEM, err := a.keyLookup.PrivateKey(kid)
	if err != nil {
		return "", fmt.Errorf("private key: %w", err)
	}
	// convert to pem format
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		return "", fmt.Errorf("parsing private pem: %w", err)
	}

	// sign the token
	str, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return str, nil
}

// Authenticate processes to validate the token's signature.
func (a *Auth) Authenticate(ctx context.Context, bearerToken string) (Claims, error) {
	parts := strings.Split(bearerToken, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return Claims{}, errors.New("expected authorization header format: Bearer <token>")
	}

	var claims Claims
	// parsing token back to claim, here we don't worry token if is valid ot not
	// for valid token we use OPA in later steps.
	token, _, err := a.parser.ParseUnverified(parts[1], &claims)
	if err != nil {
		return Claims{}, fmt.Errorf("error parsing token: %w", err)
	}

	// Perform an extra level of authentication verification with OPA.
	kidRaw, exists := token.Header["kid"]
	if !exists {
		return Claims{}, fmt.Errorf("kid missing from header: %w", err)
	}

	kid, ok := kidRaw.(string)
	if !ok {
		return Claims{}, fmt.Errorf("kid malformed: %w", err)
	}

	// use keyStore find back the public key
	pem, err := a.publicKeyLookup(kid)
	if err != nil {
		return Claims{}, fmt.Errorf("failed to fetch public key: %w", err)
	}

	// prepare input struct for opa to validate the token
	input := map[string]any{
		"Key":   pem,
		"Token": parts[1],
		"ISS":   a.issuer,
	}

	if err := a.opaPolicyEvaluation(ctx, opaAuthentication, RuleAuthenticate, input); err != nil {
		return Claims{}, fmt.Errorf("authentication failed : %w", err)
	}

	// TODO additional check for authentication
	//  check our the database for this user still enabled.
	
	return claims, nil
}

// Authorize attempts to authorize the user with the provided input roles, if
// none of the input roles are within the user's claims, we return an error
// otherwise the user is authorized.
func (a *Auth) Authorize(ctx context.Context, claims Claims, rule string) error {
	input := map[string]any{
		"Roles":   claims.Roles,
		"Subject": claims.Subject,
		"UserID":  claims.Subject,
	}

	if err := a.opaPolicyEvaluation(ctx, opaAuthorization, rule, input); err != nil {
		return fmt.Errorf("rego evaluation failed : %w", err)
	}

	return nil
}

// =============================================================================

// publicKeyLookup performs a lookup for the public pem for the specified kid.
func (a *Auth) publicKeyLookup(kid string) (string, error) {
	pem, err := func() (string, error) {
		a.mu.RLock()
		defer a.mu.RUnlock()

		pem, exists := a.cache[kid]
		if !exists {
			return "", errors.New("not found")
		}
		return pem, nil
	}()
	if err == nil {
		return pem, nil
	}

	pem, err = a.keyLookup.PublicKey(kid)
	if err != nil {
		return "", fmt.Errorf("fetching public key: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache[kid] = pem

	return pem, nil
}

// opaPolicyEvaluation asks opa to evaulate the token against the specified token
// policy and public key.
func (a *Auth) opaPolicyEvaluation(ctx context.Context, opaPolicy string, rule string, input any) error {
	query := fmt.Sprintf("x = data.%s.%s", opaPackage, rule)

	q, err := rego.New(
		rego.Query(query),
		rego.Module("policy.rego", opaPolicy),
	).PrepareForEval(ctx)
	if err != nil {
		return err
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

	return nil
}
