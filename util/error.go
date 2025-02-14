package util

import (
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/guygma/goKCL/record"
)

// ErrorCode is unified definition of numerical error codes
type ErrorCode int32

// pre-defined error codes
const (
	// System Wide      41000 - 42000
	KinesisClientLibError ErrorCode = 41000

	// KinesisClientLibrary Retryable Errors 41001 - 41100
	KinesisClientLibRetryableError ErrorCode = 41001

	KinesisClientLibIOError         ErrorCode = 41002
	BlockedOnParentShardError       ErrorCode = 41003
	KinesisClientLibDependencyError ErrorCode = 41004
	ThrottlingError                 ErrorCode = 41005

	// KinesisClientLibrary NonRetryable Errors 41100 - 41200
	KinesisClientLibNonRetryableException ErrorCode = 41100

	InvalidStateError ErrorCode = 41101
	ShutdownError     ErrorCode = 41102

	// Kinesis Lease Errors 41200 - 41300
	LeasingError ErrorCode = 41200

	LeasingInvalidStateError          ErrorCode = 41201
	LeasingDependencyError            ErrorCode = 41202
	LeasingProvisionedThroughputError ErrorCode = 41203

	// Misc Errors 41300 - 41400
	// NotImplemented
	KinesisClientLibNotImplemented ErrorCode = 41301

	// Error indicates passing illegal or inappropriate argument
	IllegalArgumentError ErrorCode = 41302
)

var errorMap = map[ErrorCode]ClientLibraryError{
	KinesisClientLibError: {ErrorCode: KinesisClientLibError, Retryable: true, Status: http.StatusServiceUnavailable, Msg: "Top level error of Kinesis Client Library"},

	// Retryable
	KinesisClientLibRetryableError:  {ErrorCode: KinesisClientLibRetryableError, Retryable: true, Status: http.StatusServiceUnavailable, Msg: "Retryable exceptions (e.g. transient errors). The request/operation is expected to succeed upon (back off and) retry."},
	KinesisClientLibIOError:         {ErrorCode: KinesisClientLibIOError, Retryable: true, Status: http.StatusServiceUnavailable, Msg: "Error in reading/writing information (e.g. shard information from Kinesis may not be current/complete)."},
	BlockedOnParentShardError:       {ErrorCode: BlockedOnParentShardError, Retryable: true, Status: http.StatusServiceUnavailable, Msg: "Cannot start processing data for a shard because the data from the parent shard has not been completely processed (yet)."},
	KinesisClientLibDependencyError: {ErrorCode: KinesisClientLibDependencyError, Retryable: true, Status: http.StatusServiceUnavailable, Msg: "Cannot talk to its dependencies (e.g. fetching data from Kinesis, DynamoDB table reads/writes, emitting metrics to CloudWatch)."},
	ThrottlingError:                 {ErrorCode: ThrottlingError, Retryable: true, Status: http.StatusTooManyRequests, Msg: "Requests are throttled by a service (e.g. DynamoDB when storing a checkpoint)."},

	// Non-Retryable
	KinesisClientLibNonRetryableException: {ErrorCode: KinesisClientLibNonRetryableException, Retryable: false, Status: http.StatusServiceUnavailable, Msg: "Non-retryable exceptions. Simply retrying the same request/operation is not expected to succeed."},
	InvalidStateError:                     {ErrorCode: InvalidStateError, Retryable: false, Status: http.StatusServiceUnavailable, Msg: "Kinesis Library has issues with internal state (e.g. DynamoDB table is not found)."},
	ShutdownError:                         {ErrorCode: ShutdownError, Retryable: false, Status: http.StatusServiceUnavailable, Msg: "The RecordProcessor instance has been shutdown (e.g. and attempts a checkpiont)."},

	// Leasing
	LeasingError:                      {ErrorCode: LeasingError, Retryable: true, Status: http.StatusServiceUnavailable, Msg: "Top-level error type for the leasing code."},
	LeasingInvalidStateError:          {ErrorCode: LeasingInvalidStateError, Retryable: true, Status: http.StatusServiceUnavailable, Msg: "Error in a lease operation has failed because DynamoDB is an invalid state"},
	LeasingDependencyError:            {ErrorCode: LeasingDependencyError, Retryable: true, Status: http.StatusServiceUnavailable, Msg: "Error in a lease operation has failed because a dependency of the leasing system has failed."},
	LeasingProvisionedThroughputError: {ErrorCode: LeasingProvisionedThroughputError, Retryable: false, Status: http.StatusServiceUnavailable, Msg: "Error in a lease operation has failed due to lack of provisioned throughput for a DynamoDB table."},

	// IllegalArgumentError
	IllegalArgumentError: {ErrorCode: IllegalArgumentError, Retryable: false, Status: http.StatusBadRequest, Msg: "Error indicates that a method has been passed an illegal or inappropriate argument."},

	// Not Implemented
	KinesisClientLibNotImplemented: {ErrorCode: KinesisClientLibNotImplemented, Retryable: false, Status: http.StatusNotImplemented, Msg: "Not Implemented"},
}

