package custom_errors

type CustomError struct {
	Code          uint
	Message       string
	MessageToSend string
	Function      string
	Property      string
}

func (e *CustomError) Error() string {
	return e.Message
}
func (e *CustomError) Compile() {
	if e.MessageToSend == "" {
		e.MessageToSend = "Something went wrong. Please contact our administrator"
	}
}
