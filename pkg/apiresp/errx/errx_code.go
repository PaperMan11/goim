package errx

// ErrCodeUnknown represents the error code when code is not parsed or parsed code equals 0.
const ErrCodeUnknown = 1000

// Error codes for various error scenarios.
const (
	ErrCodeFormattingError      = 10001 // Error in formatting
	ErrCodeHasRegistered        = 10002 // user has already registered
	ErrCodeNotRegistered        = 10003 // user is not registered
	ErrCodePasswordError        = 10004 // Password error
	ErrCodeGetIMTokenError      = 10005 // Error in getting IM token
	ErrCodeRepeatSendCode       = 10006 // Repeat sending code
	ErrCodeMailSendCodeErr      = 10007 // Error in sending code via email
	ErrCodeSmsSendCodeErr       = 10008 // Error in sending code via SMS
	ErrCodeCodeInvalidOrExpired = 10009 // Code is invalid or expired
	ErrCodeRegisterFailed       = 10010 // Registration failed
	ErrCodeResetPasswordFailed  = 10011 // Resetting password failed
	ErrCodeRegisterLimit        = 10012 // Registration limit exceeded
	ErrCodeLoginLimit           = 10013 // Login limit exceeded
	ErrCodeInvitationError      = 10014 // Error in invitation
	ErrCodeLogoutError          = 10015 // User has logged out
	ErrCodeHandshakeError       = 10016 // Handshake error
)

// General error codes.
const (
	SuccessCode = 0

	ErrCodeInternalError = 90001 // Internal error
	ErrCodeDatabaseError = 90002 // Database error (redis/mysql, etc.)
	ErrCodeNetworkError  = 90004 // Network error
	ErrCodeDataError     = 90007 // Data error

	ErrCodeCallbackError = 80000

	// General error codes.
	ErrCodeServerInternalError   = 500  // Server internal error
	ErrCodeArgsError             = 1001 // Input parameter error
	ErrCodeNoPermissionError     = 1002 // Insufficient permission
	ErrCodeDuplicateKeyError     = 1003
	ErrCodeRecordNotFoundError   = 1004 // Record does not exist
	ErrCodeUnknownMessageError   = 1005 // Unknown message error
	ErrCodeSecretNotChangedError = 1050 // secret not changed

	// Account error codes.
	ErrCodeUserIDNotFoundError    = 1101 // UserID does not exist or is not registered
	ErrCodeRegisteredAlreadyError = 1102 // user is already registered

	// Group error codes.
	ErrCodeGroupIDNotFoundError  = 1201 // GroupID does not exist
	ErrCodeGroupIDExisted        = 1202 // GroupID already exists
	ErrCodeNotInGroupYetError    = 1203 // Not in the group yet
	ErrCodeDismissedAlreadyError = 1204 // Group has already been dismissed
	ErrCodeGroupTypeNotSupport   = 1205
	ErrCodeGroupRequestHandled   = 1206
	ErrCodeGroupNotFoundError    = 1207 // Group does not exist

	// Relationship error codes.
	ErrCodeCanNotAddYourselfError   = 1301 // Cannot add yourself as a friend
	ErrCodeBlockedByPeer            = 1302 // Blocked by the peer
	ErrCodeNotPeersFriend           = 1303 // Not the peer's friend
	ErrCodeRelationshipAlreadyError = 1304 // Already in a friend relationship
	ErrCodeFriendRequestHandled     = 1305 // Friend request has already been handled

	// Message error codes.
	ErrCodeMessageHasReadDisable = 1401
	ErrCodeMutedInGroup          = 1402 // Member muted in the group
	ErrCodeMutedGroup            = 1403 // Group is muted
	ErrCodeMsgAlreadyRevoke      = 1404 // Message already revoked

	// Token error codes.
	ErrCodeTokenExpiredError     = 1501
	ErrCodeTokenInvalidError     = 1502
	ErrCodeTokenMalformedError   = 1503
	ErrCodeTokenNotValidYetError = 1504
	ErrCodeTokenUnknownError     = 1505
	ErrCodeTokenKickedError      = 1506
	ErrCodeTokenNotExistError    = 1507

	// Long connection gateway error codes.
	ErrCodeConnOverMaxNumLimit    = 1601
	ErrCodeConnArgsError          = 1602
	ErrCodePushMsgError           = 1603
	ErrCodeIOSBackgroundPushError = 1604
	ErrCodeConnResetError         = 1605 // Connection reset

	// S3 error codes.
	ErrCodeFileUploadedExpiredError = 1701 // Upload expired
)
