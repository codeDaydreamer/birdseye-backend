package email

import (
	"fmt"
	"strings"

	"birdseye-backend/pkg/models"
)

// Every Birdseye email. Names, farm names and reasons are user-entered or come from M-Pesa, so they go
// through Esc before landing in a line of HTML.

const whatsApp = `<a href="https://wa.me/254792803063" style="color:#1a6b1b;font-weight:600;">WhatsApp us on 0792 803 063</a>`

func greeting(name string) string {
	if f := strings.Fields(name); len(f) > 0 {
		return "Hi " + f[0] + ","
	}
	return "Hello,"
}

func kes(amount float64) string {
	s := fmt.Sprintf("%.0f", amount)
	for i := len(s) - 3; i > 0 && s[i-1] != '-'; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return "KES " + s
}

// SendOTPEmail sends the code that confirms a new account's email address
func SendOTPEmail(toEmail, name, otp string) error {
	return send(toEmail, "Your Birdseye verification code: "+otp, Message{
		Preview:  "Use this code to confirm your email address. It expires in 15 minutes.",
		Title:    "Confirm your email",
		Greeting: greeting(name),
		Lines:    []string{"Thanks for signing up to Birdseye Poultry. Enter this code in the app to confirm your email address:"},
		Code:     otp,
		Note:     "The code expires in 15 minutes. Don't share it with anyone. If you didn't sign up, you can ignore this email.",
	})
}

// SendPasswordResetOTP sends the code that lets someone choose a new password
func SendPasswordResetOTP(toEmail, name, otp string) error {
	return send(toEmail, "Your Birdseye password reset code: "+otp, Message{
		Preview:  "Use this code to choose a new password. It expires in 15 minutes.",
		Title:    "Reset your password",
		Greeting: greeting(name),
		Lines:    []string{"We received a request to reset your Birdseye Poultry password. Enter this code in the app to choose a new one:"},
		Code:     otp,
		Note:     "The code expires in 15 minutes. If you didn't ask for this, ignore this email: your password stays the same.",
	})
}

// SendWelcomeEmail greets a new user once their account is ready
func SendWelcomeEmail(toEmail, name string) error {
	subject := "Welcome to Birdseye Poultry"
	first := "Your account is ready."
	if settings, err := models.GetPlatformSettings(); err == nil && settings.TrialEnabled && settings.TrialDays > 0 {
		first = fmt.Sprintf("Your account is ready, and your <strong>%d-day free trial</strong> has started.", settings.TrialDays)
	}
	return send(toEmail, subject, Message{
		Preview:  "Your Birdseye account is ready.",
		Title:    "Welcome to Birdseye Poultry",
		Greeting: greeting(name),
		Lines: []string{
			first,
			"Birdseye keeps your whole farm in one place: batches and bird counts, eggs, feed and FCR, vaccinations, sales and costs, with reports you can share.",
			"A good first step is to add your first batch, then invite the people who work on the farm so they can record from their phones.",
		},
		Button: &Button{Label: "Open Birdseye", URL: appURL},
		Note:   "Need a hand getting set up? " + whatsApp + ".",
	})
}

// SendFarmInviteEmail invites someone to join a farm's staff; inviteURL opens the acceptance page
func SendFarmInviteEmail(toEmail, farmName, inviterName, role, inviteURL string) error {
	return send(toEmail, fmt.Sprintf("You're invited to join %s on Birdseye Poultry", farmName), Message{
		Preview: fmt.Sprintf("%s invited you to join %s as %s.", inviterName, farmName, role),
		Title:   "You're invited to " + farmName,
		Lines: []string{
			fmt.Sprintf("<strong>%s</strong> invited you to join <strong>%s</strong> on Birdseye Poultry as a <strong>%s</strong>.", Esc(inviterName), Esc(farmName), Esc(role)),
			"Accept the invitation to start recording batches, eggs, feed, sales and costs with the rest of the team. If you don't have a Birdseye account yet, you can create one on the next page.",
		},
		Button: &Button{Label: "Accept invitation", URL: inviteURL},
		Note:   "This invitation expires in 7 days. If you weren't expecting it, you can ignore this email.",
	})
}

