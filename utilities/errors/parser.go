package custom_errors

import (
	"encoding/json"
	"errors"
	"fmt"

	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/lib/pq"
)

type HashidError struct{}

// SanitizeError strips database-specific prefixes (like "pq: ") from postgres driver errors.
// Custom database errors prefixed with "ERROR " are returned as-is. Unexpected/unhandled database
// errors return a generic "Internal Server Error. Please contact our administrator" to avoid leaking system info.
// Other Go errors are returned as-is.
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}

	var pqErr *pq.Error
	isDbErr := false
	var msg string

	if errors.As(err, &pqErr) {
		isDbErr = true
		msg = pqErr.Message
	} else {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "pq: ") {
			isDbErr = true
			msg = strings.TrimPrefix(errStr, "pq: ")
		}
	}

	if isDbErr {
		if strings.HasPrefix(msg, "ERROR ") {
			return msg
		}
		return "Internal Server Error. Please contact our administrator"
	}

	return err.Error()
}

// IsUnexpectedDBError returns true if the error is a PostgreSQL database error
// that does NOT start with the custom error prefix "ERROR ".
func IsUnexpectedDBError(err error) bool {
	if err == nil {
		return false
	}

	var pqErr *pq.Error
	isDbErr := false
	var msg string

	if errors.As(err, &pqErr) {
		isDbErr = true
		msg = pqErr.Message
	} else {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "pq: ") {
			isDbErr = true
			msg = strings.TrimPrefix(errStr, "pq: ")
		}
	}

	if isDbErr {
		return !strings.HasPrefix(msg, "ERROR ")
	}

	return false
}

func (hashErr *HashidError) Error() string {
	return "[ERR109] Invalid value."
}
func ParseError(errors ...error) map[string][]string {
	var errorList map[string][]string = map[string][]string{}

	for _, err := range errors {
		switch typedError := any(err).(type) {
		case validator.ValidationErrors:
			for _, e := range typedError {
				parseFieldError(e, errorList)
			}
		case *json.UnmarshalTypeError:
			parseUnmarshallingError(typedError, errorList)
		default:
			fmt.Printf("Error type %s\n", typedError)
			_, ok := errorList["others"]
			if ok {
				errorList["others"] = append(errorList["others"], err.Error())
			} else {
				errorList["others"] = []string{err.Error()}
			}
		}

	}

	return errorList
}

func parseFieldError(err validator.FieldError, errorData map[string][]string) {
	var errorMessage string

	switch err.Tag() {
	case "required":
		if err.Field() == "StrParentId" {
			errorMessage = "parentId field is required."
		} else {
			errorMessage = fmt.Sprintf("%s field is required.", err.Field())
		}
	case "lt":
		errorMessage = fmt.Sprintf("%s field must be lower than predetermined value.", err.Field())
	case "lte":
		errorMessage = fmt.Sprintf("%s field must be lower than or equal to predetermined value.", err.Field())
	case "gt":
		errorMessage = fmt.Sprintf("%s field must be greater than predetermined value.", err.Field())
	case "gte":
		errorMessage = fmt.Sprintf("%s field must be greater than or equal to predetermined value.", err.Field())
	case "oneof":
		errorMessage = fmt.Sprintf("Value for field %s does not match the allowed values.", err.Field())
	case "email":
		errorMessage = fmt.Sprintf("Value for field %s must be a valid email.", err.Field())
	case "max":
		errorMessage = "Maximum length exceeded."
	case "min":
		errorMessage = "Minimum length not reached."
	case "alphanum":
		errorMessage = fmt.Sprintf("Value for field %s must be an alphanumeric string.", err.Field())
	case "hashid":
		errorMessage = "Invalid id"
	case "password":
		errorMessage = "Password must contains letters, number, and special characters"
	default:
		errorMessage = fmt.Errorf("%v", err).Error()
	}

	errorField := err.Field()
	if errorField == "StrParentId" {
		errorField = "parentId"
	}
	_, ok := errorData[errorField]
	if ok {
		errorData[errorField] = append(errorData[errorField], errorMessage)
	} else {
		errorData[errorField] = []string{errorMessage}
	}
}

func parseUnmarshallingError(err *json.UnmarshalTypeError, errorData map[string][]string) {
	_, ok := errorData[err.Field]
	if ok {
		errorData[err.Field] = append(errorData[err.Field], fmt.Sprintf("The field %s must be a %s", err.Field, err.Type.String()))
	} else {
		errorData[err.Field] = []string{fmt.Sprintf("The field %s must be a %s", err.Field, err.Type.String())}
	}
}

func DetectOtherError(err error) error {
	if strings.Contains(err.Error(), "mismatch between encode and decode") {
		return &HashidError{}
	}
	return err
}
