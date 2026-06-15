package errx

// ErrCodeUnknown represents the error code when code is not parsed or parsed code equals 0.
var Unknown = NewErrInfo(ErrCodeUnknown, "unknown error")

// Error codes for various error scenarios.
var (
	FormattingError      = NewErrInfo(ErrCodeFormattingError, "error in formatting")
	HasRegistered        = NewErrInfo(ErrCodeHasRegistered, "user has already registered")
	NotRegistered        = NewErrInfo(ErrCodeNotRegistered, "user is not registered")
	PasswordError        = NewErrInfo(ErrCodePasswordError, "password error")
	GetIMTokenError      = NewErrInfo(ErrCodeGetIMTokenError, "error in getting IM token")
	RepeatSendCode       = NewErrInfo(ErrCodeRepeatSendCode, "repeat sending code")
	MailSendCodeErr      = NewErrInfo(ErrCodeMailSendCodeErr, "error in sending code via email")
	SmsSendCodeErr       = NewErrInfo(ErrCodeSmsSendCodeErr, "error in sending code via SMS")
	CodeInvalidOrExpired = NewErrInfo(ErrCodeCodeInvalidOrExpired, "code is invalid or expired")
	RegisterFailed       = NewErrInfo(ErrCodeRegisterFailed, "registration failed")
	ResetPasswordFailed  = NewErrInfo(ErrCodeResetPasswordFailed, "resetting password failed")
	RegisterLimit        = NewErrInfo(ErrCodeRegisterLimit, "registration limit exceeded")
	LoginLimit           = NewErrInfo(ErrCodeLoginLimit, "login limit exceeded")
	InvitationError      = NewErrInfo(ErrCodeInvitationError, "error in invitation")
	LogoutError          = NewErrInfo(ErrCodeLogoutError, "user has logged out")
	HandshakeError       = NewErrInfo(ErrCodeHandshakeError, "handshake error")
)

// General error codes.
var (
	Success = NewErrInfo(SuccessCode, "success")

	InternalError = NewErrInfo(ErrCodeInternalError, "internal error")
	DatabaseError = NewErrInfo(ErrCodeDatabaseError, "database error (redis/mysql, etc.)")
	NetworkError  = NewErrInfo(ErrCodeNetworkError, "network error")
	DataError     = NewErrInfo(ErrCodeDataError, "data error")

	CallbackError = NewErrInfo(ErrCodeCallbackError, "callback error")

	// General error codes.
	ServerInternalError   = NewErrInfo(ErrCodeServerInternalError, "server internal error")
	ArgsError             = NewErrInfo(ErrCodeArgsError, "input parameter error")
	NoPermissionError     = NewErrInfo(ErrCodeNoPermissionError, "insufficient permission")
	DuplicateKeyError     = NewErrInfo(ErrCodeDuplicateKeyError, "duplicate key")
	RecordNotFoundError   = NewErrInfo(ErrCodeRecordNotFoundError, "record does not exist")
	UnknownMessageError   = NewErrInfo(ErrCodeUnknownMessageError, "unknown message error")
	SecretNotChangedError = NewErrInfo(ErrCodeSecretNotChangedError, "secret not changed")

	// Account error codes.
	UserIDNotFoundError        = NewErrInfo(ErrCodeUserIDNotFoundError, "UserID does not exist or not registered")
	UserRegisteredAlreadyError = NewErrInfo(ErrCodeRegisteredAlreadyError, "user is already registered")

	// Group error codes.
	IDNotFoundError       = NewErrInfo(ErrCodeGroupIDNotFoundError, "GroupID does not exist")
	GroupIDExisted        = NewErrInfo(ErrCodeGroupIDExisted, "GroupID already exists")
	NotInGroupYetError    = NewErrInfo(ErrCodeNotInGroupYetError, "not in the group yet")
	DismissedAlreadyError = NewErrInfo(ErrCodeDismissedAlreadyError, "group has already been dismissed")
	GroupTypeNotSupport   = NewErrInfo(ErrCodeGroupTypeNotSupport, "group type not supported")
	GroupRequestHandled   = NewErrInfo(ErrCodeGroupRequestHandled, "group request has already been handled")

	// Relationship error codes.
	CanNotAddYourselfError   = NewErrInfo(ErrCodeCanNotAddYourselfError, "cannot add yourself as a friend")
	BlockedByPeer            = NewErrInfo(ErrCodeBlockedByPeer, "blocked by the peer")
	NotPeersFriend           = NewErrInfo(ErrCodeNotPeersFriend, "not the peer's friend")
	RelationshipAlreadyError = NewErrInfo(ErrCodeRelationshipAlreadyError, "already in a friend relationship")
	FriendRequestHandled     = NewErrInfo(ErrCodeFriendRequestHandled, "friend request has already been handled")

	// Message error codes.
	MessageHasReadDisable = NewErrInfo(ErrCodeMessageHasReadDisable, "message has been read")
	MutedInGroup          = NewErrInfo(ErrCodeMutedInGroup, "member muted in the group")
	MutedGroup            = NewErrInfo(ErrCodeMutedGroup, "group is muted")
	MessageAlreadyRevoke  = NewErrInfo(ErrCodeMsgAlreadyRevoke, "message already revoked")

	// Token error codes.
	TokenExpiredError     = NewErrInfo(ErrCodeTokenExpiredError, "token expired")
	TokenInvalidError     = NewErrInfo(ErrCodeTokenInvalidError, "token is invalid")
	TokenMalformedError   = NewErrInfo(ErrCodeTokenMalformedError, "token is malformed")
	TokenNotValidYetError = NewErrInfo(ErrCodeTokenNotValidYetError, "token is not valid yet")
	TokenUnknownError     = NewErrInfo(ErrCodeTokenUnknownError, "token is unknown")
	TokenKickedError      = NewErrInfo(ErrCodeTokenKickedError, "token is kicked out")
	TokenNotExistError    = NewErrInfo(ErrCodeTokenNotExistError, "token does not exist")

	// Long connection gateway error codes.
	ConnOverMaxNumLimit    = NewErrInfo(ErrCodeConnOverMaxNumLimit, "connection number exceeds the limit")
	ConnArgsError          = NewErrInfo(ErrCodeConnArgsError, "connection arguments error")
	PushMsgError           = NewErrInfo(ErrCodePushMsgError, "push message error")
	IOSBackgroundPushError = NewErrInfo(ErrCodeIOSBackgroundPushError, "IOS background push error")
	ConnResetError         = NewErrInfo(ErrCodeConnResetError, "connection reset")

	// S3 error codes.
	FileUploadedExpiredError = NewErrInfo(ErrCodeFileUploadedExpiredError, "upload expired")
)
