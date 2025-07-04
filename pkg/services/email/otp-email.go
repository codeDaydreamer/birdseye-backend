package email

import "fmt"

// SendOTPEmail sends a formatted OTP verification email to a new user
func SendOTPEmail(toEmail, name, otp string) error {
	subject := "🔐 Your Birdseye Poultry OTP Code"

	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Email Verification</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					background-color: #e0f7fa;
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
					font-size: 22px;
				}
				.content {
					padding: 24px;
					font-size: 16px;
					line-height: 1.6;
				}
				.otp-box {
					background-color: #10b981;
					color: white;
					font-size: 28px;
					font-weight: bold;
					padding: 14px;
					border-radius: 8px;
					text-align: center;
					margin: 20px 0;
					letter-spacing: 4px;
				}
				.footer {
					padding: 20px;
					font-size: 14px;
					text-align: center;
					color: #666;
					background-color: #f1f5f9;
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
					<h1>Email Verification</h1>
				</div>
				<div class="content">
					<p>Hi %s,</p>
					<p>Thanks for signing up to <strong>Birdseye Poultry</strong>.</p>
					<p>Please use the following OTP to verify your email address. It will expire in 15 minutes:</p>
					<div class="otp-box">%s</div>
					<p>Do not share this code with anyone. If you did not sign up, you can safely ignore this email.</p>
					<p>Warm regards,<br><strong>Kevin</strong><br>Developer, 816 Dynamics</p>
				</div>
				<div class="footer">
					<p>&copy; 2025 Birdseye Poultry. All rights reserved.</p>
				</div>
			</div>
		</body>
		</html>
	`, name, otp)

	return SendEmail(toEmail, subject, body)
}
