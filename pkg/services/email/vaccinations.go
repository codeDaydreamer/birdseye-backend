package email

import (
	"fmt"
	
	"birdseye-backend/pkg/models"
)

// SendVaccinationReminderEmail sends a nicely formatted vaccination reminder email to the user
func SendVaccinationReminderEmail(toEmail string, userName string, vaccination *models.Vaccination) error {
	subject := "🐓 Birdseye Poultry: Vaccination Reminder"

	vaccinationDate := vaccination.Date.Format("January 2, 2006")

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1" />
		<title>Vaccination Reminder</title>
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
				<h1>Vaccination Reminder 🐓</h1>
			</div>
			<div class="content">
				<p>Hi %s,</p>
				<p>This is a friendly reminder that your flock has an upcoming vaccination scheduled:</p>
				<ul>
					<li><strong>Vaccine:</strong> %s</li>
					<li><strong>Scheduled Date:</strong> %s</li>
				</ul>
				<p>Please make sure to prepare accordingly to keep your flock healthy and thriving.</p>
				<a href="https://app.birdseye-poultry.com/vaccinations" class="button">View Vaccinations</a>
				<p style="margin-top: 30px;">Thank you for trusting Birdseye Poultry Manager!</p>
				<p>Warmly,<br><strong>Kevin</strong><br>Developer, 816 Dynamics</p>
			</div>
			<div class="footer">
				<p>&copy; 2025 Birdseye Poultry. All rights reserved.</p>
				<p>A proud product of <strong>816 Dynamics</strong> – empowering modern agriculture with smart software solutions.</p>
			</div>
		</div>
	</body>
	</html>
	`, userName, vaccination.VaccineName, vaccinationDate)

	return SendEmail(toEmail, subject, body)
}
