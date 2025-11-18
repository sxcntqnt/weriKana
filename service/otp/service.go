// service/otp/service.go
package otp

var instance = New()

func GetService() *Service {
	return instance
}
