package errofy

import (
	"account-service/services/config"
	"fmt"
)

type Api struct {
	ApiErrorCode int64  `json:"api_error_code"`
	ApiErrorDesc string `json:"api_error_desc"`
}

type Logic struct {
	LogicErrorCode int64  `json:"logic_error_code"`
	LogicErrorDesc string `json:"logic_error_desc"`
	ApiError       Api    `json:"api_error"`
}

// map[LogicErrorCode]LogicStruct
var Errors map[int64]Logic

func NewError(LogicErrCode int64, LogicErrDesc string, ApiErrCode int64, ApiErrDesc string) {
	Errors[LogicErrCode] = Logic{
		LogicErrorCode: LogicErrCode,
		LogicErrorDesc: LogicErrDesc,
		ApiError: Api{
			ApiErrorCode: ApiErrCode,
			ApiErrorDesc: ApiErrDesc,
		},
	}
}

var Loki bool

func InitErrors() error {
	//Loki := config
	Loki = config.Data.Loki.Use
	if !Loki {
		return fmt.Errorf("Env variable LokiUse is not set")
	}
	Errors = make(map[int64]Logic)

	// ==========================================
	// STANDARD HTTP STATUS CODES (1xx-5xx)
	// ==========================================

	// 1xx Informational
	NewError(100, "Continue", 100, "Continue")
	NewError(101, "Switching Protocols", 101, "Switching Protocols")
	NewError(102, "Processing", 102, "Processing")

	// 2xx Success
	NewError(200, "OK", 200, "OK")
	NewError(201, "Created", 201, "Created")
	NewError(202, "Accepted", 202, "Accepted")
	NewError(203, "Non-Authoritative Information", 203, "Non-Authoritative Information")
	NewError(204, "No Content", 204, "No Content")
	NewError(205, "Reset Content", 205, "Reset Content")
	NewError(206, "Partial Content", 206, "Partial Content")
	NewError(207, "Multi-Status", 207, "Multi-Status")
	NewError(208, "Already Reported", 208, "Already Reported")
	NewError(226, "IM Used", 226, "IM Used")

	// 3xx Redirection
	NewError(300, "Multiple Choices", 300, "Multiple Choices")
	NewError(301, "Moved Permanently", 301, "Moved Permanently")
	NewError(302, "Found", 302, "Found")
	NewError(303, "See Other", 303, "See Other")
	NewError(304, "Not Modified", 304, "Not Modified")
	NewError(305, "Use Proxy", 305, "Use Proxy")
	NewError(307, "Temporary Redirect", 307, "Temporary Redirect")
	NewError(308, "Permanent Redirect", 308, "Permanent Redirect")

	// 4xx Client Errors
	NewError(400, "Bad Request", 400, "Bad Request")
	NewError(401, "Unauthorized", 401, "Unauthorized")
	NewError(402, "Payment Required", 402, "Payment Required")
	NewError(403, "Forbidden", 403, "Forbidden")
	NewError(404, "Not Found", 404, "Not Found")
	NewError(405, "Method Not Allowed", 405, "Method Not Allowed")
	NewError(406, "Not Acceptable", 406, "Not Acceptable")
	NewError(407, "Proxy Authentication Required", 407, "Proxy Authentication Required")
	NewError(408, "Request Timeout", 408, "Request Timeout")
	NewError(409, "Conflict", 409, "Conflict")
	NewError(410, "Gone", 410, "Gone")
	NewError(411, "Length Required", 411, "Length Required")
	NewError(412, "Precondition Failed", 412, "Precondition Failed")
	NewError(413, "Payload Too Large", 413, "Payload Too Large")
	NewError(414, "URI Too Long", 414, "URI Too Long")
	NewError(415, "Unsupported Media Type", 415, "Unsupported Media Type")
	NewError(416, "Range Not Satisfiable", 416, "Range Not Satisfiable")
	NewError(417, "Expectation Failed", 417, "Expectation Failed")
	NewError(418, "I'm a teapot", 418, "I'm a teapot")
	NewError(421, "Misdirected Request", 421, "Misdirected Request")
	NewError(422, "Unprocessable Entity", 422, "Unprocessable Entity")
	NewError(423, "Locked", 423, "Locked")
	NewError(424, "Failed Dependency", 424, "Failed Dependency")
	NewError(425, "Too Early", 425, "Too Early")
	NewError(426, "Upgrade Required", 426, "Upgrade Required")
	NewError(428, "Precondition Required", 428, "Precondition Required")
	NewError(429, "Too Many Requests", 429, "Too Many Requests")
	NewError(431, "Request Header Fields Too Large", 431, "Request Header Fields Too Large")
	NewError(451, "Unavailable For Legal Reasons", 451, "Unavailable For Legal Reasons")

	// 5xx Server Errors
	NewError(500, "Internal Server Error", 500, "Internal Server Error")
	NewError(501, "Not Implemented", 501, "Not Implemented")
	NewError(502, "Bad Gateway", 502, "Bad Gateway")
	NewError(503, "Service Unavailable", 503, "Service Unavailable")
	NewError(504, "Gateway Timeout", 504, "Gateway Timeout")
	NewError(505, "HTTP Version Not Supported", 505, "HTTP Version Not Supported")
	NewError(506, "Variant Also Negotiates", 506, "Variant Also Negotiates")
	NewError(507, "Insufficient Storage", 507, "Insufficient Storage")
	NewError(508, "Loop Detected", 508, "Loop Detected")
	NewError(510, "Not Extended", 510, "Not Extended")
	NewError(511, "Network Authentication Required", 511, "Network Authentication Required")

	// ==========================================
	// CUSTOM LOGIC ERROR CODES (EXTENSIONS)
	// ==========================================

	// 2xx Success Extensions (200-299)
	NewError(290, "Login successful", 200, "OK")
	NewError(291, "Token refreshed", 200, "OK")
	NewError(292, "Profile updated", 200, "OK")
	NewError(293, "Password changed", 200, "OK")
	NewError(294, "Account verified", 200, "OK")

	// 4xx Client Error Extensions (400-499)
	NewError(490, "Username is required", 400, "Bad Request")
	NewError(491, "Password is required", 400, "Bad Request")
	NewError(492, "Email is required", 400, "Bad Request")
	NewError(493, "Invalid email format", 400, "Bad Request")
	NewError(494, "Password too weak", 400, "Bad Request")
	NewError(495, "Invalid request format", 400, "Bad Request")

	NewError(4041, "User not found", 404, "Not Found")
	NewError(4042, "Resource not found", 404, "Not Found")

	NewError(4091, "User already exists", 409, "Conflict")
	NewError(4092, "Resource already exists", 409, "Conflict")

	NewError(4011, "Invalid credentials", 401, "Unauthorized")
	NewError(4012, "Access token expired", 401, "Unauthorized")
	NewError(4013, "Refresh token expired", 401, "Unauthorized")
	NewError(4014, "Invalid access token", 401, "Unauthorized")
	NewError(4015, "Invalid refresh token", 401, "Unauthorized")
	NewError(4016, "Session expired", 401, "Unauthorized")

	NewError(4031, "User account is blocked", 403, "Forbidden")
	NewError(4032, "Account not verified", 403, "Forbidden")
	NewError(4033, "Insufficient permissions", 403, "Forbidden")

	NewError(4291, "Too many login attempts", 429, "Too Many Requests")
	NewError(4292, "Rate limit exceeded", 429, "Too Many Requests")

	// 5xx Server Error Extensions (500-599)
	NewError(5001, "Database connection failed", 500, "Internal Server Error")
	NewError(5002, "Database query failed", 500, "Internal Server Error")
	NewError(5003, "Database timeout", 500, "Internal Server Error")
	NewError(5004, "Database constraint violation", 500, "Internal Server Error")
	NewError(5005, "Database deadlock detected", 500, "Internal Server Error")

	NewError(5011, "LDAP connection failed", 500, "Internal Server Error")
	NewError(5012, "LDAP authentication failed", 500, "Internal Server Error")
	NewError(5013, "LDAP timeout", 500, "Internal Server Error")
	NewError(5014, "LDAP admin-account wrong credentials", 500, "Internal Server Error")
	NewError(5015, "LDAP Bad Request", 400, "Bad Request")

	NewError(5021, "External service error", 502, "Bad Gateway")
	NewError(5022, "Payment service error", 502, "Bad Gateway")

	NewError(5031, "Cache service unavailable", 503, "Service Unavailable")
	NewError(5032, "Queue service unavailable", 503, "Service Unavailable")

	NewError(5041, "Database timeout", 504, "Gateway Timeout")
	NewError(5042, "External service timeout", 504, "Gateway Timeout")

	return nil
}
