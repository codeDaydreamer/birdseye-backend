package email

import "fmt"

// SendPasswordResetOTP sends a formatted password reset OTP email to a user
func SendPasswordResetOTP(toEmail, name, otp string) error {
	subject := "🔐 Password Reset OTP - Birdseye Poultry"

	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Password Reset</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					background-color: #fef3c7;
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
					background-color: #dc2626;
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
					background-color: #2563eb;
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
					<h1>Password Reset Request</h1>
				</div>
				<div class="content">
					<p>Hi %s,</p>
					<p>We received a request to reset your <strong>Birdseye Poultry</strong> account password.</p>
					<p>Use the following OTP to complete your password reset. It will expire in 15 minutes:</p>
					<div class="otp-box">%s</div>
					<p>If you didn’t request this change, please ignore this email.</p>
					<p>Stay safe,<br><strong>Kevin</strong><br>Developer, 816 Dynamics</p>
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
