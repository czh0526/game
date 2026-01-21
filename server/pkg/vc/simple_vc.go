package vc

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SimpleCredential 简化的可验证凭证
type SimpleCredential struct {
	Context           []string               `json:"@context"`
	ID                string                 `json:"id"`
	Type              []string               `json:"type"`
	Issuer            string                 `json:"issuer"`
	IssuanceDate      time.Time              `json:"issuanceDate"`
	ExpirationDate    *time.Time             `json:"expirationDate,omitempty"`
	CredentialSubject CredentialSubject      `json:"credentialSubject"`
	Proof             *Proof                 `json:"proof,omitempty"`
}

// CredentialSubject 凭证主体
type CredentialSubject struct {
	ID          string                 `json:"id"`
	PlayerID    string                 `json:"playerId"`
	GameID      string                 `json:"gameId"`
	Achievement string                 `json:"achievement,omitempty"`
	Level       int                    `json:"level,omitempty"`
	Score       int                    `json:"score,omitempty"`
	Skills      []string               `json:"skills,omitempty"`
	Items       []string               `json:"items,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
}

// Proof 证明
type Proof struct {
	Type               string    `json:"type"`
	Created            time.Time `json:"created"`
	VerificationMethod string    `json:"verificationMethod"`
	ProofPurpose       string    `json:"proofPurpose"`
	ProofValue         string    `json:"proofValue"`
}

// IssueCredential 颁发凭证
func IssueCredential(issuerDID, subjectDID string, credType string, subject CredentialSubject) (*SimpleCredential, error) {
	credentialID := fmt.Sprintf("urn:uuid:%s", uuid.New().String())
	now := time.Now()

	credential := &SimpleCredential{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
			"https://game.example.com/contexts/credentials/v1",
		},
		ID:     credentialID,
		Type:   []string{"VerifiableCredential", credType},
		Issuer: issuerDID,
		IssuanceDate: now,
		CredentialSubject: subject,
	}

	// 设置凭证主体ID
	credential.CredentialSubject.ID = subjectDID

	return credential, nil
}

// VerifyCredential 验证凭证
func VerifyCredential(credential *SimpleCredential, issuerDID string) (bool, string) {
	// 基础验证
	if credential == nil {
		return false, "credential is nil"
	}

	// 检查颁发者
	if credential.Issuer != issuerDID {
		return false, "invalid issuer"
	}

	// 检查过期时间
	if credential.ExpirationDate != nil && credential.ExpirationDate.Before(time.Now()) {
		return false, "credential has expired"
	}

	// 检查颁发时间
	if credential.IssuanceDate.After(time.Now()) {
		return false, "credential issued in the future"
	}

	return true, "credential is valid"
}

// ToJSON 转换为JSON
func (c *SimpleCredential) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// FromJSON 从JSON解析
func CredentialFromJSON(data []byte) (*SimpleCredential, error) {
	var credential SimpleCredential
	err := json.Unmarshal(data, &credential)
	return &credential, err
}