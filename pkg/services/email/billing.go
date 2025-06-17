package email

import "fmt"

func SendInvoiceEmail(toEmail, name, invoiceURL string) error {
	subject := "Your Birdseye Invoice is Ready"
	body := fmt.Sprintf(`
		<h2>Hi %s,</h2>
		<p>Thank you for using Birdseye.</p>
		<p>Your invoice is available here: <a href="%s">View Invoice</a></p>
		<p>If you have any questions, feel free to reach out.</p>
	`, name, invoiceURL)

	return sendEmail(toEmail, subject, body)
}
