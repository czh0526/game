package aries

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/czh0526/game/server/internal/storage/mysqlstore"
)

// AriesService wraps simplified Aries functionality for DID operations
type AriesService struct {
	storageProvider *mysqlstore.Provider
}

// Config Aries configuration
type Config struct {
	MySQLDSN string
	Label    string
}

// Doc simplified DID document structure
type Doc struct {
	Context            []string                  `json:"@context"`
	ID                 string                    `json:"id"`
	VerificationMethod  []VerificationMethod       `json:"verificationMethod"`
	AssertionMethod    []VerificationRelationship `json:"assertionMethod"`
}

// VerificationMethod represents a verification method
type VerificationMethod struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Controller string         `json:"controller"`
	PublicKey  *PublicKeyJwk  `json:"publicKeyJwk"`
}

// PublicKeyJwk represents a public key in JWK format
type PublicKeyJwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
}

// VerificationRelationship represents a verification relationship
type VerificationRelationship struct {
	VerificationMethod VerificationMethod `json:"verificationMethod"`
}

// NewAriesService creates a new Aries service
func NewAriesService(config *Config) (*AriesService, error) {
	// Create MySQL storage provider
	storageProvider, err := mysqlstore.NewProvider(config.MySQLDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage provider: %w", err)
	}

	return &AriesService{
		storageProvider: storageProvider,
	}, nil
}

// CreatePlayerDID creates a new player DID using Aries framework
func (s *AriesService) CreatePlayerDID(gameID, playerID string) (*CreatePlayerDIDResponse, error) {
	// Generate Ed25519 key pair
	publicKeyBytes, privateKeyBytes, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Encode keys as hex
	publicKey := hex.EncodeToString(publicKeyBytes)
	privateKey := hex.EncodeToString(privateKeyBytes)

	// Create DID string: did:player:{gameID}:{playerID}
	didStr := fmt.Sprintf("did:player:%s:%s", gameID, playerID)

	// Create DID document
	doc := &Doc{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://game.example.com/contexts/player/v1",
		},
		ID: didStr,
		VerificationMethod: []VerificationMethod{
			{
				ID:         fmt.Sprintf("%s#key-1", didStr),
				Type:       "Ed25519VerificationKey2018",
				Controller: didStr,
				PublicKey: &PublicKeyJwk{
					Kty: "OKP",
					Crv: "Ed25519",
					X:   publicKey,
				},
			},
		},
		AssertionMethod: []VerificationRelationship{
			{
				VerificationMethod: VerificationMethod{
					ID: fmt.Sprintf("%s#key-1", didStr),
				},
			},
		},
	}

	// Store DID document in MySQL
	store, err := s.storageProvider.OpenStore("did_store")
	if err != nil {
		return nil, fmt.Errorf("failed to open DID store: %w", err)
	}

	docJSON, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DID document: %w", err)
	}

	if err := store.Put(didStr, docJSON); err != nil {
		return nil, fmt.Errorf("failed to store DID document: %w", err)
	}

	return &CreatePlayerDIDResponse{
		DID:       didStr,
		DIDDoc:    doc,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// ResolveDID resolves a DID from storage
func (s *AriesService) ResolveDID(didStr string) (*Doc, error) {
	store, err := s.storageProvider.OpenStore("did_store")
	if err != nil {
		return nil, fmt.Errorf("failed to open DID store: %w", err)
	}

	data, err := store.Get(didStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get DID document: %w", err)
	}

	var doc Doc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse DID document: %w", err)
	}

	return &doc, nil
}

// CreatePlayerDIDResponse response for creating player DID
type CreatePlayerDIDResponse struct {
	DID       string
	DIDDoc    *Doc
	PublicKey string
	PrivateKey string
}

// Close closes the Aries service
func (s *AriesService) Close() error {
	return s.storageProvider.Close()
}
