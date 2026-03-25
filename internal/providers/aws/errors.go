package aws

import (
	"errors"

	"github.com/aws/smithy-go"
)

// isAccessDenied returns true if the error indicates insufficient IAM permissions.
// Discoverers should treat this as non-fatal: skip the resource type and log the error.
func isAccessDenied(err error) bool {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		switch ae.ErrorCode() {
		case "AccessDenied", "UnauthorizedOperation", "AccessDeniedException",
			"NoCredentialProviders", "InvalidClientTokenId":
			return true
		}
	}
	return false
}

// isNotFound returns true if the error indicates the resource no longer exists.
// Discoverers should silently skip these (resource was deleted between list and describe).
func isNotFound(err error) bool {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		switch ae.ErrorCode() {
		case "NoSuchBucket", "NoSuchKey", "NoSuchEntity",
			"InvalidInstanceID.NotFound", "InvalidGroupId.NotFound",
			"InvalidVpcID.NotFound", "InvalidSubnetID.NotFound",
			"NoSuchHostedZone", "ResourceNotFoundException",
			"DBInstanceNotFoundFault", "DBClusterNotFoundFault",
			"NoSuchTagSet":
			return true
		}
	}
	return false
}

// isThrottled returns true if the error is a rate-limiting response.
// The AWS SDK v2 retrier handles this automatically, but discoverers can use this
// to log throttling at debug level.
func isThrottled(err error) bool {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		switch ae.ErrorCode() {
		case "Throttling", "ThrottlingException", "RequestLimitExceeded",
			"TooManyRequestsException", "Rate exceeded":
			return true
		}
	}
	return false
}
