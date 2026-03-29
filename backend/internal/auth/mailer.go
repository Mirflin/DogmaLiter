package auth

import (
	"fmt"
	"strconv"

	"gopkg.in/gomail.v2"
)

type Mailer struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewMailer(host, port, user, password, from string) *Mailer {
	p, _ := strconv.Atoi(port)
	return &Mailer{
		host:     host,
		port:     p,
		user:     user,
		password: password,
		from:     from,
	}
}

func (m *Mailer) SendVerificationEmail(toEmail, username, verifyURL string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", toEmail)
	msg.SetHeader("Subject", "DogmaLiter — Confirm email")

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; background: #1a1a2e; color: #eee; padding: 40px;">
			<div style="max-width: 500px; margin: 0 auto; background: #16213e; border-radius: 12px; padding: 30px;">
				<h1 style="color: #e94560;">DogmaLiter</h1>
				<p>Hello, <strong>%s</strong>!</p>
				<p>Thank you for registering. Please confirm your email by clicking the button below:</p>
				<div style="text-align: center; margin: 30px 0;">
					<a href="%s" style="background: #e94560; color: white; padding: 14px 28px; text-decoration: none; border-radius: 8px; font-size: 16px;">
						Confirm Email
					</a>
				</div>
				<p style="color: #888; font-size: 12px;">The link is valid for 24 hours. If you did not register, please ignore this email.</p>
			</div>
		</body>
		</html>
	`, username, verifyURL)

	msg.SetBody("text/html", body)

	dialer := gomail.NewDialer(m.host, m.port, m.user, m.password)
	if err := dialer.DialAndSend(msg); err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	return nil
}

func (m *Mailer) SendPasswordResetEmail(toEmail, username, resetURL string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", toEmail)
	msg.SetHeader("Subject", "DogmaLiter — Password Reset")

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; background: #1a1a2e; color: #eee; padding: 40px;">
			<div style="max-width: 500px; margin: 0 auto; background: #16213e; border-radius: 12px; padding: 30px;">
				<h1 style="color: #e94560;">DogmaLiter</h1>
				<p>Hello, <strong>%s</strong>!</p>
				<p>You requested a password reset. Click the button below:</p>
				<div style="text-align: center; margin: 30px 0;">
					<a href="%s" style="background: #e94560; color: white; padding: 14px 28px; text-decoration: none; border-radius: 8px; font-size: 16px;">
						Reset Password
					</a>
				</div>
				<p style="color: #888; font-size: 12px;">The link is valid for 1 hour. If you did not request a password reset, please ignore this email.</p>
			</div>
		</body>
		</html>
	`, username, resetURL)

	msg.SetBody("text/html", body)

	dialer := gomail.NewDialer(m.host, m.port, m.user, m.password)
	if err := dialer.DialAndSend(msg); err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	return nil
}
