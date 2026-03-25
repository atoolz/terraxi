package aws

import (
	"fmt"
	"testing"

	"github.com/aws/smithy-go"
)

func apiError(code, msg string) error {
	return &smithy.GenericAPIError{Code: code, Message: msg}
}

func TestIsAccessDenied(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{apiError("AccessDenied", "not allowed"), true},
		{apiError("UnauthorizedOperation", "no"), true},
		{apiError("AccessDeniedException", "denied"), true},
		{apiError("InvalidClientTokenId", "bad token"), true},
		{apiError("NoCredentialProviders", "no creds"), false},
		{apiError("NoSuchBucket", "not found"), false},
		{fmt.Errorf("random error"), false},
	}
	for _, tt := range tests {
		got := isAccessDenied(tt.err)
		if got != tt.want {
			t.Errorf("isAccessDenied(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{apiError("NoSuchBucket", ""), true},
		{apiError("NoSuchEntity", ""), true},
		{apiError("InvalidInstanceID.NotFound", ""), true},
		{apiError("NoSuchTagSet", ""), true},
		{apiError("ResourceNotFoundException", ""), true},
		{apiError("DBInstanceNotFoundFault", ""), true},
		{apiError("AccessDenied", ""), false},
		{fmt.Errorf("network error"), false},
	}
	for _, tt := range tests {
		got := isNotFound(tt.err)
		if got != tt.want {
			t.Errorf("isNotFound(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestIsThrottled(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{apiError("Throttling", "slow down"), true},
		{apiError("ThrottlingException", ""), true},
		{apiError("TooManyRequestsException", ""), true},
		{apiError("RequestLimitExceeded", ""), true},
		{apiError("AccessDenied", ""), false},
	}
	for _, tt := range tests {
		got := isThrottled(tt.err)
		if got != tt.want {
			t.Errorf("isThrottled(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}
