package models

import (
	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

// Response is the normalized single-secret read response from Vault.
type Response struct {
	Data          map[string]interface{} `json:"data"`
	Metadata      *Metadata              `json:"metadata"`
	LeaseID       string                 `json:"lease_id"`
	LeaseDuration int                    `json:"lease_duration"`
	Renewable     bool                   `json:"renewable"`
}

// Convert converts a raw Vault response into a normalized Response.
func Convert(raw *vault.Response[map[string]interface{}]) *Response {
	return &Response{
		Data: raw.Data,
		Metadata: &Metadata{
			Warnings:  raw.Warnings,
			RequestID: raw.RequestID,
		},
		LeaseID:       raw.LeaseID,
		LeaseDuration: raw.LeaseDuration,
		Renewable:     raw.Renewable,
	}
}

// ListResponse is the normalized key-list response from Vault.
type ListResponse struct {
	Data          []string  `json:"data"`
	Metadata      *Metadata `json:"metadata"`
	LeaseID       string    `json:"lease_id"`
	LeaseDuration int       `json:"lease_duration"`
	Renewable     bool      `json:"renewable"`
}

// ConvertFromList converts a raw Vault list response into a normalized ListResponse.
func ConvertFromList(raw *vault.Response[schema.StandardListResponse]) *ListResponse {
	return &ListResponse{
		Data: raw.Data.Keys,
		Metadata: &Metadata{
			Warnings:  raw.Warnings,
			RequestID: raw.RequestID,
		},
		LeaseID:       raw.LeaseID,
		LeaseDuration: raw.LeaseDuration,
		Renewable:     raw.Renewable,
	}
}

// Metadata carries Vault request metadata alongside a response.
type Metadata struct {
	Warnings  []string `json:"warnings"`
	RequestID string   `json:"request_id"`
}
