package email

import (
	"fmt"
	"html"
)

// SendFarmInviteEmail invites someone to join a farm's staff; inviteURL opens the acceptance page
func SendFarmInviteEmail(toEmail, farmName, inviterName, role, inviteURL string) error {
	subject := fmt.Sprintf("You're invited to join %s on Birdseye Poultry", farmName)

	// Farm and inviter names are user-entered, so escape them for HTML
	farmName = html.EscapeString(farmName)
	inviterName = html.EscapeString(inviterName)
	role = html.EscapeString(role)
	inviteURL = html.EscapeString(inviteURL)

	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Farm invitation</title>
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
					<h1>You're invited to %s</h1>
				</div>
				<div class="content">
					<p>Hi there,</p>
					<p><strong>%s</strong> has invited you to join <strong>%s</strong> on <strong>Birdseye Poultry Manager</strong> as a <strong>%s</strong>.</p>
					<p>Accept the invitation to start recording flocks, eggs, sales and expenses with the rest of the team. If you don't have a Birdseye account yet, you can create one on the next page.</p>
					<a href="%s" class="button">Accept Invitation</a>
					<p style="margin-top: 30px; font-size: 14px; color: #555;">This invitation expires in 7 days. If you weren't expecting it, you can ignore this email.</p>
				</div>
				<div class="footer">
					<p>&copy; 2025 Birdseye Poultry. All rights reserved.</p>
					<p>A proud product of <strong>816 Dynamics</strong> – empowering modern agriculture with smart software solutions.</p>
				</div>
			</div>
		</body>
		</html>
	`, farmName, inviterName, farmName, role, inviteURL)

	return SendEmail(toEmail, subject, body)
}
