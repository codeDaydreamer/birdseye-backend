package email

import "fmt"

// SendWelcomeEmail sends a beautifully formatted welcome email to a new user
func SendWelcomeEmail(toEmail, name string) error {
	subject := "🎉 Welcome to Birdseye Poultry!"

	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Welcome to Birdseye</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					background-color:rgb(170, 245, 255);
					color: #333;
					padding: 0;
					margin: 0;
				}
				.container {
					max-width: 600px;
					margin: 40px auto;
					background: white;
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
					line-height: 1.6;
				}
				.footer {
					padding: 20px;
					font-size: 14px;
					text-align: center;
					color: #666;
					background-color: #f1f5f9;
				}
				a.button {
					display: inline-block;
					margin-top: 20px;
					padding: 10px 20px;
					background-color: #10b981;
					color: white;
					text-decoration: none;
					border-radius: 5px;
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
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Welcome to Birdseye Poultry 🐓</h1>
				</div>
				<div class="content">
					<p>Hi %s,</p>
					<p>Welcome to <strong>Birdseye Poultry</strong> — your smart assistant for managing flocks, tracking egg production, recording expenses, and gaining insights all in one dashboard.</p>
					<p>We’re excited to have you join our growing community of modern farmers.</p>
					<p>If you ever need a hand or have questions, just reply to this email or reach out via WhatsApp:</p>
					<p><a href="https://wa.me/254750109154">+254 750 109 154</a></p>
					<a href="https://app.birdseye-poultry.com/auth" class="button">Get Started</a>
					<p style="margin-top: 30px;">Happy farming! 🐣</p>
					<p>Warm regards,<br><strong>Kevin</strong><br>Developer, 816 Dynamics</p>
				</div>
				<div class="footer">
					<p>&copy; 2025 Birdseye Poultry. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, name)

	return SendEmail(toEmail, subject, body)
}