// SendVaccinationReminderEmail reminds a farm about an upcoming vaccination
func SendVaccinationReminderEmail(toEmail string, userName string, vaccination *models.Vaccination) error {
	return send(toEmail, "Vaccination due: "+vaccination.VaccineName, Message{
		Preview:  fmt.Sprintf("%s is due on %s.", vaccination.VaccineName, vaccination.Date.Format("2 January 2006")),
		Title:    "Vaccination reminder",
		Greeting: greeting(userName),
		Lines:    []string{"Your flock has a vaccination coming up:"},
		Details: []Detail{
			{Label: "Vaccine", Value: vaccination.VaccineName},
			{Label: "Date", Value: vaccination.Date.Format("Monday, 2 January 2006")},
		},
		Button: &Button{Label: "View vaccinations", URL: appURL + "/vaccinations"},
		Note:   "Once it's done, mark it as given in Birdseye so the schedule stays up to date.",
	})
}

// SendPaymentSuccessEmail confirms a subscription payment
func SendPaymentSuccessEmail(toEmail, name, mpesaRef string, amount float64) error {
	return send(toEmail, "Payment received: "+kes(amount), Message{
		Preview:  fmt.Sprintf("We received %s by M-Pesa. Thank you.", kes(amount)),
		Title:    "Payment received",
		Greeting: greeting(name),
		Lines:    []string{"Thank you. We've received your M-Pesa payment and your Birdseye subscription is active."},
		Details: []Detail{
			{Label: "Amount", Value: kes(amount)},
			{Label: "M-Pesa receipt", Value: mpesaRef},
		},
		Button: &Button{Label: "Open Birdseye", URL: appURL},
		Note:   "Keep this email as your receipt. Questions about billing? " + whatsApp + ".",
	})
}

// SendPaymentFailureEmail tells a payer their M-Pesa payment didn't go through
func SendPaymentFailureEmail(toEmail, name string, amount float64, reason string) error {
	lines := []string{fmt.Sprintf("Your M-Pesa payment of <strong>%s</strong> didn't go through, so nothing was charged.", kes(amount))}
	if strings.TrimSpace(reason) != "" {
		lines = append(lines, "M-Pesa said: "+Esc(reason))
	}
	lines = append(lines, "Check that the phone has enough M-Pesa balance and that you entered the PIN in time, then try again from Billing.")
	return send(toEmail, "Payment didn't go through", Message{
		Preview:  "Your M-Pesa payment wasn't completed. You can try again from Billing.",
		Title:    "Payment didn't go through",
		Greeting: greeting(name),
		Lines:    lines,
		Button:   &Button{Label: "Try again", URL: appURL + "/billing"},
		Note:     "Still stuck? " + whatsApp + ".",
		Danger:   true,
	})
}

// SendBillingEmail reminds a farm that a payment is due
func SendBillingEmail(toEmail, name string, amount float64, dueDate string, tillNumber string) error {
	details := []Detail{{Label: "Amount due", Value: kes(amount)}, {Label: "Due by", Value: dueDate}}
	if tillNumber != "" {
		details = append(details, Detail{Label: "M-Pesa till", Value: tillNumber})
	}
	return send(toEmail, "Your Birdseye payment is due", Message{
		Preview:  fmt.Sprintf("%s is due by %s.", kes(amount), dueDate),
		Title:    "Payment due",
		Greeting: greeting(name),
		Lines:    []string{"Your next Birdseye subscription payment is coming up. Pay before the due date to keep your farm's records running without a break."},
		Details:  details,
		Button:   &Button{Label: "Pay now", URL: appURL + "/billing"},
		Note:     "Already paid? You can ignore this email.",
	})
}

// SendInvoiceEmail sends a subscription invoice
func SendInvoiceEmail(toEmail, name string, amount float64, reference string) error {
	return send(toEmail, "Your Birdseye invoice: "+kes(amount), Message{
		Preview:  fmt.Sprintf("Invoice %s for %s.", reference, kes(amount)),
		Title:    "Your invoice",
		Greeting: greeting(name),
		Lines:    []string{"Here is your invoice for this month's Birdseye subscription."},
		Details:  []Detail{{Label: "Reference", Value: reference}, {Label: "Total", Value: kes(amount)}},
		Button:   &Button{Label: "Pay now", URL: appURL + "/billing"},
	})
}

// SendPaymentReminderEmail reminds a farm whose subscription is overdue
func SendPaymentReminderEmail(toEmail, name string) error {
	return send(toEmail, "Your Birdseye subscription is overdue", Message{
		Preview:  "Renew to keep using Birdseye without a break.",
		Title:    "Your subscription is overdue",
		Greeting: greeting(name),
		Lines:    []string{"Your Birdseye subscription payment is overdue. Renew now to keep recording and to keep your reports up to date."},
		Button:   &Button{Label: "Renew now", URL: appURL + "/billing"},
		Note:     "Need more time or have a question? " + whatsApp + ".",
	})
}
