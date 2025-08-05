package email

import "fmt"

// SendWelcomeEmail sends a beautifully formatted welcome email to a new user
func SendWelcomeEmail(toEmail, name string) error {
	subject := "🎉 Welcome to Birdseye Poultry – Your Trial Starts Now!"

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
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Welcome to Birdseye Poultry 🐓</h1>
				</div>
				<div class="content">
					<p>Hi %s,</p>
					<p>We're so glad you're here! 🎉 You've officially started your <strong>30-day free trial</strong> of <strong>Birdseye Poultry Manager</strong>.</p>
					<p>With Birdseye, you can:</p>
					<ul>
						<li>📊 Track egg production and sales</li>
						<li>🧾 Record expenses & income easily</li>
						<li>📅 Monitor flock performance</li>
						<li>📈 View insights from one smart dashboard</li>
					</ul>
					<p>Need help getting started or have questions?</p>
					<p><a href="https://wa.me/254750109154" class="whatsapp">💬 Chat with us on WhatsApp</a></p>
					<a href="https://app.birdseye-poultry.com/auth" class="button">Launch Your Dashboard</a>
					<p style="margin-top: 30px;">We’re rooting for you! 🐣</p>
					<p>Warmly,<br><strong>Kevin</strong><br>Developer, 816 Dynamics</p>
				</div>
				<div class="footer">
					<p>&copy; 2025 Birdseye Poultry. All rights reserved.</p>
					<p>A proud product of <strong>816 Dynamics</strong> – empowering modern agriculture with smart software solutions.</p>
				</div>
			</div>
		</body>
		</html>
	`, name)

	return SendEmail(toEmail, subject, body)
}
