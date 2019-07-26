package utils

func GetRecovery(method string, queueName string, message string) string {
	return method + " " + queueName + " " + message
}