// Message returns the message of the error code
func (c ErrorCode) Message() string {
	return errorMap[c].Msg
}

// MakeErr makes an error with default message
func (c ErrorCode) MakeErr() *ClientLibraryError {
	e := errorMap[c]
	return &e
}

// MakeError makes an error with message and data
func (c ErrorCode) MakeError(detail string) error {
	e := errorMap[c]
	return e.WithDetail(detail)
}

// ClientLibraryError is unified error
type ClientLibraryError struct {
	// ErrorCode is the numerical error code.
	ErrorCode `json:"code"`
	// Retryable is a bool flag to indicate the whether the error is retryable or not.
	Retryable bool `json:"tryable"`
	// Status is the HTTP status code.
	Status int `json:"status"`
	// Msg provides a terse description of the error. Its value is defined in errorMap.
	Msg string `json:"msg"`
	// Detail provides a detailed description of the error. Its value is set using WithDetail.
	Detail string `json:"detail"`
}

// Error implements error
func (e *ClientLibraryError) Error() string {
	var prefix string
	if e.Retryable {
		prefix = "Retryable"
	} else {
		prefix = "NonRetryable"
	}
	msg := fmt.Sprintf("%v Error [%d]: %s", prefix, int32(e.ErrorCode), e.Msg)
	if e.Detail != "" {
		msg = fmt.Sprintf("%s, detail: %s", msg, e.Detail)
	}
	return msg
}

// WithMsg overwrites the default error message
func (e *ClientLibraryError) WithMsg(format string, v ...interface{}) *ClientLibraryError {
	e.Msg = fmt.Sprintf(format, v...)
	return e
}

// WithDetail adds a detailed message to error
func (e *ClientLibraryError) WithDetail(format string, v ...interface{}) *ClientLibraryError {
	if len(e.Detail) == 0 {
		e.Detail = fmt.Sprintf(format, v...)
	} else {
		e.Detail += ", " + fmt.Sprintf(format, v...)
	}
	return e
}

// WithCause adds CauseBy to error
func (e *ClientLibraryError) WithCause(err error) *ClientLibraryError {
	if err != nil {
		// Store error message in Detail, so the info can be preserved
		// when CascadeError is marshaled to json.
		if len(e.Detail) == 0 {
			e.Detail = err.Error()
		} else {
			e.Detail += ", cause: " + err.Error()
		}
	}
	return e
}

const (
	/**
	 * Indicates that the entire application is being shutdown, and if desired the record processor will be given a
	 * final chance to checkpoint. This state will not trigger a direct call to
	 * {@link com.amazonaws.services.kinesis.clientlibrary.interfaces.v2.IRecordProcessor#shutdown(ShutdownInput)}, but
	 * instead depend on a different interface for backward compatibility.
	 */
	REQUESTED ShutdownReason = iota + 1
	/**
	 * Terminate processing for this RecordProcessor (resharding use case).
	 * Indicates that the shard is closed and all record from the shard have been delivered to the application.
	 * Applications SHOULD checkpoint their progress to indicate that they have successfully processed all record
	 * from this shard and processing of child shards can be started.
	 */
	TERMINATE
	/**
	 * Processing will be moved to a different record processor (fail over, load balancing use cases).
	 * Applications SHOULD NOT checkpoint their progress (as another record processor may have already started
	 * processing data).
	 */
	ZOMBIE
)

// Containers for the parameters to the IRecordProcessor
type (
	/**
	 * Reason the RecordProcessor is being shutdown.
	 * Used to distinguish between a fail-over vs. a termination (shard is closed and all record have been delivered).
	 * In case of a fail over, applications should NOT checkpoint as part of shutdown,
	 * since another record processor may have already started processing record for that shard.
	 * In case of termination (resharding use case), applications SHOULD checkpoint their progress to indicate
	 * that they have successfully processed all the record (processing of child shards can then begin).
	 */
	ShutdownReason int

	ShutdownInput struct {
		ShutdownReason ShutdownReason
		Checkpointer   record.IRecordProcessorCheckpointer
	}
)

var shutdownReasonMap = map[ShutdownReason]*string{
	REQUESTED: aws.String("REQUESTED"),
	TERMINATE: aws.String("TERMINATE"),
	ZOMBIE:    aws.String("ZOMBIE"),
}

func ShutdownReasonMessage(reason ShutdownReason) *string {
	return shutdownReasonMap[reason]
}
