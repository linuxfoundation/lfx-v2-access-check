// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
)

// AccessCheckClient handles the NATS protocol for access checking and tuple reading.
// It owns the message build/parse logic and the two NATS subjects, with no knowledge
// of HTTP, Goa types, or authentication.
type AccessCheckClient struct {
	messagingRepo contracts.MessagingRepository
}

// NewAccessCheckClient creates a new AccessCheckClient backed by the given messaging repository.
func NewAccessCheckClient(messagingRepo contracts.MessagingRepository) *AccessCheckClient {
	return &AccessCheckClient{messagingRepo: messagingRepo}
}

// HealthCheck checks the messaging repository health, returning ErrMessagingRepoNotInit
// if the repository was never provided.
func (c *AccessCheckClient) HealthCheck(ctx context.Context) error {
	if c.messagingRepo == nil {
		return constants.ErrMessagingRepoNotInit
	}
	return c.messagingRepo.HealthCheck(ctx)
}

// CheckAccess sends resource-action pairs to fga-sync via NATS and returns the
// newline-delimited result lines in the same order as the request.
// principal must be non-empty; empty resource strings are skipped.
func (c *AccessCheckClient) CheckAccess(ctx context.Context, principal string, resources []string) ([]string, error) {
	if principal == "" {
		return nil, constants.ErrPrincipalRequired
	}

	if len(resources) == 0 {
		return []string{}, nil
	}

	message := c.buildMessage(principal, resources)
	if message == "" {
		return []string{}, nil
	}

	responseData, err := c.messagingRepo.Request(ctx, constants.AccessCheckSubject, []byte(message), constants.DefaultNATSTimeout)
	if err != nil {
		return nil, fmt.Errorf("NATS request to subject %s failed: %w", constants.AccessCheckSubject, err)
	}

	return c.parseResponse(responseData)
}

// ReadTuples fetches the direct OpenFGA tuples for a principal via NATS.
func (c *AccessCheckClient) ReadTuples(ctx context.Context, principal string, objectType string) ([]string, error) {
	reqPayload, err := json.Marshal(readTuplesRequest{
		User:       constants.UserTypePrefix + principal,
		ObjectType: objectType,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to build read tuples request: %v", constants.ErrUnexpectedResponse, err)
	}

	responseData, err := c.messagingRepo.Request(ctx, constants.ReadTuplesSubject, reqPayload, constants.DefaultNATSTimeout)
	if err != nil {
		return nil, fmt.Errorf("NATS request to subject %s failed: %w", constants.ReadTuplesSubject, err)
	}

	var resp readTuplesResponse
	if err := json.Unmarshal(responseData, &resp); err != nil {
		return nil, fmt.Errorf("%w: failed to parse read tuples response: %v", constants.ErrUnexpectedResponse, err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("message to subject %s failed with FGA error: %s", constants.ReadTuplesSubject, resp.Error)
	}

	if resp.Results == nil {
		return []string{}, nil
	}
	return resp.Results, nil
}

// buildMessage constructs the newline-separated plaintext NATS payload for
// access-check requests in the form "object#relation@user:principal".
// Empty resource strings are skipped.
func (c *AccessCheckClient) buildMessage(principal string, resources []string) string {
	var builder strings.Builder

	totalCapacity := 0
	for _, resource := range resources {
		if resource != "" {
			// resource + "@user:" + principal + newline
			totalCapacity += len(resource) + len(constants.UserRelationPrefix) + len(principal) + 1
		}
	}

	if totalCapacity > 0 {
		builder.Grow(totalCapacity)
	}

	for _, resource := range resources {
		if resource == "" {
			continue
		}
		builder.WriteString(resource)
		builder.WriteString(constants.UserRelationPrefix)
		builder.WriteString(principal)
		builder.WriteByte('\n')
	}

	message := builder.String()
	if len(message) > 0 && message[len(message)-1] == '\n' {
		message = message[:len(message)-1]
	}

	return message
}

// parseResponse validates and splits the NATS access-check reply into result lines.
// A space appearing in the first DefaultResponseSanityCheckBytes bytes indicates an
// error message from fga-sync rather than a valid result payload.
func (c *AccessCheckClient) parseResponse(responseData []byte) ([]string, error) {
	topRange := constants.DefaultResponseSanityCheckBytes
	if len(responseData) < topRange {
		topRange = len(responseData)
	}
	if bytes.Contains(responseData[:topRange], []byte(" ")) {
		return nil, fmt.Errorf("%w: response_preview=%q", constants.ErrUnexpectedResponse, string(responseData[:topRange]))
	}

	lines := bytes.Split(responseData, []byte("\n"))
	results := make([]string, 0, len(lines))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		results = append(results, string(line))
	}
	return results, nil
}

// readTuplesRequest is the JSON payload sent to fga-sync over NATS.
type readTuplesRequest struct {
	User       string `json:"user"`
	ObjectType string `json:"object_type"`
}

// readTuplesResponse is the JSON response received from fga-sync over NATS.
type readTuplesResponse struct {
	Results []string `json:"results,omitempty"`
	Error   string   `json:"error,omitempty"`
}
