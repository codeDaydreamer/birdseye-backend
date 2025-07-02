package email

import "fmt"

// SendBillingEmail sends a billing-related email to the user
func SendBillingEmail(toEmail, name string, amount float64, dueDate string, tillNumber string) error {
	subject := "Birdseye Billing Notice – Action Required"

	body := fmt.Sprintf(`
		<h2>Hi %s,</h2>
		<p>We hope your poultry management is going smoothly.</p>
		<p>This is a reminder that a payment of <strong>Ksh %.2f</strong> is due. To avoid interruption of service, please complete your payment by <strong>%s</strong>.</p>
		<p>You can pay via M-PESA TILL number <strong>%s</strong>.</p>
		<p>If you've already made the payment, kindly ignore this message. Otherwise, feel free to <a href="https://wa.me/254750109154">contact our support team on WhatsApp</a> if you need help.</p>
		<p>Thank you for choosing Birdseye Poultry!</p>
		<p>– The Birdseye Team 🐓</p>
	`, name, amount, dueDate, tillNumber)

	return SendEmail(toEmail, subject, body)
}
