package email

import "fmt"

func SendPaymentSuccessEmail(toEmail, name, mpesaRef string, amount float64) error {
	subject := "Payment Received – Birdseye Poultry"

	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
		<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Payment Successful !</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					background-color: #e0f7fa;
					color: #333;
					margin: 0;
					padding: 0;
				}
				.container {
					max-width: 600px;
					margin: 40px auto;
					background: #fff;
					border-radius: 10px;
					box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
					overflow: hidden;
				}
				.header {
					background-color: #2563eb;
					color: white;
					padding: 24px;
					text-align: center;
				}
				.header h1 {
					margin: 0;
					font-size: 24px;
				}
				.content {
					padding: 24px;
					font-size: 16px;
					line-height: 1.7;
				}
				.footer {
					padding: 20px;
					font-size: 13px;
					text-align: center;
					color: #555;
					background-color: #f1f5f9;
				}
				a.button {
					display: inline-block;
					margin-top: 20px;
					padding: 12px 24px;
					background-color: #10b981;
					color: white;
					text-decoration: none;
					border-radius: 6px;
					font-weight: bold;
				}
				a.whatsapp {
					display: inline-block;
					margin-top: 15px;
					color: #25D366;
					text-decoration: none;
					font-weight: bold;
				}
				@media (max-width: 600px) {
					.container {
						border-radius: 0;
					}
					.header, .content, .footer {
						padding: 16px;
					}
				}
			</style></head> 
		<body>
			<div class="container">
				<div class="header">
					<h1>Payment Confirmed ✅</h1>
				</div>
				<div class="content">
					<p>Hi %s,</p>
					<p>We've successfully received your payment of <strong>KES %.0f</strong> via M-PESA. 🎉</p>
					<p><strong>Receipt Number:</strong> %s</p>
					<p>Your Birdseye account is now fully active. You can continue using all features without interruption.</p>
					<p>Need help or a receipt?</p>
					<p><a href="https://wa.me/254750109154" class="whatsapp">💬 Contact us on WhatsApp</a></p>
					<a href="https://app.birdseye-poultry.com" class="button">Open My Dashboard</a>
					<p style="margin-top: 30px;">Thanks for choosing Birdseye Poultry!</p>
					<p>Best regards,<br><strong>Kevin</strong><br>Developer, 816 Dynamics</p>
				</div>
				<div class="footer">
					<p>&copy; 2025 Birdseye Poultry. All rights reserved.</p>
					<p>A product of <strong>816 Dynamics</strong>.</p>
				</div>
			</div>
		</body>
		</html>
	`, name, amount, mpesaRef)

	return SendEmail(toEmail, subject, body)
}
func SendPaymentFailureEmail(toEmail, name string, amount float64, reason string) error {
	subject := "Payment Failed – Birdseye Poultry"

	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
		<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Payment Failure</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					background-color: #e0f7fa;
					color: #333;
					margin: 0;
					padding: 0;
				}
				.container {
					max-width: 600px;
					margin: 40px auto;
					background: #fff;
					border-radius: 10px;
					box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
					overflow: hidden;
				}
				.header {
					background-color: #2563eb;
					color: white;
					padding: 24px;
					text-align: center;
				}
				.header h1 {
					margin: 0;
					font-size: 24px;
				}
				.content {
					padding: 24px;
					font-size: 16px;
					line-height: 1.7;
				}
				.footer {
					padding: 20px;
					font-size: 13px;
					text-align: center;
					color: #555;
					background-color: #f1f5f9;
				}
				a.button {
					display: inline-block;
					margin-top: 20px;
					padding: 12px 24px;
					background-color: #10b981;
					color: white;
					text-decoration: none;
					border-radius: 6px;
					font-weight: bold;
				}
				a.whatsapp {
					display: inline-block;
					margin-top: 15px;
					color: #25D366;
					text-decoration: none;
					font-weight: bold;
				}
				@media (max-width: 600px) {
					.container {
						border-radius: 0;
					}
					.header, .content, .footer {
						padding: 16px;
					}
				}
			</style></head>
		<body>
			<div class="container">
				<div class="header" style="background-color:#ef4444;">
					<h1>Payment Failed ❌</h1>
				</div>
				<div class="content">
					<p>Hi %s,</p>
					<p>We attempted to process your payment of <strong>KES %.0f</strong> via M-PESA, but it was not successful.</p>
					<p><strong>Reason:</strong> %s</p>
					<p>Please ensure your M-PESA number is correct and has sufficient funds, then try again.</p>
					<p>If the issue persists, reach out for help:</p>
					<p><a href="https://wa.me/254750109154" class="whatsapp">💬 Chat with us on WhatsApp</a></p>
					<p>We’re here to help you stay on track with your farm.</p>
					<p>Warm regards,<br><strong>Kevin</strong><br>Developer, 816 Dynamics</p>
				</div>
				<div class="footer">
					<p>&copy; 2025 Birdseye Poultry. All rights reserved.</p>
					<p>A product of <strong>816 Dynamics</strong>.</p>
				</div>
			</div>
		</body>
		</html>
	`, name, amount, reason)

	return SendEmail(toEmail, subject, body)
}
func SendInvoiceEmail(toEmail, name string, amount float64, reference string) error {
	subject := fmt.Sprintf("🧾 Invoice – KES %.0f Payment Due", amount)
	body := fmt.Sprintf(`<!DOCTYPE html>
	<!-- Similar style to welcome email... -->
	<body>
		<p>Hi %s,</p>
		<p>We’ve generated your invoice for this month’s subscription. Your reference is <strong>%s</strong>.</p>
		<p><strong>Total:</strong> KES %.0f</p>
		<p>Please complete payment to continue enjoying premium access.</p>
		<a href="https://app.birdseye-poultry.com/billing" class="button">Pay Now</a>
	</body>
	</html>`, name, reference, amount)

	return SendEmail(toEmail, subject, body)
}
func SendPaymentReminderEmail(toEmail, name string) error {
	subject := "📢 Reminder: Your Subscription Payment is Due"
	body := fmt.Sprintf(`
	<body>
		<p>Hi %s,</p>
		<p>We noticed your subscription is overdue. To keep your account active, please renew your payment.</p>
		<p>We’ve also sent your invoice. Let us know if you need help.</p>
		<a href="https://app.birdseye-poultry.com/billing" class="button">Renew Now</a>
	</body>
	</html>`, name)

	return SendEmail(toEmail, subject, body)
}
