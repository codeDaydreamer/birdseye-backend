package email

import "fmt"

func SendWelcomeEmail(toEmail, name string) error {
	subject := "Welcome to Birdseye 🐣"
	body := fmt.Sprintf(`
		<h2>Hello %s,</h2>
		<p>Welcome to Birdseye Poultry Management System!</p>
		<p>We’re thrilled to have you. 🐔</p>
		<p>Start managing your flock efficiently.</p>
		<br/>
		<p>— Birdseye Team</p>
	`, name)

	return sendEmail(toEmail, subject, body)
}
